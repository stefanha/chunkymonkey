package main

import (
	"bytes"
	"log"
	"net"
	"time"
	"fmt"
)

type XYZ struct {
	x, y, z float64
}

type Orientation struct {
	rotation float32
	pitch    float32
}

type Game struct {
	chunkManager  *ChunkManager
	mainQueue     chan func(*Game)
	entityManager EntityManager
	players       map[EntityID]*Player
	time          int64
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

	StartPlayer(game, conn, username)
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

		go game.Login(WrapConn(conn))
	}
}

func (game *Game) AddPlayer(player *Player) {
	game.entityManager.AddEntity(&player.Entity)
	game.players[player.EntityID] = player
	game.SendChatMessage(fmt.Sprintf("%s has joined", player.name))
	game.chunkManager.AddPlayer(player)
}

func (game *Game) RemovePlayer(player *Player) {
	game.chunkManager.RemovePlayer(player)
	game.players[player.EntityID] = nil, false
	game.entityManager.RemoveEntity(&player.Entity)
	game.SendChatMessage(fmt.Sprintf("%s has left", player.name))
}

func (game *Game) MulticastPacket(packet []byte, except *Player) {
	for _, player := range game.players {
		if player == except {
			continue
		}

		player.TransmitPacket(packet)
	}
}

func (game *Game) SendChatMessage(message string) {
	buf := &bytes.Buffer{}
	WriteChatMessage(buf, message)
	game.MulticastPacket(buf.Bytes(), nil)
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

func (game *Game) sendTimeUpdate() {
	buf := &bytes.Buffer{}
	WriteTimeUpdate(buf, game.time)
	game.MulticastPacket(buf.Bytes(), nil)
}

func (game *Game) tick() {
	game.time += 20
	game.sendTimeUpdate()
}

func NewGame(chunkManager *ChunkManager) (game *Game) {
	game = &Game{
		chunkManager: chunkManager,
		mainQueue:    make(chan func(*Game), 256),
		players:      make(map[EntityID]*Player),
	}

	go game.mainLoop()
	go game.timer()
	return
}
