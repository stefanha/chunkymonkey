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
	packetIDLogin              = 0x1
	packetIDHandshake          = 0x2
	packetIDPlayerInventory    = 0x5
	packetIDSpawnPosition      = 0x6
	packetIDPlayerPositionLook = 0xd
	packetIDPreChunk           = 0x32
	packetIDMapChunk           = 0x33

	// Inventory types
	inventoryTypeMain     = -1
	inventoryTypeArmor    = -2
	inventoryTypeCrafting = -3
)

func ReadByte(reader io.Reader) (b byte, err os.Error) {
	err = binary.Read(reader, binary.BigEndian, &b)
	return
}

func WriteByte(writer io.Writer, b byte) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, b)
}

func WriteBool(writer io.Writer, b bool) (err os.Error) {
	var val byte

	if b {
		val = 1
	} else {
		val = 0
	}

	return WriteByte(writer, val)
}

func ReadInt16(reader io.Reader) (i int16, err os.Error) {
	err = binary.Read(reader, binary.BigEndian, &i)
	return
}

func WriteInt16(writer io.Writer, i int16) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, i)
}

func ReadInt32(reader io.Reader) (i int32, err os.Error) {
	err = binary.Read(reader, binary.BigEndian, &i)
	return
}

func WriteInt32(writer io.Writer, i int32) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, i)
}

func WriteFloat32(writer io.Writer, f float32) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, f)
}

func WriteFloat64(writer io.Writer, f float64) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, f)
}

func ReadString(reader io.Reader) (s string, err os.Error) {
	n, e := ReadInt16(reader)
	if e != nil {
		return "", e
	}

	bs := make([]byte, uint16(n))
	_, err = io.ReadFull(reader, bs)
	return string(bs), err
}

func WriteString(writer io.Writer, s string) (err os.Error) {
	bs := []byte(s)

	err = WriteInt16(writer, int16(len(bs)))
	if err != nil {
		return err
	}

	_, err = writer.Write(bs)
	return err
}

func ReadHandshake(reader io.Reader) (username string, err os.Error) {
	packetID, e := ReadByte(reader)
	if e != nil {
		return "", e
	}
	if packetID != packetIDHandshake {
		panic(fmt.Sprintf("ReadHandshake: invalid packet ID %#x", packetID))
	}

	return ReadString(reader)
}

func WriteHandshake(writer io.Writer, reply string) (err os.Error) {
	err = WriteByte(writer, packetIDHandshake)
	if err != nil {
		return
	}

	return WriteString(writer, reply)
}

func ReadLogin(reader io.Reader) (username, password string, err os.Error) {
	packetID, e := ReadByte(reader)
	if e != nil {
		return "", "", e
	}
	if packetID != packetIDLogin {
		panic(fmt.Sprintf("ReadLogin: invalid packet ID %#x", packetID))
	}

	version, e2 := ReadInt32(reader)
	if e2 != nil {
		return "", "", e2
	}
	if version != protocolVersion {
		panic(fmt.Sprintf("ReadLogin: unsupported protocol version %#x", version))
	}

	username, e3 := ReadString(reader)
	if e3 != nil {
		return "", "", e3
	}

	password, e4 := ReadString(reader)
	if e4 != nil {
		return "", "", e4
	}

	return username, password, nil
}

func WriteLogin(writer io.Writer) (err os.Error) {
	_, err = writer.Write([]byte{packetIDLogin, 0, 0, 0, 0, 0, 0, 0, 0})
	return err
}

func WriteSpawnPosition(writer io.Writer, position *XYZ) (err os.Error) {
	err = WriteByte(writer, packetIDSpawnPosition)
	if err != nil {
		return
	}

	err = WriteInt32(writer, int32(position.x))
	if err != nil {
		return
	}

	err = WriteInt32(writer, int32(position.y))
	if err != nil {
		return
	}

	err = WriteInt32(writer, int32(position.z))
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
		err = WriteByte(writer, packetIDPlayerInventory)
		if err != nil {
			return
		}

		err = WriteInt32(writer, inventory.inventoryType)
		if err != nil {
			return
		}

		err = WriteInt16(writer, inventory.count)
		if err != nil {
			return
		}

		for i := int16(0); i < inventory.count; i++ {
			err = WriteInt16(writer, -1)
			if err != nil {
				return
			}
		}
	}
	return
}

func WritePlayerPositionLook(writer io.Writer, position *XYZ, orientation *Orientation, stance float64, flying bool) (err os.Error) {
	err = WriteByte(writer, packetIDPlayerPositionLook)
	if err != nil {
		return
	}

	err = WriteFloat64(writer, position.x)
	if err != nil {
		return
	}

	err = WriteFloat64(writer, position.y)
	if err != nil {
		return
	}

	err = WriteFloat64(writer, stance)
	if err != nil {
		return
	}

	err = WriteFloat64(writer, position.z)
	if err != nil {
		return
	}

	err = WriteFloat32(writer, orientation.rotation)
	if err != nil {
		return
	}

	err = WriteFloat32(writer, orientation.pitch)
	if err != nil {
		return
	}

	err = WriteBool(writer, flying)
	return
}

func WritePreChunk(writer io.Writer, x int32, z int32, willSend bool) (err os.Error) {
	err = WriteByte(writer, packetIDPreChunk)
	if err != nil {
		return
	}

	err = WriteInt32(writer, x)
	if err != nil {
		return
	}

	err = WriteInt32(writer, z)
	if err != nil {
		return
	}

	err = WriteBool(writer, willSend)
	return
}

func WriteMapChunk(writer io.Writer, chunk *Chunk) (err os.Error) {
	err = WriteByte(writer, packetIDMapChunk)
	if err != nil {
		return
	}

	err = WriteInt32(writer, chunk.x)
	if err != nil {
		return
	}

	err = WriteInt16(writer, 0)
	if err != nil {
		return
	}

	err = WriteInt32(writer, chunk.z)
	if err != nil {
		return
	}

	err = WriteByte(writer, byte(ChunkSizeX - 1))
	if err != nil {
		return
	}

	err = WriteByte(writer, byte(ChunkSizeY - 1))
	if err != nil {
		return
	}

	err = WriteByte(writer, byte(ChunkSizeZ - 1))
	if err != nil {
		return
	}

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

	err = WriteInt32(writer, int32(len(bs)))
	if err != nil {
		return
	}

	_, err = writer.Write(bs)
	return
}
