package session

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

func benchClaims() []*claim.PlayerClaim {
	return []*claim.PlayerClaim{{
		ID:        "home",
		OwnerXUID: "owner",
		Location: claim.Location{
			Dimension: "minecraft:overworld",
			Pos1:      claim.Vector2{X: 0, Z: 0},
			Pos2:      claim.Vector2{X: 15, Z: 15},
		},
		Features: []claim.Feature{{
			Type:         claim.FeatureTypeMineable,
			FromLocation: &claim.Vector3{X: 0, Y: 0, Z: 0},
			ToLocation:   &claim.Vector3{X: 3, Y: 0, Z: 3},
		}},
		TrustedXUIDS: []string{"trusted-a", "trusted-b"},
	}}
}

// legacyFloorDenyMask mirrors the pre-optimization per-cell path:
// ClaimActionPermitted by value + interface{} position boxing + 256 decisions.
func legacyFloorDenyMask(claims []*claim.PlayerClaim, actor ClaimActor, chunkBlockX, chunkBlockZ int32, mask *[16]uint16) (forwarded, denied int) {
	*mask = [16]uint16{}
	for z := uint8(0); z < 16; z++ {
		var row uint16
		for x := uint8(0); x < 16; x++ {
			position := mgl32.Vec3{
				float32(chunkBlockX + int32(x)),
				0,
				float32(chunkBlockZ + int32(z)),
			}
			matched, ambiguous := singleClaimAt(claims, position.X(), position.Z())
			deniedCell := !ambiguous && matched != nil &&
				!ClaimActionPermitted(*matched, actor, ClaimActionRender, position)
			if deniedCell {
				row |= 1 << x
				denied++
				continue
			}
			forwarded++
		}
		mask[z] = row
	}
	return forwarded, denied
}

func BenchmarkFloorDenyMask_Legacy(b *testing.B) {
	claims := benchClaims()
	actor := ClaimActor{XUID: "stranger"}
	var mask [16]uint16
	b.ReportAllocs()
	for b.Loop() {
		legacyFloorDenyMask(claims, actor, 0, 0, &mask)
	}
}

func BenchmarkFloorDenyMask(b *testing.B) {
	claims := benchClaims()
	actor := ClaimActor{XUID: "stranger"}
	var mask [16]uint16
	b.ReportAllocs()
	for b.Loop() {
		floorDenyMask(claims, actor, 0, 0, &mask)
	}
}

func BenchmarkFloorDenyMask_Owner(b *testing.B) {
	claims := benchClaims()
	actor := ClaimActor{XUID: "owner"}
	var mask [16]uint16
	b.ReportAllocs()
	for b.Loop() {
		floorDenyMask(claims, actor, 0, 0, &mask)
	}
}

func BenchmarkFloorDenyMask_FullDeny(b *testing.B) {
	claims := []*claim.PlayerClaim{{
		ID:        "home",
		OwnerXUID: "owner",
		Location: claim.Location{
			Dimension: "minecraft:overworld",
			Pos1:      claim.Vector2{X: 0, Z: 0},
			Pos2:      claim.Vector2{X: 15, Z: 15},
		},
	}}
	actor := ClaimActor{XUID: "stranger"}
	var mask [16]uint16
	b.ReportAllocs()
	for b.Loop() {
		floorDenyMask(claims, actor, 0, 0, &mask)
	}
}

func BenchmarkRenderMetrics_PerCell(b *testing.B) {
	metrics := claim.NewMetrics("BENCH")
	b.ReportAllocs()
	for b.Loop() {
		for i := 0; i < 256; i++ {
			metrics.Action(uint8(ClaimActionRender), i&1 == 0)
		}
	}
}

func BenchmarkRenderMetrics_Batched(b *testing.B) {
	metrics := claim.NewMetrics("BENCH")
	b.ReportAllocs()
	for b.Loop() {
		metrics.ActionN(uint8(ClaimActionRender), 128, 128)
	}
}
