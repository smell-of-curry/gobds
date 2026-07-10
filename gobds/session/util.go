package session

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

var (
	denyRuntimeID       uint32
	hashedDenyRuntimeID uint32
)

// SetupRuntimeIDs ...
func SetupRuntimeIDs() {
	registry := world.DefaultBlockRegistry
	denyRuntimeID = registry.BlockRuntimeID(gblock.Deny{})
	var ok bool
	hashedDenyRuntimeID, ok = registry.RuntimeIDToHash(denyRuntimeID)
	if !ok {
		panic("cannot find hashed deny runtime ID")
	}
}

// blockByRuntimeID ...
func blockByRuntimeID(id uint32, hashed bool) (world.Block, bool) {
	registry := world.DefaultBlockRegistry
	if hashed {
		var ok bool
		id, ok = registry.HashToRuntimeID(id)
		if !ok {
			return nil, false
		}
	}
	return registry.BlockByRuntimeID(id)
}

// denyBlockRuntimeID ...
func denyBlockRuntimeID(hashed bool) uint32 {
	if hashed {
		return hashedDenyRuntimeID
	}
	return denyRuntimeID
}

// claimDimensionToInt resolves vanilla and data-driven dimension names.
func claimDimensionToInt(dimension string, definitions []protocol.DimensionDefinition) int32 {
	switch dimension {
	case "minecraft:overworld":
		return 0
	case "minecraft:nether":
		return 1
	case "minecraft:end":
		return 2
	}
	for _, definition := range definitions {
		if definition.Name == dimension {
			return definition.DimensionType
		}
	}
	return -1
}

func dimensionRangeByID(id int32, definitions []protocol.DimensionDefinition) (cube.Range, bool) {
	for _, definition := range definitions {
		if definition.DimensionType == id {
			return cube.Range{int(definition.Range[0]), int(definition.Range[1])}, true
		}
	}
	if dimension, ok := world.DimensionByID(int(id)); ok {
		return dimension.Range(), true
	}
	return cube.Range{}, false
}

// ClaimAt ...
func ClaimAt(
	claims map[string]claim.PlayerClaim,
	dimension int32,
	definitions []protocol.DimensionDefinition,
	x, z float32,
) (claim.PlayerClaim, bool) {
	for _, c := range claims {
		if claimDimensionToInt(c.Location.Dimension, definitions) == dimension && claimContains(c, x, z) {
			return c, true
		}
	}
	return claim.PlayerClaim{}, false
}

func claimContains(c claim.PlayerClaim, x, z float32) bool {
	minX := min(c.Location.Pos1.X, c.Location.Pos2.X)
	maxX := max(c.Location.Pos1.X, c.Location.Pos2.X)
	minZ := min(c.Location.Pos1.Z, c.Location.Pos2.Z)
	maxZ := max(c.Location.Pos1.Z, c.Location.Pos2.Z)
	return x >= minX && x <= maxX && z >= minZ && z <= maxZ
}

func claimsAtChunk(
	claims map[string]claim.PlayerClaim,
	dimension int32,
	definitions []protocol.DimensionDefinition,
	chunkPos protocol.ChunkPos,
) []claim.PlayerClaim {
	chunkMinX := float32(chunkPos.X() << 4)
	chunkMinZ := float32(chunkPos.Z() << 4)
	chunkMaxX := chunkMinX + 15
	chunkMaxZ := chunkMinZ + 15
	claimsInChunk := make([]claim.PlayerClaim, 0, 1)
	for _, c := range claims {
		if claimDimensionToInt(c.Location.Dimension, definitions) != dimension {
			continue
		}
		minX := min(c.Location.Pos1.X, c.Location.Pos2.X)
		maxX := max(c.Location.Pos1.X, c.Location.Pos2.X)
		minZ := min(c.Location.Pos1.Z, c.Location.Pos2.Z)
		maxZ := max(c.Location.Pos1.Z, c.Location.Pos2.Z)
		if chunkMinX <= maxX && chunkMaxX >= minX &&
			chunkMinZ <= maxZ && chunkMaxZ >= minZ {
			claimsInChunk = append(claimsInChunk, c)
		}
	}
	return claimsInChunk
}

// blockPosToVec3 ...
func blockPosToVec3(pos protocol.BlockPos) mgl32.Vec3 {
	return mgl32.Vec3{float32(pos.X()), float32(pos.Y()), float32(pos.Z())}
}
