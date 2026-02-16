package main

import (
	"fmt"
	"log"
	"net"

	"tcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("error", "error", err)
	}

	fmt.Println("TCP server listening on :42069")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		fmt.Println("client connected:", conn.RemoteAddr())

		r, err := request.RequestFromReader(conn)
		if err != nil {
			log.Fatal("error", "error", err)
		}
		fmt.Printf("Request line:\n")
		fmt.Printf("- Method: %s\n", r.RequestLine.Method)
		fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)
	}
}
