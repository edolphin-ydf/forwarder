//
// main.go
// Copyright (C) 2021 edolphin <dngfngyang@gmail.com>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/xtaci/smux"
)

var (
	server = flag.String("server", "", "")
)

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	for { // for retry
		conn, err := net.Dial("tcp", *server)
		if err != nil {
			log.Println("E:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("connected to server:", *server)

		session, err := smux.Server(conn, nil)
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}

		for {
			stream, err := session.AcceptStream()
			if err != nil {
				log.Println("E:", err)
				break
			}


			go handleConn(stream)
		}
	}
}

func handleConn(c net.Conn) {
	var buf [4]byte
	io.ReadFull(c, buf[:])
	port := binary.LittleEndian.Uint32(buf[:])
	dstConn, err := net.Dial("tcp", "127.0.0.1:" + strconv.Itoa(int(port)))
	if err != nil {
		log.Println("E:", err)
		c.Close()
		return
	}

	log.Println("D: new stream from:", c.RemoteAddr().String(), "port:", port)

	closeOnce := sync.Once{}
	go func() {
		if _, err := io.Copy(dstConn, c); err != nil {
			log.Println("E:", err)
		}
		closeOnce.Do(func() {
			dstConn.Close()
			c.Close()
		})
	}()
	go func() {
		if _, err := io.Copy(c, dstConn); err != nil {
			log.Println("E:", err)
		}
		closeOnce.Do(func() {
			dstConn.Close()
			c.Close()
		})
	}()
}



