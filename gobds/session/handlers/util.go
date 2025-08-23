package handlers

import (
	"log/slog"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

var (
	airRuntimeID  uint32
	denyRuntimeID uint32

	log *slog.Logger
)

func init() {
	log = slog.Default()
}

// SetupRuntimeIDs ...
func SetupRuntimeIDs(hashedNetworkIDS bool) {
	if hashedNetworkIDS {
		airRuntimeID = uint32(Hash(block.Air{}))
		denyRuntimeID = uint32(Hash(gblock.Deny{}))
	} else {
		air, ok := chunk.StateToRuntimeID("minecraft:air", nil)
		if !ok {
			panic("cannot find air runtime ID")
		}
		deny, ok := chunk.StateToRuntimeID("minecraft:deny", nil)
		if !ok {
			panic("cannot find deny runtime ID")
		}

		airRuntimeID = air
		denyRuntimeID = deny
	}
}

// claimDimensionToInt ...
func claimDimensionToInt(dimension string) int32 {
	switch dimension {
	case "minecraft:overworld":
		return 0
	case "minecraft:nether":
		return 1
	case "minecraft:end":
		return 2
	default:
		return -1
	}
}

// ClaimAt ...
func ClaimAt(dimension int32, x, z float32) (claim.PlayerClaim, bool) {
	for _, c := range infra.Claims() {
		if claimDimensionToInt(c.Location.Dimension) == dimension {
			minX := min(c.Location.Pos1.X, c.Location.Pos2.X)
			maxX := max(c.Location.Pos1.X, c.Location.Pos2.X)
			minZ := min(c.Location.Pos1.Z, c.Location.Pos2.Z)
			maxZ := max(c.Location.Pos1.Z, c.Location.Pos2.Z)
			if x >= minX && x <= maxX && z >= minZ && z <= maxZ {
				return c, true
			}
		}
	}
	return claim.PlayerClaim{}, false
}

// ClaimAtChunk ...
func ClaimAtChunk(dimension int32, chunkPos protocol.ChunkPos) (claim.PlayerClaim, bool) {
	chunkMinX := float32(chunkPos.X() << 4)
	chunkMinZ := float32(chunkPos.Z() << 4)
	chunkMaxX := chunkMinX + 15
	chunkMaxZ := chunkMinZ + 15
	for _, c := range infra.Claims() {
		if claimDimensionToInt(c.Location.Dimension) != dimension {
			continue
		}
		minX := min(c.Location.Pos1.X, c.Location.Pos2.X)
		maxX := max(c.Location.Pos1.X, c.Location.Pos2.X)
		minZ := min(c.Location.Pos1.Z, c.Location.Pos2.Z)
		maxZ := max(c.Location.Pos1.Z, c.Location.Pos2.Z)
		if chunkMinX <= maxX && chunkMaxX >= minX &&
			chunkMinZ <= maxZ && chunkMaxZ >= minZ {
			return c, true
		}
	}
	return claim.PlayerClaim{}, false
}

// blockPosToVec3 ...
func blockPosToVec3(pos protocol.BlockPos) mgl32.Vec3 {
	return mgl32.Vec3{float32(pos.X()), float32(pos.Y()), float32(pos.Z())}
}
