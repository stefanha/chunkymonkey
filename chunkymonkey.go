package main

import (
	"os"
	"path"
	"flag"
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

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " <world>\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	worldPath := flag.Arg(0)

	loadStartPosition(worldPath)
	chunkManager := NewChunkManager(worldPath)
	game := NewGame(chunkManager)
	game.Serve(":25565")
}
