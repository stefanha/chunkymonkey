package main

import (
	"os"
	"io"
	"log"
	"net"
	"math"
	"bytes"
)

type Player struct {
	Entity
	game        *Game
	conn        net.Conn
	name        string
	position    XYZ
	orientation Orientation
	currentItem int16
	txQueue     chan []byte
}

func StartPlayer(game *Game, conn net.Conn, name string) {
	player := &Player{
		game:        game,
		conn:        conn,
		name:        name,
		position:    StartPosition,
		orientation: Orientation{0, 0},
		txQueue:     make(chan []byte, 128),
	}

	go player.ReceiveLoop()
	go player.TransmitLoop()

	game.Enqueue(func(game *Game) {
		game.AddPlayer(player)
		player.postLogin()
	})
}

func (player *Player) PacketKeepAlive() {
}

func (player *Player) PacketChatMessage(message string) {
	log.Stderrf("PacketChatMessage message=%s", message)

	player.game.Enqueue(func(game *Game) { game.SendChatMessage(message) })
}

func (player *Player) PacketFlying(flying bool) {
}

func (player *Player) PacketPlayerPosition(position *XYZ, stance float64, flying bool) {
	log.Stderrf("PacketPlayerPosition position=(%.2f, %.2f, %.2f) stance=%.2f flying=%v",
		position.x, position.y, position.z, stance, flying)

	player.game.Enqueue(func(game *Game) {
		var delta = XYZ{position.x - player.position.x,
			position.y - player.position.y,
			position.z - player.position.z}
		distance := math.Sqrt(delta.x*delta.x + delta.y*delta.y + delta.z*delta.z)
		if distance > 10 {
			log.Stderrf("Discarding player position that is too far removed (%.2f, %.2f, %.2f)",
				position.x, position.y, position.z)
			return
		}

		player.position = *position

		buf := &bytes.Buffer{}
		WriteEntityTeleport(buf, player.EntityID, &player.position, &player.orientation)
		game.MulticastPacket(buf.Bytes(), player)
	})
}

func (player *Player) PacketPlayerLook(orientation *Orientation, flying bool) {
	player.game.Enqueue(func(game *Game) {
		// TODO input validation
		player.orientation = *orientation

		buf := &bytes.Buffer{}
		WriteEntityLook(buf, player.EntityID, orientation)
		game.MulticastPacket(buf.Bytes(), player)
	})
}

func (player *Player) PacketPlayerDigging(status byte, x int32, y byte, z int32, face byte) {
	log.Stderrf("PacketPlayerDigging status=%d x=%d y=%d z=%d face=%d",
		status, x, y, z, face)
}

func (player *Player) PacketPlayerBlockPlacement(blockItemID int16, x int32, y byte, z int32, direction byte) {
	log.Stderrf("PacketPlayerBlockPlacement blockItemID=%d x=%d y=%d z=%d direction=%d",
		blockItemID, x, y, z, direction)
}

func (player *Player) PacketHoldingChange(blockItemID int16) {
	log.Stderrf("PacketHoldingChange blockItemID=%d", blockItemID)
}

func (player *Player) PacketArmAnimation(forward bool) {
	log.Stderrf("PacketArmAnimation forward=%v", forward)
}

func (player *Player) PacketDisconnect(reason string) {
	log.Stderrf("PacketDisconnect reason=%s", reason)
	player.game.Enqueue(func(game *Game) {
		game.RemovePlayer(player)
		close(player.txQueue)
		player.conn.Close()
	})
}

func (player *Player) ReceiveLoop() {
	for {
		err := ReadPacket(player.conn, player)
		if err != nil {
			if err != os.EOF {
				log.Stderr("ReceiveLoop failed: ", err.String())
			}
			return
		}
	}
}

func (player *Player) TransmitLoop() {
	for {
		bs := <-player.txQueue
		if bs == nil {
			return // txQueue closed
		}

		_, err := player.conn.Write(bs)
		if err != nil {
			if err != os.EOF {
				log.Stderr("TransmitLoop failed: ", err.String())
			}
			return
		}
	}
}

func (player *Player) sendChunks(writer io.Writer) {
	playerX := ChunkCoord(player.position.x / ChunkSizeX)
	playerZ := ChunkCoord(player.position.z / ChunkSizeZ)

	for z := playerZ - ChunkRadius; z <= playerZ+ChunkRadius; z++ {
		for x := playerX - ChunkRadius; x <= playerX+ChunkRadius; x++ {
			WritePreChunk(writer, x, z, true)
		}
	}

	for z := playerZ - ChunkRadius; z <= playerZ+ChunkRadius; z++ {
		for x := playerX - ChunkRadius; x <= playerX+ChunkRadius; x++ {
			chunk := player.game.chunkManager.Get(x, z)
			WriteMapChunk(writer, chunk)
		}
	}
}

func (player *Player) TransmitPacket(packet []byte) {
	if packet == nil {
		return // skip empty packets
	}
	player.txQueue <- packet
}

func (player *Player) postLogin() {
	buf := &bytes.Buffer{}
	WriteSpawnPosition(buf, &player.position)
	player.sendChunks(buf)
	WritePlayerInventory(buf)
	WritePlayerPositionLook(buf, &player.position, &player.orientation,
		0, false)
	player.TransmitPacket(buf.Bytes())
}
