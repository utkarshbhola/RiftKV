package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func encodeRESP(msg string) string {
	parts := strings.Fields(msg)

	result := "*" + fmt.Sprint(len(parts)) + "\r\n"

	for _, part := range parts {
		result += "$" + fmt.Sprint(len(part)) + "\r\n"
		result += part + "\r\n"
	}

	return result
}

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:6380")

	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}

	keyboard := bufio.NewReader(os.Stdin)
	reader := bufio.NewReader(conn)

	for {
		msg, _ := keyboard.ReadString('\n')

		resp := encodeRESP(msg)

		conn.Write([]byte(resp))

		response, _ := reader.ReadString('\n')

		fmt.Print("Response: ", response)
	}
}
