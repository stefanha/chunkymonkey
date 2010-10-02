package main

import (
	"fmt"
	"log"
	"net"
)

type XYZ struct {
	x, y, z float64
}

type Orientation struct {
	rotation float32
	pitch float32
}

type Game struct {
	chunkManager *ChunkManager
}

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

	WriteSpawnPosition(conn, &XYZ{0, 64, 0})
	WritePlayerInventory(conn)
	WritePlayerPositionLook(conn, &XYZ{0, 64, 0}, &Orientation{0, 0},
	                        0, false)
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

func Serve(addr string, game *Game) {
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
