package main

import (
	"io"
	"os"
	"log"
	"path"
	"nbt"
)

type Chunk struct {
	x, z       int32
	blocks     []byte
	blockData  []byte
	skyLight   []byte
	blockLight []byte
	heightMap  []byte
}

func loadChunk(reader io.Reader) (chunk *Chunk, err os.Error) {
	level, err := nbt.Read(reader)
	if err != nil {
		return
	}

	chunk = &Chunk{
		x:          level.Lookup("/Level/xPos").(*nbt.Int).Value,
		z:          level.Lookup("/Level/zPos").(*nbt.Int).Value,
		blocks:     level.Lookup("/Level/Blocks").(*nbt.ByteArray).Value,
		blockData:  level.Lookup("/Level/Data").(*nbt.ByteArray).Value,
		skyLight:   level.Lookup("/Level/SkyLight").(*nbt.ByteArray).Value,
		blockLight: level.Lookup("/Level/BlockLight").(*nbt.ByteArray).Value,
		heightMap:  level.Lookup("/Level/HeightMap").(*nbt.ByteArray).Value,
	}
	return
}

type ChunkManager struct {
	worldPath string
	chunks    map[uint64]*Chunk
}

func NewChunkManager(worldPath string) *ChunkManager {
	return &ChunkManager{
		worldPath: worldPath,
		chunks:    make(map[uint64]*Chunk),
	}
}

func base36Encode(n int32) (s string) {
	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
	negative := false

	if n < 0 {
		n = -n
		negative = true
	}
	if n == 0 {
		return "0"
	}

	for n != 0 {
		i := n % int32(len(alphabet))
		n /= int32(len(alphabet))
		s = string(alphabet[i : i+1]) + s
	}
	if negative {
		s = "-" + s
	}
	return
}

func (mgr *ChunkManager) chunkPath(x int32, z int32) string {
	return path.Join(mgr.worldPath, base36Encode(x&63), base36Encode(z&63),
		"c."+base36Encode(x)+"."+base36Encode(z)+".dat")
}

func (mgr *ChunkManager) Get(x int32, z int32) (chunk *Chunk) {
	key := uint64(x)<<32 | uint64(uint32(z))
	chunk, ok := mgr.chunks[key]
	if ok {
		return
	}

	file, err := os.Open(mgr.chunkPath(x, z), os.O_RDONLY, 0)
	if err != nil {
		log.Exit("ChunkManager.Get: ", err.String())
	}

	chunk, err = loadChunk(file)
	file.Close()
	if err != nil {
		log.Exit("ChunkManager.loadChunk: ", err.String())
	}

	mgr.chunks[key] = chunk
	return
}
