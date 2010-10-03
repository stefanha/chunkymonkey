package main

import (
	"os"
)

func main() {
	if len(os.Args) != 2 {
		os.Stderr.WriteString("usage: " + os.Args[0] + " <world>\n")
		os.Exit(1)
	}

	chunkManager := NewChunkManager(os.Args[1])
	game := NewGame(chunkManager)
	game.Serve(":25565")
}
