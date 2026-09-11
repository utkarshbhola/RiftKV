package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, _ := net.Dial("tcp", "127.0.0.1:6380")

	keyboard := bufio.NewReader(os.Stdin)
	reader := bufio.NewReader(conn)

	for {
		msg, _ := keyboard.ReadString('\n')

		conn.Write([]byte(msg))

		response, _ := reader.ReadString('\n')

		fmt.Print("Response: ", response)
	}
}
