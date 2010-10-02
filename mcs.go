package main

import (
	"os"
)

func main() {
	chunkManager := NewChunkManager(os.Args[1])
	game := &Game{chunkManager}
	Serve(":25565", game)
}
