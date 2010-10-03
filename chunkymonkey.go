package main

import (
	"os"
	"path"
	"log"
	"nbt"
)

// The player's starting position is loaded from level.dat for now
var StartPosition XYZ
func loadStartPosition(worldPath string) {
	file, err := os.Open(path.Join(worldPath, "level.dat"), os.O_RDONLY, 0)
	if err != nil {
		log.Exit("loadStartPosition: ", err.String())
	}

	level, err := nbt.Read(file)
	file.Close()
	if err != nil {
		log.Exit("loadStartPosition: ", err.String())
	}

	pos := level.Lookup("/Data/Player/Pos")
	StartPosition = XYZ{
		pos.(*nbt.List).Value[0].(*nbt.Double).Value,
		pos.(*nbt.List).Value[1].(*nbt.Double).Value,
		pos.(*nbt.List).Value[2].(*nbt.Double).Value,
	}
}

func main() {
	if len(os.Args) != 2 {
		os.Stderr.WriteString("usage: " + os.Args[0] + " <world>\n")
		os.Exit(1)
	}

	loadStartPosition(os.Args[1])
	chunkManager := NewChunkManager(os.Args[1])
	game := NewGame(chunkManager)
	game.Serve(":25565")
}
