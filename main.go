package main

import (
	"log"
	"time"
)

func main() {
	cache.StartCleanup(time.Minute)
	log.Fatal(startUDPListener("127.0.0.1:5353"))
}
