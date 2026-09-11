package main

import (
	"bufio"
	"fmt"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		value, err := readRESP(reader)
		if err != nil {
			return
		}

		fmt.Printf("Received: %#v\n", value)

		conn.Write([]byte("+OK\r\n"))
	}
}

func main() {
	listener, _ := net.Listen("tcp", "127.0.0.1:6380")

	fmt.Println("RiftKV server started on 6380")

	for {
		conn, _ := listener.Accept()

		go handleConnection(conn)
	}
}
