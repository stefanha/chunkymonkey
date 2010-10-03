package main

import (
	"io"
	"log"
	"net"
	"bytes"
)

const (
	chunkRadius = 1
)

type Player struct {
	game         *Game
	conn         net.Conn
	position     XYZ
	orientation  Orientation
	txQueue      chan []byte
}

func StartPlayer(game *Game, conn net.Conn) {
	player := &Player{
		game:        game,
		conn:        conn,
		position:    XYZ{0, 64, 0},
		orientation: Orientation{0, 0},
		txQueue:     make(chan []byte, 128),
	}

	go player.ReceiveLoop()
	go player.TransmitLoop()

	game.Enqueue(func(*Game) { player.postLogin() })
}

func (player *Player) ReceiveLoop() {
	// TODO
}

func (player *Player) TransmitLoop() {
	for {
		bs := <-player.txQueue
		_, err := player.conn.Write(bs)
		if err != nil {
			log.Stderr("TransmitLoop failed: ", err.String())
			return
		}
	}
}

func (player *Player) sendChunks(writer io.Writer) {
	playerX := int32(player.position.x) / ChunkSizeX
	playerZ := int32(player.position.z) / ChunkSizeZ

	for z := playerZ - chunkRadius; z < playerZ + chunkRadius; z++ {
		for x := playerX - chunkRadius; x < playerX + chunkRadius; x++ {
			WritePreChunk(writer, x, z, true)
		}
	}

	for z := playerZ - chunkRadius; z < playerZ + chunkRadius; z++ {
		for x := playerX - chunkRadius; x < playerX + chunkRadius; x++ {
			log.Stderr("sendChunks x=", x, " z=", z)
			chunk := player.game.chunkManager.Get(x, z)
			WriteMapChunk(writer, chunk)
		}
	}
}

func (player *Player) postLogin() {
	buf := &bytes.Buffer{}
	WriteSpawnPosition(buf, &player.position)
	player.sendChunks(buf)
	WritePlayerInventory(buf)
	WritePlayerPositionLook(buf, &player.position, &player.orientation,
		0, false)
	player.txQueue <- buf.Bytes()
}
