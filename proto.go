package main

import (
	"net"
	"os"
	"fmt"
	"encoding/binary"
)

const (
	protocolVersion = 2

	// Packet type IDs
	packetIDLogin = 0x1
	packetIDHandshake = 0x2
	packetIDPlayerInventory = 0x5
	packetIDSpawnPosition = 0x6
	packetIDPlayerPositionLook = 0xd

	// Inventory types
	inventoryTypeMain = -1
	inventoryTypeArmor = -2
	inventoryTypeCrafting = -3
)

func ReadByte(conn net.Conn) (b byte, err os.Error) {
	err = binary.Read(conn, binary.BigEndian, &b)
	return
}

func WriteByte(conn net.Conn, b byte) (err os.Error) {
	return binary.Write(conn, binary.BigEndian, b)
}

func WriteBool(conn net.Conn, b bool) (err os.Error) {
	var val byte

	if b {
		val = 1
	} else {
		val = 0
	}

	return WriteByte(conn, val)
}

func ReadInt16(conn net.Conn) (i int16, err os.Error) {
	err = binary.Read(conn, binary.BigEndian, &i)
	return
}

func WriteInt16(conn net.Conn, i int16) (err os.Error) {
	return binary.Write(conn, binary.BigEndian, i)
}

func ReadInt32(conn net.Conn) (i int32, err os.Error) {
	err = binary.Read(conn, binary.BigEndian, &i)
	return
}

func WriteInt32(conn net.Conn, i int32) (err os.Error) {
	return binary.Write(conn, binary.BigEndian, i)
}

func WriteFloat32(conn net.Conn, f float32) (err os.Error) {
	return binary.Write(conn, binary.BigEndian, f)
}

func WriteFloat64(conn net.Conn, f float64) (err os.Error) {
	return binary.Write(conn, binary.BigEndian, f)
}

func ReadString(conn net.Conn) (s string, err os.Error) {
	n, e := ReadInt16(conn)
	if e != nil {
		return "", e
	}

	bs := make([]byte, uint16(n))
	_, err = conn.Read(bs)
	return string(bs), err
}

func WriteString(conn net.Conn, s string) (err os.Error) {
	bs := []byte(s)

	err = WriteInt16(conn, int16(len(bs)))
	if err != nil {
		return err
	}

	_, err = conn.Write(bs)
	return err
}

func ReadHandshake(conn net.Conn) (username string, err os.Error) {
	packetID, e := ReadByte(conn)
	if e != nil {
		return "", e
	}
	if packetID != packetIDHandshake {
		panic(fmt.Sprintf("ReadHandshake: invalid packet ID %#x", packetID))
	}

	return ReadString(conn)
}

func WriteHandshake(conn net.Conn, reply string) (err os.Error) {
	err = WriteByte(conn, packetIDHandshake)
	if err != nil {
		return
	}

	return WriteString(conn, reply)
}

func ReadLogin(conn net.Conn) (username, password string, err os.Error) {
	packetID, e := ReadByte(conn)
	if e != nil {
		return "", "", e
	}
	if packetID != packetIDLogin {
		panic(fmt.Sprintf("ReadLogin: invalid packet ID %#x", packetID))
	}

	version, e2 := ReadInt32(conn)
	if e2 != nil {
		return "", "", e2
	}
	if version != protocolVersion {
		panic(fmt.Sprintf("ReadLogin: unsupported protocol version %#x", version))
	}

	username, e3 := ReadString(conn)
	if e3 != nil {
		return "", "", e3
	}

	password, e4 := ReadString(conn)
	if e4 != nil {
		return "", "", e4
	}

	return username, password, nil
}

func WriteLogin(conn net.Conn) (err os.Error) {
	_, err = conn.Write([]byte{packetIDLogin, 0, 0, 0, 0, 0, 0, 0, 0})
	return err
}

func WriteSpawnPosition(conn net.Conn, position *XYZ) (err os.Error) {
	err = WriteByte(conn, packetIDSpawnPosition)
	if err != nil {
		return
	}

	err = WriteInt32(conn, int32(position.x))
	if err != nil {
		return
	}

	err = WriteInt32(conn, int32(position.y))
	if err != nil {
		return
	}

	err = WriteInt32(conn, int32(position.z))
	return
}

func WritePlayerInventory(conn net.Conn) (err os.Error) {
	type InventoryType struct {
		inventoryType int32
		count int16
	}
	var inventories = []InventoryType{
		InventoryType{inventoryTypeMain, 36},
		InventoryType{inventoryTypeArmor, 4},
		InventoryType{inventoryTypeCrafting, 4},
	}

	for _, inventory := range inventories {
		err = WriteByte(conn, packetIDPlayerInventory)
		if err != nil {
			return
		}

		err = WriteInt32(conn, inventory.inventoryType)
		if err != nil {
			return
		}

		err = WriteInt16(conn, inventory.count)
		if err != nil {
			return
		}

		for i := int16(0); i < inventory.count; i++ {
			err = WriteInt16(conn, -1)
			if err != nil {
				return
			}
		}
	}
	return
}

func WritePlayerPositionLook(conn net.Conn, position *XYZ,
                             orientation *Orientation, stance float64,
                             flying bool) (err os.Error) {
	err = WriteByte(conn, packetIDPlayerPositionLook)
	if err != nil {
		return
	}

	err = WriteFloat64(conn, position.x)
	if err != nil {
		return
	}

	err = WriteFloat64(conn, position.y)
	if err != nil {
		return
	}

	err = WriteFloat64(conn, stance)
	if err != nil {
		return
	}

	err = WriteFloat64(conn, position.z)
	if err != nil {
		return
	}

	err = WriteFloat32(conn, orientation.rotation)
	if err != nil {
		return
	}

	err = WriteFloat32(conn, orientation.pitch)
	if err != nil {
		return
	}

	err = WriteBool(conn, flying)
	return
}
