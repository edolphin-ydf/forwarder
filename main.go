package main

import (
	"flag"
	"io"
	"log"
	"net"
	"strings"
)

var (
	portsStr = flag.String("ports", "", "")
	server = flag.String("server", "", "")
)

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

	ch := make(chan struct{}, 1)
	<-ch
}
