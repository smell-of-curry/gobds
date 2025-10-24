package session

import (
	"slices"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

// ClaimAction ...
type ClaimAction uint8

const (
	ClaimActionRender ClaimAction = iota
	ClaimActionBlockInteract
	ClaimActionItemRelease
	ClaimActionItemThrow
)

// ClaimActionPermitted ...
func ClaimActionPermitted(cl claim.PlayerClaim, s *Session, action ClaimAction, data any) bool {
	switch action {
	case ClaimActionRender:
		return handleClaimActionRender(cl, s, data)
	case ClaimActionBlockInteract:
		return handleClaimActionBlockInteract(cl, s, data)
	case ClaimActionItemRelease:
		return handleClaimActionItemRelease(cl, s, data)
	case ClaimActionItemThrow:
		return handleClaimActionItemThrow(cl, s, data)
	}
	return true
}

// handleClaimActionRender ...
func handleClaimActionRender(cl claim.PlayerClaim, s *Session, data any) (permitted bool) {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	switch data := data.(type) {
	case protocol.ChunkPos:
		for _, feature := range cl.Features {
			pos1, pos2 := feature.ToLocation, feature.FromLocation
			if insideChunkPosition(data, pos1, pos2) {
				return true
			}
		}
	case mgl32.Vec3:
		for _, feature := range cl.Features {
			pos1, pos2 := feature.ToLocation, feature.FromLocation
			if insideVector3(data, pos1, pos2) {
				return true
			}
		}
	}
	return false
}

// handleClaimActionBlockInteract ...
func handleClaimActionBlockInteract(cl claim.PlayerClaim, s *Session, data any) (permitted bool) {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	transactionPosition := data.(mgl32.Vec3)
	for _, feature := range cl.Features {
		switch feature.Type {
		case claim.FeatureTypeBlockIntractable:
			pos1, pos2 := feature.ToLocation, feature.FromLocation
			if insideVector3(transactionPosition, pos1, pos2) {
				return true
			}
		}
	}
	return false
}

// handleClaimActionItemRelease ...
func handleClaimActionItemRelease(cl claim.PlayerClaim, s *Session, _ any) (permitted bool) {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	return false
}

// handleClaimActionItemThrow ...
func handleClaimActionItemThrow(cl claim.PlayerClaim, s *Session, _ any) (permitted bool) {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	return false
}

// claimOwnerOrTrusted ...
func claimOwnerOrTrusted(cl claim.PlayerClaim, s *Session) bool {
	clientXUID := s.IdentityData().XUID
	return cl.ID == "" || cl.OwnerXUID == "*" ||
		cl.OwnerXUID == clientXUID || slices.Contains(cl.TrustedXUIDS, clientXUID)
}

// insideChunkPosition ...
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

// insideVector3 ...
func insideVector3(vec3 mgl32.Vec3, pos1, pos2 claim.Vector2) bool {
	minX := min(pos1.X, pos2.X)
	maxX := max(pos1.X, pos2.X)
	minZ := min(pos1.Z, pos2.Z)
	maxZ := max(pos1.Z, pos2.Z)

	return vec3.X() >= minX && vec3.X() <= maxX &&
		vec3.Z() >= minZ && vec3.Z() <= maxZ
}
