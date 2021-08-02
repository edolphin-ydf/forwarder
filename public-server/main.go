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
	listen= flag.String("listen", "", "")
	apiListen = flag.String("listenapi", "", "")

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

	l, err := net.Listen("tcp", *listen)
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
				log.Println("E:", err)
				continue
			} else {
				session.Store(s)
			}
		}
	}()

	var ports = strings.Split(*portsStr, ",")

	log.Println("D:", ports)
	// simple verify port is not empty
	for _, port := range ports {
		if port == "" {
			log.Fatal("invalid port")
		}
	}
	for _, port := range ports {
		go listenOnPort(port)
	}

	if err := http.ListenAndServe(*apiListen, http.HandlerFunc(Handler)); err != nil {
		log.Fatal(err)
	}
}

func listenOnPort(port string) {
	p := strings.Split(port, ":")
	l, err := net.Listen("tcp", "0.0.0.0:" + p[0])
	if err != nil {
		log.Fatal(err)
	}

	dstPort := p[0]
	if len(p) >= 2 {
		dstPort = p[1]
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("E:", err)
			continue
		}
		log.Println("I: new connection from:", conn.RemoteAddr().String())

		go func(port string) {
			s, ok := session.Load().(*smux.Session)
			if !ok {
				log.Println("I: session not ready, do nothing")
				return
			}
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
					conn.Close()
					srvCon.Close()
				}
			}()
			if _, err := io.Copy(srvCon, conn); err != nil {
				log.Println("E:", err)
				conn.Close()
				srvCon.Close()
			}
		}(dstPort)
	}
}

