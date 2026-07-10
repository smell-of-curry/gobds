package session

import (
	"slices"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

// ClaimAction ...
type ClaimAction uint8

type claimActionData struct {
	position mgl32.Vec3
	typeID   string
}

const (
	// ClaimActionRender ...
	ClaimActionRender ClaimAction = iota
	// ClaimActionBlockBreak ...
	ClaimActionBlockBreak
	// ClaimActionBlockInteract ...
	ClaimActionBlockInteract
	// ClaimActionEntityInteract ...
	ClaimActionEntityInteract
	// ClaimActionEntityHurt ...
	ClaimActionEntityHurt
	// ClaimActionItemRelease ...
	ClaimActionItemRelease
	// ClaimActionItemThrow ...
	ClaimActionItemThrow
)

// ClaimActionPermitted ...
func ClaimActionPermitted(cl claim.PlayerClaim, s *Session, action ClaimAction, data any) bool {
	switch action {
	case ClaimActionRender:
		return handleClaimActionRender(cl, s, data)
	case ClaimActionBlockBreak:
		return handleClaimActionInFeature(cl, s, data, claim.FeatureTypeMineable)
	case ClaimActionBlockInteract:
		return handleClaimActionInFeature(cl, s, data, claim.FeatureTypeBlockInteractable)
	case ClaimActionEntityInteract:
		if cl.OwnerXUID == "*" {
			return true
		}
		return handleClaimActionInFeature(cl, s, data, claim.FeatureTypeEntityInteractable)
	case ClaimActionEntityHurt:
		return handleClaimActionInFeature(cl, s, data, claim.FeatureTypeEntityHurt)
	case ClaimActionItemRelease:
		return handleClaimActionItemRelease(cl, s, data)
	case ClaimActionItemThrow:
		return handleClaimActionItemThrow(cl, s, data)
	}
	return false
}

// handleClaimActionRender ...
func handleClaimActionRender(cl claim.PlayerClaim, s *Session, data any) (permitted bool) {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	position, ok := data.(mgl32.Vec3)
	if !ok {
		return false
	}
	for _, feature := range cl.Features {
		if !featureRemovesDenyBlock(feature) {
			continue
		}
		pos1, pos2 := featureBounds(cl, feature)
		if insideVector3(position, pos1, pos2) {
			return true
		}
	}
	return false
}

// featureRemovesDenyBlock ...
func featureRemovesDenyBlock(feature claim.Feature) bool {
	return feature.Type == claim.FeatureTypeMineable || feature.Type == claim.FeatureTypeBlockPlaceable
}

// handleClaimActionInFeature ...
func handleClaimActionInFeature(cl claim.PlayerClaim, s *Session, data any, featureType string) bool {
	if claimOwnerOrTrusted(cl, s) {
		return true
	}
	actionData, ok := claimActionDataFrom(data)
	if !ok {
		return false
	}
	for _, feature := range cl.Features {
		if feature.Type != featureType {
			continue
		}
		if insideFeature(cl, feature, actionData.position) && featureAllowsType(feature, featureType, actionData.typeID) {
			return true
		}
	}
	return false
}

func claimActionDataFrom(data any) (claimActionData, bool) {
	switch data := data.(type) {
	case mgl32.Vec3:
		return claimActionData{position: data}, true
	case claimActionData:
		return data, true
	default:
		return claimActionData{}, false
	}
}

func featureAllowsType(feature claim.Feature, featureType, typeID string) bool {
	if typeID == "" {
		return true
	}
	switch featureType {
	case claim.FeatureTypeMineable, claim.FeatureTypeBlockPlaceable, claim.FeatureTypeBlockInteractable:
		return feature.BlockTypeIDs == nil || slices.Contains(feature.BlockTypeIDs, typeID)
	case claim.FeatureTypeEntityInteractable, claim.FeatureTypeEntityHurt:
		return feature.EntityTypeIDs == nil || slices.Contains(feature.EntityTypeIDs, typeID)
	default:
		return false
	}
}

// handleClaimActionItemRelease ...
func handleClaimActionItemRelease(cl claim.PlayerClaim, s *Session, _ any) (permitted bool) {
	return cl.OwnerXUID == "*" || claimOwnerOrTrusted(cl, s)
}

// handleClaimActionItemThrow ...
func handleClaimActionItemThrow(cl claim.PlayerClaim, s *Session, _ any) (permitted bool) {
	return cl.OwnerXUID == "*" || claimOwnerOrTrusted(cl, s)
}

// claimOwnerOrTrusted ...
func claimOwnerOrTrusted(cl claim.PlayerClaim, s *Session) bool {
	clientXUID := s.IdentityData().XUID
	return cl.OwnerXUID == clientXUID || slices.Contains(cl.TrustedXUIDS, clientXUID) ||
		s.Data().Operator()
}

// featureBounds ...
func featureBounds(cl claim.PlayerClaim, feature claim.Feature) (claim.Vector2, claim.Vector2) {
	from := cl.Location.Pos1
	to := cl.Location.Pos2
	if feature.FromLocation != nil {
		from = claim.Vector2{X: feature.FromLocation.X, Z: feature.FromLocation.Z}
	}
	if feature.ToLocation != nil {
		to = claim.Vector2{X: feature.ToLocation.X, Z: feature.ToLocation.Z}
	}
	return from, to
}

// insideFeature ...
func insideFeature(cl claim.PlayerClaim, feature claim.Feature, position mgl32.Vec3) bool {
	from, to := featureBounds(cl, feature)
	if !insideVector3(position, from, to) {
		return false
	}
	if feature.FromLocation != nil && feature.ToLocation != nil {
		minY := min(feature.FromLocation.Y, feature.ToLocation.Y)
		maxY := max(feature.FromLocation.Y, feature.ToLocation.Y)
		return position.Y() >= minY && position.Y() <= maxY
	}
	if feature.FromLocation != nil {
		return position.Y() >= feature.FromLocation.Y
	}
	if feature.ToLocation != nil {
		return position.Y() <= feature.ToLocation.Y
	}
	return true
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
