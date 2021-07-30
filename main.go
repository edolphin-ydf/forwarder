package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

var (
	portsStr = flag.String("ports", "", "")
	server = flag.String("server", "", "")
	authKey = flag.String("auth", "", "")
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if ak, exist := r.Header["Authorization"]; !exist {
		log.Println("1")
		return
	} else if len(ak) == 0 {
		log.Println("2")
		return
	} else if ak[0] != *authKey {
		log.Println("3")
		return
	}

	if r.URL.Path != "/updatesrv" {
		log.Println("4")
		return
	}

	if srv, exist := r.URL.Query()["srv"]; !exist {
		log.Println("5")
		return
	} else if len(srv) == 0 {
		log.Println("6")
		return	
	} else {
		log.Println("set new server:", srv[0])
		*server = srv[0]
		return
	}
}

func main() {
	flag.Parse()

	var ports = strings.Split(*portsStr, ",")

	log.Println(ports)
	for _, port := range ports {
		l, err := net.Listen("tcp", "0.0.0.0:" + port)
		if err != nil {
			log.Fatal(err)
		}

		go func(port string) {
			for {
				conn, err := l.Accept()
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println("new connection from:", conn.RemoteAddr().String())

				go func(port string) {
					srvCon, err := net.Dial("tcp", *server + ":" + port)
					log.Println("dialed to server:", *server, "port:", port)
					if err != nil {
						log.Println(err)
						return
					}
					go func() {
						if _, err := io.Copy(srvCon, conn); err != nil {
							log.Println(err)
						}
					}()
					go func() {
						if _, err := io.Copy(conn, srvCon); err != nil {
							log.Println(err)
						}
					}()
				}(port)
			}
		}(port)
	}

	if err := http.ListenAndServe("0.0.0.0:19999", http.HandlerFunc(Handler)); err != nil {
		log.Fatal(err)
	}
}

