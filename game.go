package main

import (
	"log"
	"net"
	"time"
)

type XYZ struct {
	x, y, z float64
}

type Orientation struct {
	rotation float32
	pitch    float32
}

type Game struct {
	chunkManager *ChunkManager
	mainQueue    chan func(*Game)
	player       *Player
	time         int64
}

func (game *Game) Login(conn net.Conn) {
	username, err := ReadHandshake(conn)
	if err != nil {
		log.Stderr("ReadHandshake: ", err.String())
		return
	}
	log.Stderr("Client ", conn.RemoteAddr(), " connected as ", username)
	WriteHandshake(conn, "-")

	_, _, err = ReadLogin(conn)
	if err != nil {
		log.Stderr("ReadLogin: ", err.String())
		return
	}
	WriteLogin(conn)

	StartPlayer(game, conn)
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

		go game.Login(conn)
	}
}

func (game *Game) AddPlayer(player *Player) {
	// Trivial implementation for now, need to track multiple players in the future
	game.player = player
}

func (game *Game) RemovePlayer(player *Player) {
	game.player = nil
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

func (game *Game) timer() {
	ticker := time.NewTicker(1000000000) // 1 sec
	for {
		<-ticker.C
		game.Enqueue(func(game *Game) { game.tick() })
	}
}

func (game *Game) tick() {
	game.time += 20

	if game.player != nil {
		game.player.SendTimeUpdate(game.time)
	}
}

func NewGame(chunkManager *ChunkManager) (game *Game) {
	game = &Game{
		chunkManager: chunkManager,
		mainQueue:    make(chan func(*Game), 256),
	}

	go game.mainLoop()
	go game.timer()
	return
}
