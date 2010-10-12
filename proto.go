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
	// Sometimes it is useful to convert block coordinates to pixels
	PixelsPerBlock = 32

	// Currently only this protocol version is supported
	protocolVersion = 2

	// Packet type IDs
	packetIDKeepAlive            = 0x0
	packetIDLogin                = 0x1
	packetIDHandshake            = 0x2
	packetIDChatMessage          = 0x3
	packetIDTimeUpdate           = 0x4
	packetIDPlayerInventory      = 0x5
	packetIDSpawnPosition        = 0x6
	packetIDFlying               = 0xa
	packetIDPlayerPosition       = 0xb
	packetIDPlayerLook           = 0xc
	packetIDPlayerPositionLook   = 0xd
	packetIDPlayerDigging        = 0xe
	packetIDPlayerBlockPlacement = 0xf
	packetIDHoldingChange        = 0x10
	packetIDArmAnimation         = 0x12
	packetIDNamedEntitySpawn     = 0x14
	packetIDDestroyEntity        = 0x1d
	packetIDEntityLook           = 0x20
	packetIDEntityTeleport       = 0x22
	packetIDPreChunk             = 0x32
	packetIDMapChunk             = 0x33
	packetIDDisconnect           = 0xff

	// Inventory types
	inventoryTypeMain     = -1
	inventoryTypeArmor    = -2
	inventoryTypeCrafting = -3
)

// Callers must implement this interface to receive packets
type PacketHandler interface {
	PacketKeepAlive()
	PacketChatMessage(message string)
	PacketFlying(flying bool)
	PacketPlayerPosition(position *XYZ, stance float64, flying bool)
	PacketPlayerLook(orientation *Orientation, flying bool)
	PacketPlayerDigging(status byte, x int32, y byte, z int32, face byte)
	PacketPlayerBlockPlacement(blockItemID int16, x int32, y byte, z int32, direction byte)
	PacketHoldingChange(blockItemID int16)
	PacketArmAnimation(forward bool)
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
		Version  int32
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

func WriteSpawnPosition(writer io.Writer, position *XYZ) os.Error {
	var packet = struct {
		PacketID byte
		X        int32
		Y        int32
		Z        int32
	}{
		packetIDSpawnPosition,
		int32(position.x),
		int32(position.y),
		int32(position.z),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WriteTimeUpdate(writer io.Writer, time int64) os.Error {
	var packet = struct {
		PacketID byte
		Time     int64
	}{
		packetIDTimeUpdate,
		time,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
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
			PacketID      byte
			InventoryType int32
			Count         int16
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

func WritePlayerPosition(writer io.Writer, position *XYZ, stance float64, flying bool) os.Error {
	var packet = struct {
		PacketID byte
		X        float64
		Y        float64
		Stance   float64
		Z        float64
		Flying   byte
	}{
		packetIDPlayerPosition,
		position.x,
		position.y,
		stance,
		position.z,
		boolToByte(flying),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WritePlayerPositionLook(writer io.Writer, position *XYZ, orientation *Orientation, stance float64, flying bool) os.Error {
	var packet = struct {
		PacketID byte
		X        float64
		Y        float64
		Stance   float64
		Z        float64
		Rotation float32
		Pitch    float32
		Flying   byte
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
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WriteEntityLook(writer io.Writer, entityID EntityID, orientation *Orientation) os.Error {
	var packet = struct {
		PacketID byte
		EntityID int32
		Rotation byte
		Pitch    byte
	}{
		packetIDEntityLook,
		int32(entityID),
		byte(orientation.rotation * 256 / 360),
		byte(orientation.pitch * 64 / 90),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WriteEntityTeleport(writer io.Writer, entityID EntityID, position *XYZ, orientation *Orientation) os.Error {
	var packet = struct {
		PacketID byte
		EntityID int32
		X        int32
		Y        int32
		Z        int32
		Rotation byte
		Pitch    byte
	}{
		packetIDEntityTeleport,
		int32(entityID),
		int32(position.x * PixelsPerBlock),
		int32(position.y * PixelsPerBlock),
		int32(position.z * PixelsPerBlock),
		byte(orientation.rotation * 256 / 360),
		byte(orientation.pitch * 64 / 90),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WritePreChunk(writer io.Writer, x ChunkCoord, z ChunkCoord, willSend bool) os.Error {
	var packet = struct {
		PacketID byte
		X        int32
		Z        int32
		WillSend byte
	}{
		packetIDPreChunk,
		int32(x),
		int32(z),
		boolToByte(willSend),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func WriteMapChunk(writer io.Writer, chunk *Chunk) (err os.Error) {
	buf := &bytes.Buffer{}
	compressed, err := zlib.NewWriter(buf)
	if err != nil {
		return
	}

	compressed.Write(chunk.Blocks)
	compressed.Write(chunk.BlockData)
	compressed.Write(chunk.BlockLight)
	compressed.Write(chunk.SkyLight)
	compressed.Close()
	bs := buf.Bytes()

	var packet = struct {
		PacketID         byte
		X                int32
		Y                int16
		Z                int32
		SizeX            byte
		SizeY            byte
		SizeZ            byte
		CompressedLength int32
	}{
		packetIDMapChunk,
		int32(chunk.X * ChunkSizeX),
		0,
		int32(chunk.Z * ChunkSizeZ),
		ChunkSizeX - 1,
		ChunkSizeY - 1,
		ChunkSizeZ - 1,
		int32(len(bs)),
	}

	err = binary.Write(writer, binary.BigEndian, &packet)
	if err != nil {
		return
	}
	err = binary.Write(writer, binary.BigEndian, bs)
	return
}

func WriteNamedEntitySpawn(writer io.Writer, entityID EntityID, name string, position *XYZ, orientation *Orientation, currentItem int16) (err os.Error) {
	var packetStart = struct {
		PacketID byte
		EntityID int32
	}{
		packetIDNamedEntitySpawn,
		int32(entityID),
	}

	err = binary.Write(writer, binary.BigEndian, &packetStart)
	if err != nil {
		return
	}

	err = WriteString(writer, name)
	if err != nil {
		return
	}

	var packetFinish = struct {
		X           int32
		Y           int32
		Z           int32
		Rotation    byte
		Pitch       byte
		CurrentItem int16
	}{
		int32(position.x * PixelsPerBlock),
		int32(position.y * PixelsPerBlock),
		int32(position.z * PixelsPerBlock),
		byte(orientation.rotation),
		byte(orientation.pitch),
		currentItem,
	}

	err = binary.Write(writer, binary.BigEndian, &packetFinish)
	return
}

func WriteDestroyEntity(writer io.Writer, entityID EntityID) os.Error {
	var packet = struct {
		PacketID byte
		EntityID int32
	}{
		packetIDDestroyEntity,
		int32(entityID),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func ReadKeepAlive(reader io.Reader, handler PacketHandler) (err os.Error) {
	handler.PacketKeepAlive()
	return
}

func ReadChatMessage(reader io.Reader, handler PacketHandler) (err os.Error) {
	var length int16
	err = binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return
	}

	bs := make([]byte, length)
	_, err = io.ReadFull(reader, bs)
	if err != nil {
		return
	}

	// TODO sanitize chat message

	handler.PacketChatMessage(string(bs))
	return
}

func WriteChatMessage(writer io.Writer, message string) (err os.Error) {
	err = binary.Write(writer, binary.BigEndian, byte(packetIDChatMessage))
	if err != nil {
		return
	}

	err = WriteString(writer, message)
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
		X      float64
		Y      float64
		Stance float64
		Z      float64
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
		Pitch    float32
		Flying   byte
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
		X        float64
		Y        float64
		Stance   float64
		Z        float64
		Rotation float32
		Pitch    float32
		Flying   byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(&XYZ{packet.X, packet.Y, packet.Z}, packet.Stance, byteToBool(packet.Flying))
	handler.PacketPlayerLook(&Orientation{packet.Rotation, packet.Pitch}, byteToBool(packet.Flying))
	return
}

func ReadPlayerDigging(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		Status byte
		X      int32
		Y      byte
		Z      int32
		Face   byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerDigging(packet.Status, packet.X, packet.Y, packet.Z, packet.Face)
	return
}

func ReadPlayerBlockPlacement(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		ID        int16
		X         int32
		Y         byte
		Z         int32
		Direction byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerBlockPlacement(packet.ID, packet.X, packet.Y, packet.Z, packet.Direction)
	return
}

func ReadHoldingChange(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		EntityID    int32
		BlockItemID int16
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketHoldingChange(packet.BlockItemID)
	return
}

func ReadArmAnimation(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packet struct {
		EntityID int32
		Forward  byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketArmAnimation(byteToBool(packet.Forward))
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
var readFns = map[byte]func(io.Reader, PacketHandler) os.Error{
	packetIDKeepAlive:            ReadKeepAlive,
	packetIDChatMessage:          ReadChatMessage,
	packetIDFlying:               ReadFlying,
	packetIDPlayerPosition:       ReadPlayerPosition,
	packetIDPlayerLook:           ReadPlayerLook,
	packetIDPlayerPositionLook:   ReadPlayerPositionLook,
	packetIDPlayerDigging:        ReadPlayerDigging,
	packetIDPlayerBlockPlacement: ReadPlayerBlockPlacement,
	packetIDHoldingChange:        ReadHoldingChange,
	packetIDArmAnimation:         ReadArmAnimation,
	packetIDDisconnect:           ReadDisconnect,
}

func ReadPacket(reader io.Reader, handler PacketHandler) (err os.Error) {
	var packetID byte

	err = binary.Read(reader, binary.BigEndian, &packetID)
	if err != nil {
		return err
	}

	fn, ok := readFns[packetID]
	if !ok {
		return os.NewError(fmt.Sprintf("unhandled packet type %#x", packetID))
	}

	err = fn(reader, handler)
	return
}
