package main

import (
	"io"
	"os"
	"nbt"
)

type Chunk struct {
	x, z int32
	blocks []byte
	blockData []byte
	skyLight []byte
	blockLight []byte
	heightMap []byte
}

func LoadChunk(reader io.Reader) (chunk *Chunk, err os.Error){
	level, err := nbt.Read(reader)
	if err != nil {
		return
	}

	chunk = &Chunk{
		x: level.Lookup("/Level/xPos").(*nbt.Int).Value,
		z: level.Lookup("/Level/zPos").(*nbt.Int).Value,
		blocks: level.Lookup("/Level/Blocks").(*nbt.ByteArray).Value,
		blockData: level.Lookup("/Level/Data").(*nbt.ByteArray).Value,
		skyLight: level.Lookup("/Level/SkyLight").(*nbt.ByteArray).Value,
		blockLight: level.Lookup("/Level/BlockLight").(*nbt.ByteArray).Value,
		heightMap: level.Lookup("/Level/HeightMap").(*nbt.ByteArray).Value,
	}
	return
}
