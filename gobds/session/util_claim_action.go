package session

import (
	"slices"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

// ClaimAction ...
type ClaimAction uint8

type claimActionData struct {
	position mgl32.Vec3
	typeID   string
}

// ClaimActor contains only player state needed by claim policy evaluation.
type ClaimActor struct {
	XUID     string
	Operator bool
}

const (
	// ClaimActionRender ...
	ClaimActionRender ClaimAction = iota
	// ClaimActionBlockBreak ...
	ClaimActionBlockBreak
	// ClaimActionBlockPlace ...
	ClaimActionBlockPlace
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
	// ClaimActionItemDrop ...
	ClaimActionItemDrop
)

// ClaimActionPermitted evaluates claim policy without session or network state.
func ClaimActionPermitted(cl claim.PlayerClaim, actor ClaimActor, action ClaimAction, data any) bool {
	if !validClaim(cl) || actor.XUID == "" {
		return true
	}
	if claimOwnerOrTrusted(cl, actor) {
		return true
	}
	switch action {
	case ClaimActionRender:
		return handleClaimActionRender(cl, data)
	case ClaimActionBlockBreak:
		return handleClaimActionInFeature(cl, data, claim.FeatureTypeMineable)
	case ClaimActionBlockPlace:
		return handleClaimActionInFeature(cl, data, claim.FeatureTypeBlockPlaceable)
	case ClaimActionBlockInteract:
		return claimActionBlockInteractPermitted(cl, data)
	case ClaimActionEntityInteract:
		return claimActionEntityInteractPermitted(cl, data)
	case ClaimActionEntityHurt:
		return handleClaimActionInFeature(cl, data, claim.FeatureTypeEntityHurt)
	case ClaimActionItemRelease, ClaimActionItemThrow:
		// ReleaseItem stops item use (food, bows, fishing), while click-air
		// throwables are not item drops. BEH does not claim-filter either.
		return true
	case ClaimActionItemDrop:
		return claimActionItemDropPermitted(cl, data)
	}
	return true
}

func claimActionBlockInteractPermitted(cl claim.PlayerClaim, data any) bool {
	if cl.OwnerXUID == "*" && adminClaimBlockInteractionAllowed(data) {
		return true
	}
	return handleClaimActionInFeature(cl, data, claim.FeatureTypeBlockInteractable)
}

func claimActionEntityInteractPermitted(cl claim.PlayerClaim, data any) bool {
	if cl.OwnerXUID == "*" {
		return true
	}
	return handleClaimActionInFeature(cl, data, claim.FeatureTypeEntityInteractable)
}

func claimActionItemDropPermitted(cl claim.PlayerClaim, data any) bool {
	if cl.OwnerXUID == "*" {
		return true
	}
	return handleClaimActionInFeature(cl, data, claim.FeatureTypeDropItems)
}

func adminClaimBlockInteractionAllowed(data any) bool {
	actionData, ok := claimActionDataFrom(data)
	if !ok {
		return true
	}
	if actionData.typeID == "" {
		return true
	}
	switch actionData.typeID {
	case "pokeb:pc", "pokeb:healing_machine", "pokeb:trade_machine", "minecraft:ender_chest":
		return true
	default:
		return false
	}
}

func (s *Session) claimActionPermitted(action ClaimAction, data any) bool {
	if s.claimFactory == nil {
		return true
	}
	metrics := s.claimFactory.Metrics()
	if !s.claimPrefilter {
		metrics.Action(uint8(action), true)
		return true
	}

	actionData, ok := claimActionDataFrom(data)
	if !ok {
		metrics.Reason(claim.QueryInvalid)
		metrics.Action(uint8(action), true)
		return true
	}
	snapshot, status := s.claimFactory.Snapshot(time.Now())
	if status != claim.QueryReady {
		metrics.Reason(status)
		metrics.Action(uint8(action), true)
		return true
	}
	dimension, ok := claimDimensionFromInt(s.Data().Dimension(), s.GameData().Dimensions)
	if !ok {
		metrics.Reason(claim.QueryUnknownDimension)
		metrics.Action(uint8(action), true)
		return true
	}
	candidates := snapshot.Candidates(dimension, actionData.position.X(), actionData.position.Z())
	metrics.Candidates(len(candidates))
	matched, ambiguous := singleClaimAt(candidates, actionData.position.X(), actionData.position.Z())
	if ambiguous {
		metrics.Reason(claim.QueryOverlap)
		metrics.Action(uint8(action), true)
		return true
	}
	if matched == nil {
		metrics.Action(uint8(action), true)
		return true
	}
	permitted := ClaimActionPermitted(
		*matched,
		ClaimActor{XUID: s.IdentityData().XUID, Operator: s.Data().Operator()},
		action,
		data,
	)
	metrics.Action(uint8(action), permitted)
	return permitted
}

func singleClaimAt(candidates []*claim.PlayerClaim, x, z float32) (*claim.PlayerClaim, bool) {
	var matched *claim.PlayerClaim
	for _, candidate := range candidates {
		if !claimContains(*candidate, x, z) {
			continue
		}
		if matched != nil {
			return nil, true
		}
		matched = candidate
	}
	return matched, false
}

// handleClaimActionRender ...
func handleClaimActionRender(cl claim.PlayerClaim, data any) (permitted bool) {
	position, ok := data.(mgl32.Vec3)
	if !ok {
		return true
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
func handleClaimActionInFeature(cl claim.PlayerClaim, data any, featureType string) bool {
	actionData, ok := claimActionDataFrom(data)
	if !ok {
		return true
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
	case claim.FeatureTypeDropItems:
		return feature.ItemTypeIDs == nil || slices.Contains(feature.ItemTypeIDs, typeID)
	default:
		return true
	}
}

// claimOwnerOrTrusted ...
func claimOwnerOrTrusted(cl claim.PlayerClaim, actor ClaimActor) bool {
	return cl.OwnerXUID == actor.XUID || slices.Contains(cl.TrustedXUIDS, actor.XUID) || actor.Operator
}

func validClaim(cl claim.PlayerClaim) bool {
	return cl.ID != "" && cl.OwnerXUID != "" && cl.Location.Dimension != ""
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
