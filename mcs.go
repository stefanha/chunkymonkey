package main

import (
	"net"
	"log"
	"fmt"
)

func StartSession(conn net.Conn) {
	username, e := ReadHandshake(conn)
	if e != nil {
		panic(fmt.Sprint("ReadHandshake: ", e.String()))
	}
	log.Stderr("username: ", username)
	WriteHandshake(conn, "-")

	_, _, e2 := ReadLogin(conn)
	if e2 != nil {
		panic(fmt.Sprint("ReadLogin: ", e2.String()))
	}
	WriteLogin(conn)
}

func ServeSession(conn net.Conn) {
	log.Stderr("Client connected from ", conn.RemoteAddr())

	defer func() {
		if err := recover(); err != nil {
			log.Stderr(err)
		}
		log.Stderr("Client disconnected from ", conn.RemoteAddr())
		conn.Close()
	}()

	StartSession(conn)
}

func Serve(addr string) {
	listener, e := net.Listen("tcp", addr)
	if e != nil {
		log.Exit("Listen: ", e.String())
	}
	log.Stderr("Listening on ", addr)

	for {
		conn, e2 := listener.Accept()
		if e2 != nil {
			log.Stderr("Accept: ", e2.String())
			continue
		}

		go ServeSession(conn)
	}
}

func main() {
	Serve(":25565")
}
