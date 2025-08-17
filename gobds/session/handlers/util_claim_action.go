package handlers

import (
	"slices"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

type ClaimAction uint8

const (
	ClaimActionRender ClaimAction = iota
)

// ClaimActionPermitted ...
func ClaimActionPermitted(cl claim.PlayerClaim, c interceptor.Client, action ClaimAction, data any) bool {
	switch action {
	case ClaimActionRender:
		return handleClaimActionRender(cl, c, data)
	}
	return true
}

// handleClaimActionRender ...
func handleClaimActionRender(cl claim.PlayerClaim, c interceptor.Client, data any) bool {
	clientXUID := c.IdentityData().XUID
	if cl.ID == "" || cl.OwnerXUID == "*" ||
		cl.OwnerXUID == clientXUID || slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return true
	}
	chunkPos := data.(protocol.ChunkPos)
	for _, feature := range cl.Features {
		pos1, pos2 := feature.ToLocation, feature.FromLocation
		if insideChunkPosition(chunkPos, pos1, pos2) {
			return true
		}
	}
	return false
}

// ClaimAt ...
func insideChunkPosition(chunkPos protocol.ChunkPos, pos1, pos2 claim.Vector2) bool {
	chunkMinX := float32(chunkPos.X() << 4)
	chunkMinZ := float32(chunkPos.Z() << 4)
	chunkMaxX := chunkMinX + 15
	chunkMaxZ := chunkMinZ + 15

	minX := min(pos1.X, pos2.X)
	maxX := max(pos1.X, pos2.X)
	minZ := min(pos1.Z, pos2.Z)
	maxZ := max(pos1.Z, pos2.Z)

	return chunkMinX <= maxX && chunkMaxX >= minX &&
		chunkMinZ <= maxZ && chunkMaxZ >= minZ
}
