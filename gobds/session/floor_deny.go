package session

import "github.com/smell-of-curry/gobds/gobds/claim"

// floorDenyMask paints a 16×16 floor deny mask for one chunk column.
//
// mask[z] bit x is set when that block column should receive a deny block at
// the dimension floor. The function is allocation-free: no maps, no interface{}
// boxing, no claim struct copies.
//
// Overlapping claims fail open (no deny), matching singleClaimAt semantics.
func floorDenyMask(
	claims []*claim.PlayerClaim,
	actor ClaimActor,
	chunkBlockX, chunkBlockZ int32,
	mask *[16]uint16,
) (forwarded, denied int) {
	*mask = [16]uint16{}
	if len(claims) == 0 {
		return 256, 0
	}
	// Hot path: one claim covering the whole chunk. Prod denies ~86% of floor
	// cells; skip the 256-cell walk when the verdict is uniform.
	if len(claims) == 1 {
		cl := claims[0]
		if claimCoversChunk(cl, chunkBlockX, chunkBlockZ) {
			if cl.ID == "" || cl.OwnerXUID == "" || actor.XUID == "" || claimOwnerOrTrustedPtr(cl, actor) {
				return 256, 0
			}
			if !claimHasPartialRenderOpen(cl) {
				for z := 0; z < 16; z++ {
					mask[z] = 0xFFFF
				}
				return 0, 256
			}
		}
	}
	for z := uint8(0); z < 16; z++ {
		wz := float32(chunkBlockZ + int32(z))
		var row uint16
		for x := uint8(0); x < 16; x++ {
			wx := float32(chunkBlockX + int32(x))
			matched, ambiguous := singleClaimAt(claims, wx, wz)
			if ambiguous || matched == nil || renderPermitted(matched, actor, wx, wz) {
				forwarded++
				continue
			}
			row |= 1 << x
			denied++
		}
		mask[z] = row
	}
	return forwarded, denied
}

func claimCoversChunk(cl *claim.PlayerClaim, chunkBlockX, chunkBlockZ int32) bool {
	minX := min(cl.Location.Pos1.X, cl.Location.Pos2.X)
	maxX := max(cl.Location.Pos1.X, cl.Location.Pos2.X)
	minZ := min(cl.Location.Pos1.Z, cl.Location.Pos2.Z)
	maxZ := max(cl.Location.Pos1.Z, cl.Location.Pos2.Z)
	return minX <= float32(chunkBlockX) && maxX >= float32(chunkBlockX+15) &&
		minZ <= float32(chunkBlockZ) && maxZ >= float32(chunkBlockZ+15)
}

// claimHasPartialRenderOpen reports mineable/placeable features that may carve
// only part of the claim — forces the per-cell walk.
func claimHasPartialRenderOpen(cl *claim.PlayerClaim) bool {
	for i := range cl.Features {
		feature := &cl.Features[i]
		if feature.Type != claim.FeatureTypeMineable && feature.Type != claim.FeatureTypeBlockPlaceable {
			continue
		}
		return true
	}
	return false
}

// renderPermitted reports whether a stranger still sees through the claim floor
// at (x,z) — owner/trust/op, or a mineable/placeable feature covering that XZ.
func renderPermitted(cl *claim.PlayerClaim, actor ClaimActor, x, z float32) bool {
	if cl == nil || cl.ID == "" || cl.OwnerXUID == "" {
		return true
	}
	if actor.XUID == "" || claimOwnerOrTrustedPtr(cl, actor) {
		return true
	}
	for i := range cl.Features {
		feature := &cl.Features[i]
		if feature.Type != claim.FeatureTypeMineable && feature.Type != claim.FeatureTypeBlockPlaceable {
			continue
		}
		if insideFeatureXZ(cl, feature, x, z) {
			return true
		}
	}
	return false
}

func claimOwnerOrTrustedPtr(cl *claim.PlayerClaim, actor ClaimActor) bool {
	if cl.OwnerXUID == actor.XUID || actor.Operator {
		return true
	}
	for _, trusted := range cl.TrustedXUIDS {
		if trusted == actor.XUID {
			return true
		}
	}
	return false
}

func insideFeatureXZ(cl *claim.PlayerClaim, feature *claim.Feature, x, z float32) bool {
	from := cl.Location.Pos1
	to := cl.Location.Pos2
	if feature.FromLocation != nil {
		from = claim.Vector2{X: feature.FromLocation.X, Z: feature.FromLocation.Z}
	}
	if feature.ToLocation != nil {
		to = claim.Vector2{X: feature.ToLocation.X, Z: feature.ToLocation.Z}
	}
	minX := min(from.X, to.X)
	maxX := max(from.X, to.X)
	minZ := min(from.Z, to.Z)
	maxZ := max(from.Z, to.Z)
	return x >= minX && x <= maxX && z >= minZ && z <= maxZ
}
