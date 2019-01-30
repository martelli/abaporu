package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"time"
)

var (
	serverPort = flag.Int("port", 69, "Set tftp server UDP port")
	rootDir    = flag.String("rootdir", "/tmp", "Set tftp root directory.")
	timeout    = flag.Int("timeout", 5, "Packet transmission timeout.")
	retries    = flag.Int("retries", 5, "Packet transmission retries.")
)

func main() {
	flag.Parse()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: *serverPort})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Server started with root: %s\n", *rootDir)

	mainBuffer := make([]byte, 2048)

	servers := make(map[string]*Server, 0)

	for {
		n, addr, err := conn.ReadFrom(mainBuffer)

		if err != nil {
			fmt.Println(err)
			continue
		}
		server, ok := servers[addr.String()]
		if ok {
			server.Buffer.Write(mainBuffer[:n])
			server.Notify()
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Printf("New connection from: %s\n", addr)
			buf := new(bytes.Buffer)
			buf.Write(mainBuffer[:n])
			s := NewServer(addr, conn, buf, *rootDir, time.Duration(*timeout)*time.Second, int64(*retries))

			servers[addr.String()] = s
			go s.Serve()
		}
	}

}
