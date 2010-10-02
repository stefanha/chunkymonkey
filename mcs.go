package main

import (
	"os"
)

func main() {
	chunkManager := NewChunkManager(os.Args[1])
	game := NewGame(chunkManager)
	game.Serve(":25565")
}
