package main

import "log"

func main() {
	srv := New()
	if err := srv.Start("127.0.0.1:7570"); err != nil {
		log.Fatal(err)
	}
}
