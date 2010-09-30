package main

import (
	"net"
	"log"
	"os"
	"fmt"
)

const (
	protocolVersion = 2

	// Packet type IDs
	packetIDLogin = 0x1
	packetIDHandshake = 0x2
)

func ReadByte(conn net.Conn) (b byte, err os.Error) {
	bs := make([]byte, 1)
	_, err = conn.Read(bs)
	return bs[0], err
}

func ReadShort(conn net.Conn) (i int, err os.Error) {
	bs := make([]byte, 2)
	_, err = conn.Read(bs)
	return int(uint16(bs[0]) << 8 | uint16(bs[1])), err
}

func WriteShort(conn net.Conn, i int) (err os.Error) {
	bs := []byte{byte(i >> 8), byte(i)}
	_, err = conn.Write(bs)
	return err
}

func ReadInt(conn net.Conn) (i int, err os.Error) {
	bs := make([]byte, 4)
	_, err = conn.Read(bs)
	return int(uint32(bs[0]) << 24 | uint32(bs[1]) << 16 | uint32(bs[2]) << 8 | uint32(bs[3])), err
}

func ReadString(conn net.Conn) (s string, err os.Error) {
	n, e := ReadShort(conn)
	if e != nil {
		return "", e
	}

	bs := make([]byte, n)
	_, err = conn.Read(bs)
	return string(bs), err
}

func WriteString(conn net.Conn, s string) (err os.Error) {
	bs := []byte(s)

	err = WriteShort(conn, len(bs))
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
		log.Exitf("ReadHandshake: invalid packet ID %#x", packetID)
	}

	return ReadString(conn)
}

func WriteHandshake(conn net.Conn, reply string) (err os.Error) {
	_, err = conn.Write([]byte{packetIDHandshake})
	if err != nil {
		return err
	}

	return WriteString(conn, reply)
}

func ReadLogin(conn net.Conn) (username, password string, err os.Error) {
	packetID, e := ReadByte(conn)
	if e != nil {
		return "", "", e
	}
	if packetID != packetIDLogin {
		log.Exitf("ReadLogin: invalid packet ID %#x", packetID)
	}

	version, e2 := ReadInt(conn)
	if e2 != nil {
		return "", "", e2
	}
	if version != protocolVersion {
		log.Exit("ReadLogin: unsupported protocol version %#x", version)
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

func StartSession(conn net.Conn) {
	log.Stderr("Client connected from ", conn.RemoteAddr())

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
}

func ServeSession(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Stderr(err)
		}
		conn.Close()
	}()

	StartSession(conn)
}

func Serve(addr string) {
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

		go ServeSession(conn)
	}
}

func main() {
	Serve(":25565")
}
