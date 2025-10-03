package area

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Area2D ...
type Area2D struct {
	MinX, MinZ int32
	MaxX, MaxZ int32
}

// NewArea2D ...
func NewArea2D(minX, minZ int32, maxX, maxZ int32) *Area2D {
	return &Area2D{
		MinX: minX,
		MinZ: minZ,
		MaxX: maxX,
		MaxZ: maxZ,
	}
}

// PositionInside ...
func (a *Area2D) PositionInside(x, z int32) bool {
	return x > a.MinX && x < a.MaxX &&
		z > a.MinZ && z < a.MaxZ
}

// ChunkInside ...
func (a *Area2D) ChunkInside(chunk protocol.ChunkPos) bool {
	chunkMinX, chunkMinZ := chunk.X()<<4, chunk.Z()<<4
	chunkMaxX, chunkMaxZ := chunkMinX+15, chunkMinZ+15
	return chunkMinX <= a.MaxX && chunkMaxX >= a.MinX &&
		chunkMinZ <= a.MaxZ && chunkMaxZ >= a.MinZ
}
