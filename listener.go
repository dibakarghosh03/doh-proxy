package main

import (
	"fmt"
	"log"
	"net"
)

var cache = NewDNSCache()

func startUDPListener(addr string) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	log.Printf("listening on %s", addr)

	for {
		buf := make([]byte, 512)
		n, clientAddr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		go handleQuery(conn, clientAddr, buf[:n])
	}
}

func handleQuery(conn net.PacketConn, addr net.Addr, buf []byte) {
	msg, err := ParseDNSMessage(buf)
	if err != nil {
		log.Printf("parse error: %v", err)
		return
	}

	if len(msg.Questions) == 0 {
		return
	}

	if len(msg.Questions) > 0 {
		log.Printf("query: %s type=%d", msg.Questions[0].Name, msg.Questions[0].Type)
	}

	response, found := cache.Get(msg.Questions[0].Name, msg.Questions[0].Type)
	if found {
		patchResponseID(response, msg.Header.ID)
		_, err = conn.WriteTo(response, addr)
		if err != nil {
			log.Printf("write error: %v", err)
		}

		return
	} else {
		response, err := forwardToDoH(buf)
		if err != nil {
			log.Printf("doh error: %v", err)
			return
		}

		message, err := ParseDNSMessage(response)
		if err != nil {
			log.Printf("error: %v", err)
		}

		cache.Set(msg.Questions[0].Name, msg.Questions[0].Type, response, message.Answers)

		patchResponseID(response, msg.Header.ID)
		_, err = conn.WriteTo(response, addr)
		if err != nil {
			log.Printf("write error: %v", err)
		}
	}
}
