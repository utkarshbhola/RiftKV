package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
	"sync"
)

type Entry struct {
	value string
	expiration time.Time
}

var(
	KV = make(map[string]Entry)
	mu sync.RWMutex
)
func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		value, err := readRESP(reader)
		if err != nil {
			return
		}

		command := value.([]interface{})

		fmt.Printf("Received: %#v\n", command)

		if command[0] == "SET" {
			key := command[1].(string)
			value := command[2].(string)

			mu.Lock()
			KV[key] = Entry{value: value}
			mu.Unlock()

			conn.Write([]byte("+OK\r\n"))
		}

		if command[0] == "GET" {
			key := command[1].(string)

			mu.RLock()
			value, ok := KV[key]
			mu.RUnlock()

			if ok {
				response := "$" + fmt.Sprint(len(value.value)) + "\r\n" + value.value + "\r\n"
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("$-1\r\n"))
			}
		}

		if command[0] == "DEL" {
			key := command[1].(string)

			_, ok := KV[key]

			if ok {
				mu.Lock()
				delete(KV, key)
				mu.Unlock()
				conn.Write([]byte(":1\r\n"))
			} else {
				conn.Write([]byte(":0\r\n"))
			}
		}
		if command[0] == "PING" {
			conn.Write([]byte("+PONG\r\n"))
		}
		if command[0] == "EXISTS" {
			mu.RLock()
			_, ok := KV[command[1].(string)]
			mu.RUnlock()

			if ok {
				conn.Write([]byte(":1\r\n"))
			} else {
				conn.Write([]byte(":0\r\n"))
			}
		}
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
