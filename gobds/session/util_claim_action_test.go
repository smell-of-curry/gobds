package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

func TestInsideFeatureUsesClaimDefaultsAndYBounds(t *testing.T) {
	cl := claim.PlayerClaim{Location: claim.Location{
		Pos1: claim.Vector2{X: 10, Z: 20},
		Pos2: claim.Vector2{X: 20, Z: 30},
	}}
	if !insideFeature(cl, claim.Feature{}, mgl32.Vec3{15, -64, 25}) {
		t.Fatal("feature without bounds should use full claim column")
	}

	from := claim.Vector3{X: 12, Y: 5, Z: 22}
	to := claim.Vector3{X: 18, Y: 10, Z: 28}
	feature := claim.Feature{FromLocation: &from, ToLocation: &to}
	if !insideFeature(cl, feature, mgl32.Vec3{15, 7, 25}) {
		t.Fatal("position inside explicit feature bounds should be permitted")
	}
	if insideFeature(cl, feature, mgl32.Vec3{15, 11, 25}) {
		t.Fatal("position above explicit feature bounds should be denied")
	}
}

func TestFeatureAllowsType(t *testing.T) {
	filtered := claim.Feature{BlockTypeIDs: []string{"minecraft:stone"}}
	if !featureAllowsType(filtered, claim.FeatureTypeMineable, "minecraft:stone") {
		t.Fatal("listed block type should be permitted")
	}
	if featureAllowsType(filtered, claim.FeatureTypeMineable, "minecraft:dirt") {
		t.Fatal("unlisted block type should be denied")
	}
	if !featureAllowsType(claim.Feature{}, claim.FeatureTypeMineable, "minecraft:dirt") {
		t.Fatal("missing type filter should permit every block type")
	}
	if featureAllowsType(claim.Feature{BlockTypeIDs: []string{}}, claim.FeatureTypeMineable, "minecraft:stone") {
		t.Fatal("empty type filter should permit no block types")
	}
}

func TestSetupRuntimeIDsSupportsBothNetworkModes(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	SetupRuntimeIDs()
	for _, hashed := range []bool{false, true} {
		block, ok := blockByRuntimeID(denyBlockRuntimeID(hashed), hashed)
		if !ok {
			t.Fatalf("deny block not found with hashed IDs set to %v", hashed)
		}
		if _, ok := block.(gblock.Deny); !ok {
			t.Fatalf("deny runtime ID resolved to %T with hashed IDs set to %v", block, hashed)
		}
	}
}

func TestClaimAtSupportsDataDrivenDimensions(t *testing.T) {
	definitions := []protocol.DimensionDefinition{{
		Name:          "pokeb:battle_arena",
		Range:         [2]int32{-64, 320},
		DimensionType: 1000,
	}}
	claims := map[string]claim.PlayerClaim{
		"arena": {
			Location: claim.Location{
				Dimension: "pokeb:battle_arena",
				Pos1:      claim.Vector2{X: 10, Z: 20},
				Pos2:      claim.Vector2{X: 20, Z: 30},
			},
		},
	}
	if _, ok := ClaimAt(claims, 1000, definitions, 15, 25); !ok {
		t.Fatal("claim in data-driven dimension was not found")
	}
	claims["second"] = claim.PlayerClaim{Location: claim.Location{
		Dimension: "pokeb:battle_arena",
		Pos1:      claim.Vector2{X: 1, Z: 21},
		Pos2:      claim.Vector2{X: 2, Z: 22},
	}}
	if got := len(claimsAtChunk(claims, 1000, definitions, protocol.ChunkPos{0, 1})); got != 2 {
		t.Fatalf("expected both claims intersecting chunk, got %d", got)
	}
	dimensionRange, ok := dimensionRangeByID(1000, definitions)
	if !ok || dimensionRange.Min() != -64 || dimensionRange.Max() != 320 {
		t.Fatalf("unexpected data-driven dimension range: %v, found=%v", dimensionRange, ok)
	}
}
