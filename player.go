package main

import (
	"log"
	"net"
	"bytes"
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

	buf := &bytes.Buffer{}
	WriteSpawnPosition(buf, &player.position)
	WritePlayerInventory(buf)
	WritePlayerPositionLook(buf, &player.position, &player.orientation,
		0, false)
	player.txQueue <- buf.Bytes()

	go player.ReceiveLoop()
	go player.TransmitLoop()
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
