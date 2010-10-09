// Map chunks

package main

import (
	"bytes"
	"io"
	"os"
	"log"
	"path"
	"nbt"
)

const (
	// Chunk coordinates can be converted to block coordinates
	ChunkSizeX = 16
	ChunkSizeY = 128
	ChunkSizeZ = 16

	// The area within which a client receives updates
	ChunkRadius = 10
)

type ChunkCoord int32

// A chunk is slice of the world map
type Chunk struct {
	X, Z       ChunkCoord
	Blocks     []byte
	BlockData  []byte
	SkyLight   []byte
	BlockLight []byte
	HeightMap  []byte
	players    map[EntityID]*Player
}

// Convert an (x, z) block coordinate pair to chunk coordinates
func BlockToChunkCoords(blockX float64, blockZ float64) (chunkX ChunkCoord, chunkZ ChunkCoord) {
	return ChunkCoord(blockX / ChunkSizeX), ChunkCoord(blockZ / ChunkSizeZ)
}

// Load a chunk from its NBT representation
func loadChunk(reader io.Reader) (chunk *Chunk, err os.Error) {
	level, err := nbt.Read(reader)
	if err != nil {
		return
	}

	chunk = &Chunk{
		X:          ChunkCoord(level.Lookup("/Level/xPos").(*nbt.Int).Value),
		Z:          ChunkCoord(level.Lookup("/Level/zPos").(*nbt.Int).Value),
		Blocks:     level.Lookup("/Level/Blocks").(*nbt.ByteArray).Value,
		BlockData:  level.Lookup("/Level/Data").(*nbt.ByteArray).Value,
		SkyLight:   level.Lookup("/Level/SkyLight").(*nbt.ByteArray).Value,
		BlockLight: level.Lookup("/Level/BlockLight").(*nbt.ByteArray).Value,
		HeightMap:  level.Lookup("/Level/HeightMap").(*nbt.ByteArray).Value,
		players:    make(map[EntityID]*Player),
	}
	return
}

// ChunkManager contains all chunks and can look them up
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
		s = string(alphabet[i:i+1]) + s
	}
	if negative {
		s = "-" + s
	}
	return
}

func (mgr *ChunkManager) chunkPath(x ChunkCoord, z ChunkCoord) string {
	return path.Join(mgr.worldPath, base36Encode(int32(x&63)), base36Encode(int32(z&63)),
		"c."+base36Encode(int32(x))+"."+base36Encode(int32(z))+".dat")
}

// Get a chunk at given coordinates
func (mgr *ChunkManager) Get(x ChunkCoord, z ChunkCoord) (chunk *Chunk) {
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

// Return a channel to iterate over all chunks within a chunk's radius
func (mgr *ChunkManager) ChunksInRadius(chunkX ChunkCoord, chunkZ ChunkCoord) (c chan *Chunk) {
	c = make(chan *Chunk)
	go func() {
		for z := chunkZ - ChunkRadius; z <= chunkZ + ChunkRadius; z++ {
			for x := chunkX - ChunkRadius; x <= chunkX + ChunkRadius; x++ {
				c <- mgr.Get(x, z)
			}
		}
		close(c)
	}()
	return
}

// Return a channel to iterate over all chunks within a player's radius
func (mgr *ChunkManager) ChunksInPlayerRadius(player *Player) chan *Chunk {
	playerX, playerZ := BlockToChunkCoords(player.position.x, player.position.z)
	return mgr.ChunksInRadius(playerX, playerZ)
}

// Return a channel to iterate over all players within a chunk's radius
func (mgr *ChunkManager) PlayersInRadius(x ChunkCoord, z ChunkCoord) (c chan *Player) {
	c = make(chan *Player)
	go func() {
		alreadySent := make(map[EntityID]*Player)
		for chunk := range mgr.ChunksInRadius(x, z) {
			for entityID, player := range chunk.players {
				if _, ok := alreadySent[entityID]; !ok {
					c <- player
					alreadySent[entityID] = player
				}
			}
		}
		close(c)
	}()
	return
}

// Return a channel to iterate over all players within a chunk's radius
func (mgr *ChunkManager) PlayersInPlayerRadius(player *Player) chan *Player {
	x, z := BlockToChunkCoords(player.position.x, player.position.z)
	return mgr.PlayersInRadius(x, z)
}

// Transmit a packet to all players in radius (except the player itself)
func (mgr *ChunkManager) MulticastPacket(packet []byte, sender *Player) {
	for receiver := range mgr.PlayersInPlayerRadius(sender) {
		if receiver == sender {
			continue
		}

		receiver.TransmitPacket(packet)
	}
}

// Add a player to the game
// This function sends spawn messages to all players in range.  It also spawns
// all existing players so the new player can see them.
func (mgr *ChunkManager) AddPlayer(player *Player) {
	// Add player to chunks within radius
	for chunk := range mgr.ChunksInPlayerRadius(player) {
		chunk.players[player.EntityID] = player
	}

	// Spawn new player for existing players
	buf := &bytes.Buffer{}
	WriteNamedEntitySpawn(buf, player.EntityID, player.name, &player.position, &player.orientation, player.currentItem)
	mgr.MulticastPacket(buf.Bytes(), player)

	// Spawn existing players for new player
	buf = &bytes.Buffer{}
	for existing := range mgr.PlayersInPlayerRadius(player) {
		if existing == player {
			continue
		}

		WriteNamedEntitySpawn(buf, existing.EntityID, existing.name, &existing.position, &existing.orientation, existing.currentItem)
	}
	player.TransmitPacket(buf.Bytes())
}

// Remove a player from the game
// This function sends destroy messages so the other players see the player
// disappear.
func (mgr *ChunkManager) RemovePlayer(player *Player) {
	// Destroy player for other players
	buf := &bytes.Buffer{}
	WriteDestroyEntity(buf, player.EntityID)
	mgr.MulticastPacket(buf.Bytes(), player)

	for chunk := range mgr.ChunksInPlayerRadius(player) {
		chunk.players[player.EntityID] = nil, false
	}
}
