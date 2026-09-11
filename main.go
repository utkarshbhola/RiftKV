package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	listener, _ := net.Listen("tcp", "127.0.0.1:6380")
	fmt.Println("server started")

	for {
		conn, _ := listener.Accept()

		go func() {
			reader := bufio.NewReader(conn)

			msg, _ := reader.ReadString('\n')
			fmt.Print("Received: ", msg)

			writer := bufio.NewWriter(conn)
			writer.WriteString(msg)
			writer.Flush()

			conn.Close()
		}()
	}
}
