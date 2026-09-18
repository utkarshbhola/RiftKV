package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func handleConnection(conn net.Conn, store *Store) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		value, err := readRESP(reader)
		if err != nil {
			return
		}

		command, ok := value.([]interface{})
		if !ok {
			conn.Write([]byte("-ERR invalid command\r\n"))
			return
		}

		if len(command) == 0 {
			continue
		}

		switch command[0] {
		case "SET":
			if len(command) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for SET\r\n"))
				continue
			}

			key := command[1].(string)
			data := command[2].(string)
			ttl := time.Duration(0)

			if len(command) >= 5 {
				if strings.EqualFold(command[3].(string), "EX") {
					seconds, err := strconv.Atoi(command[4].(string))
					if err != nil {
						conn.Write([]byte("-ERR invalid TTL\r\n"))
						continue
					}
					ttl = time.Duration(seconds) * time.Second
				}
			}

			if err := store.Set(key, data, ttl); err != nil {
				conn.Write([]byte("-ERR failed to persist value\r\n"))
				continue
			}
			conn.Write([]byte("+OK\r\n"))

		case "GET":
			if len(command) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for GET\r\n"))
				continue
			}

			key := command[1].(string)
			value, ok := store.Get(key)
			if ok {
				response := "$" + fmt.Sprint(len(value)) + "\r\n" + value + "\r\n"
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("$-1\r\n"))
			}

		case "DEL":
			if len(command) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for DEL\r\n"))
				continue
			}

			key := command[1].(string)
			if _, ok := store.Get(key); ok {
				if err := store.Delete(key); err != nil {
					conn.Write([]byte("-ERR failed to delete key\r\n"))
					continue
				}
				conn.Write([]byte(":1\r\n"))
			} else {
				conn.Write([]byte(":0\r\n"))
			}

		case "PING":
			conn.Write([]byte("+PONG\r\n"))

		case "EXISTS":
			if len(command) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for EXISTS\r\n"))
				continue
			}

			if store.Exists(command[1].(string)) {
				conn.Write([]byte(":1\r\n"))
			} else {
				conn.Write([]byte(":0\r\n"))
			}

		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

func main() {
	store, err := NewStore("riftkv.wal")
	if err != nil {
		fmt.Println("failed to initialize store:", err)
		return
	}
	defer store.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:6380")
	if err != nil {
		fmt.Println("failed to start server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("RiftKV server started on 6380")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleConnection(conn, store)
	}
}
