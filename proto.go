package main

import (
	"io"
	"os"
	"fmt"
	"bytes"
	"encoding/binary"
	"compress/zlib"
)

const (
	protocolVersion = 2

	// Packet type IDs
	packetIDKeepAlive          = 0x0
	packetIDLogin              = 0x1
	packetIDHandshake          = 0x2
	packetIDTimeUpdate         = 0x4
	packetIDPlayerInventory    = 0x5
	packetIDSpawnPosition      = 0x6
	packetIDFlying             = 0xa
	packetIDPlayerPosition     = 0xb
	packetIDPlayerLook         = 0xc
	packetIDPlayerPositionLook = 0xd
	packetIDPreChunk           = 0x32
	packetIDMapChunk           = 0x33
	packetIDDisconnect         = 0xff

	// Inventory types
	inventoryTypeMain     = -1
	inventoryTypeArmor    = -2
	inventoryTypeCrafting = -3
)

// Callers must implement this interface to receive packets
type PacketHandler interface {
	PacketKeepAlive()
	PacketFlying(flying bool)
	PacketPlayerPosition(position *XYZ, stance float64, flying bool)
	PacketPlayerLook(orientation *Orientation, flying bool)
	PacketDisconnect(reason string)
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func byteToBool(b byte) bool {
	return b != 0
}

func ReadString(reader io.Reader) (s string, err os.Error) {
	var length int16
	err = binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return
	}

	bs := make([]byte, uint16(length))
	_, err = io.ReadFull(reader, bs)
	return string(bs), err
}

func WriteString(writer io.Writer, s string) (err os.Error) {
	bs := []byte(s)

	err = binary.Write(writer, binary.BigEndian, int16(len(bs)))
	if err != nil {
		return
	}

	_, err = writer.Write(bs)
	return
}

func ReadHandshake(reader io.Reader) (username string, err os.Error) {
	var packetID byte
	err = binary.Read(reader, binary.BigEndian, &packetID)
	if err != nil {
		return
	}
	if packetID != packetIDHandshake {
		panic(fmt.Sprintf("ReadHandshake: invalid packet ID %#x", packetID))
	}

	return ReadString(reader)
}

func WriteHandshake(writer io.Writer, reply string) (err os.Error) {
	err = binary.Write(writer, binary.BigEndian, byte(packetIDHandshake))
	if err != nil {
		return
	}

	return WriteString(writer, reply)
}

func ReadLogin(reader io.Reader) (username, password string, err os.Error) {
	var packet struct {
		PacketID byte
		Version int32
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}
	if packet.PacketID != packetIDLogin {
		panic(fmt.Sprintf("ReadLogin: invalid packet ID %#x", packet.PacketID))
	}
	if packet.Version != protocolVersion {
		panic(fmt.Sprintf("ReadLogin: unsupported protocol version %#x", packet.Version))
	}

	username, err = ReadString(reader)
	if err != nil {
		return
	}

	password, err = ReadString(reader)
	return
}

func WriteLogin(writer io.Writer) (err os.Error) {
	_, err = writer.Write([]byte{packetIDLogin, 0, 0, 0, 0, 0, 0, 0, 0})
	return err
}

func WriteSpawnPosition(writer io.Writer, position *XYZ) (err os.Error) {
	var packet = struct {
		PacketID byte
		X int32
		Y int32
		Z int32
	}{
		packetIDSpawnPosition,
		int32(position.x),
		int32(position.y),
		int32(position.z),
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func WriteTimeUpdate(writer io.Writer, time int64) (err os.Error) {
	var packet = struct {
		PacketID byte
		Time int64
	}{
		packetIDTimeUpdate,
		time,
	}

	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func WritePlayerInventory(writer io.Writer) (err os.Error) {
	type InventoryType struct {
		inventoryType int32
		count         int16
	}
	var inventories = []InventoryType{
		InventoryType{inventoryTypeMain, 36},
		InventoryType{inventoryTypeArmor, 4},
		InventoryType{inventoryTypeCrafting, 4},
	}

	for _, inventory := range inventories {
		var packet = struct {
			PacketID byte
			InventoryType int32
			Count int16
		}{
			packetIDPlayerInventory,
			inventory.inventoryType,
			inventory.count,
		}
		err = binary.Write(writer, binary.BigEndian, &packet)
		if err != nil {
			return
		}

		for i := int16(0); i < inventory.count; i++ {
			err = binary.Write(writer, binary.BigEndian, int16(-1))
			if err != nil {
				return
			}
		}
	}
	return
}

func WritePlayerPosition(writer io.Writer, position *XYZ, stance float64, flying bool) (err os.Error) {
	var packet = struct {
		PacketID byte
		X float64
		Y float64
		Stance float64
		Z float64
		Flying byte
	}{
		packetIDPlayerPosition,
		position.x,
		position.y,
		stance,
		position.z,
		boolToByte(flying),
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func WritePlayerPositionLook(writer io.Writer, position *XYZ, orientation *Orientation, stance float64, flying bool) (err os.Error) {
	var packet = struct {
		PacketID byte
		X float64
		Y float64
		Stance float64
		Z float64
		Rotation float32
		Pitch float32
		Flying byte
	}{
		packetIDPlayerPositionLook,
		position.x,
		position.y,
		stance,
		position.z,
		orientation.rotation,
		orientation.pitch,
		boolToByte(flying),
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func WritePreChunk(writer io.Writer, x int32, z int32, willSend bool) (err os.Error) {
	var packet = struct {
		PacketID byte
		X int32
		Z int32
		WillSend byte
	}{
		packetIDPreChunk,
		x,
		z,
		boolToByte(willSend),
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func WriteMapChunk(writer io.Writer, chunk *Chunk) (err os.Error) {
	buf := &bytes.Buffer{}
	compressed, err := zlib.NewWriter(buf)
	if err != nil {
		return
	}

	compressed.Write(chunk.blocks)
	compressed.Write(chunk.blockData)
	compressed.Write(chunk.blockLight)
	compressed.Write(chunk.skyLight)
	compressed.Close()
	bs := buf.Bytes()

	var packet = struct {
		PacketID byte
		X int32
		Y int16
		Z int32
		SizeX byte
		SizeY byte
		SizeZ byte
		CompressedLength int32
		Compressed []byte
	}{
		packetIDMapChunk,
		chunk.x * ChunkSizeX,
		0,
		chunk.z * ChunkSizeZ,
		ChunkSizeX - 1,
		ChunkSizeY - 1,
		ChunkSizeZ - 1,
		int32(len(bs)),
		bs,
	}

	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func ReadKeepAlive(reader io.Reader, handler PacketHandler) (err os.Error) {
	handler.PacketKeepAlive()
	return
}

func ReadFlying(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		Flying byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketFlying(byteToBool(packet.Flying))
	return
}

func ReadPlayerPosition(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		X float64
		Y float64
		Stance float64
		Z float64
		Flying byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(&XYZ{packet.X, packet.Y, packet.Z}, packet.Stance, byteToBool(packet.Flying))
	return
}

func ReadPlayerLook(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		Rotation float32
		Pitch float32
		Flying byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerLook(&Orientation{packet.Rotation, packet.Pitch}, byteToBool(packet.Flying))
	return
}

func ReadPlayerPositionLook(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		X float64
		Y float64
		Stance float64
		Z float64
		Rotation float32
		Pitch float32
		Flying byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(&XYZ{packet.X, packet.Y, packet.Z}, packet.Stance, byteToBool(packet.Flying))
	handler.PacketPlayerLook(&Orientation{packet.Rotation, packet.Pitch}, byteToBool(packet.Flying))
	return
}

func ReadDisconnect(reader io.Reader, handler PacketHandler) (err os.Error) {
	reason, err := ReadString(reader)
	if err != nil {
		return
	}

	handler.PacketDisconnect(reason)
	return
}

// Packet reader functions
var readFns = map[byte]func(io.Reader, PacketHandler) os.Error {
	packetIDKeepAlive: ReadKeepAlive,
	packetIDFlying: ReadFlying,
	packetIDPlayerPosition: ReadPlayerPosition,
	packetIDPlayerLook: ReadPlayerLook,
	packetIDPlayerPositionLook: ReadPlayerPositionLook,
	packetIDDisconnect: ReadDisconnect,
}

func ReadPacket(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packetID byte

	err = binary.Read(reader, binary.BigEndian, &packetID)
	fn, ok := readFns[packetID]
	if !ok {
		return os.NewError(fmt.Sprintf("unhandled packet type %#x", packetID))
	}

	err = fn(reader, handler)
	return
}
