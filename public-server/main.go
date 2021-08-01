package main

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/xtaci/smux"
)

var (
	portsStr = flag.String("ports", "", "")
	serverFlag = flag.String("server", "", "")
	authKey = flag.String("auth", "", "")
	port = flag.String("port", "", "")

	server = atomic.Value{}
	session = atomic.Value{}
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
		log.Println("I: set new server:", srv[0])
		server.Store(srv[0])
		return
	}
}

func main() {
	flag.Parse()
	server.Store(*serverFlag)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	l, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("E:", err)
				continue
			}
			remoteAddr := strings.Split(conn.RemoteAddr().String(), ":")
			srv := server.Load().(string)
			if remoteAddr[0] != srv {
				log.Println("W: remote addr:", conn.RemoteAddr().String(), "server addr:", srv, "droped!")
				continue
			}

			if s, err := smux.Client(conn, nil); err != nil {
				log.Panicln("E:", err)
				continue
			} else {
				session.Store(s)
			}
		}
	}()

	var ports = strings.Split(*portsStr, ",")

	log.Println("D:", ports)
	for _, port := range ports {
		listenOnPort(port)
	}

	if err := http.ListenAndServe("0.0.0.0:19999", http.HandlerFunc(Handler)); err != nil {
		log.Fatal(err)
	}
}

func listenOnPort(port string) {
	go func(port string) {
		l, err := net.Listen("tcp", "0.0.0.0:" + port)
		if err != nil {
			log.Fatal(err)
		}

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("E:", err)
				continue
			}
			log.Println("I: new connection from:", conn.RemoteAddr().String())

			go func(port string) {
				s := session.Load().(*smux.Session)
				stream, err := s.OpenStream()
				if err != nil {
					log.Println("E:", err)
					return
				}
				srvCon := stream
				log.Println("I: opened new stream to server:", s.RemoteAddr().String(), "port:", port)

				// write port first
				var buf [4]byte
				p, _ := strconv.Atoi(port)
				binary.LittleEndian.PutUint32(buf[:], uint32(p))
				srvCon.Write(buf[:])

				// then copy data
				go func() {
					if _, err := io.Copy(conn, srvCon); err != nil {
						log.Println("E:", err)
					}
				}()
				if _, err := io.Copy(srvCon, conn); err != nil {
					log.Println("E:", err)
				}
			}(port)
		}
	}(port)
}

