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

type Player struct {
	position XYZ
	orientation Orientation
}

type Game struct {
	chunkManager *ChunkManager
	mainQueue chan func(*Game)
}

func startSession(conn net.Conn) (player *Player) {
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

	player = &Player {
		position: XYZ{0, 64, 0},
		orientation: Orientation{0, 0},
	}

	WriteSpawnPosition(conn, &player.position)
	WritePlayerInventory(conn)
	WritePlayerPositionLook(conn, &player.position, &player.orientation,
	                        0, false)
	return player
}

func (game *Game) serveSession(conn net.Conn) {
	log.Stderr("Client connected from ", conn.RemoteAddr())

	defer func() {
		if err := recover(); err != nil {
			log.Stderr(err)
		}
		log.Stderr("Client disconnected from ", conn.RemoteAddr())
		conn.Close()
	}()

	player := startSession(conn)
	game.Enqueue(func(g *Game) { g.AddPlayer(player) })

	// TODO
}

func (game *Game) Serve(addr string) {
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

		go game.serveSession(conn)
	}
}

func (game *Game) AddPlayer(player *Player) {
	// TODO
}

func (game *Game) Enqueue(f func(*Game)) {
	game.mainQueue <- f
}

func (game *Game) mainLoop() {
	for {
		f := <-game.mainQueue
		f(game)
	}
}

func NewGame(chunkManager *ChunkManager) (game *Game) {
	game = &Game{
		chunkManager: chunkManager,
		mainQueue: make(chan func(*Game), 256),
	}

	go game.mainLoop()
	return
}
