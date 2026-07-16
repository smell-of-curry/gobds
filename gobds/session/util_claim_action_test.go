package session

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/service"
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
			ID:        "arena",
			OwnerXUID: "owner",
			Location: claim.Location{
				Dimension: "pokeb:battle_arena",
				Pos1:      claim.Vector2{X: 10, Z: 20},
				Pos2:      claim.Vector2{X: 20, Z: 30},
			},
		},
	}
	dimension, ok := claimDimensionFromInt(1000, definitions)
	if !ok || dimension != "pokeb:battle_arena" {
		t.Fatal("data-driven dimension was not resolved")
	}
	claims["second"] = claim.PlayerClaim{
		ID:        "second",
		OwnerXUID: "owner",
		Location: claim.Location{
			Dimension: "pokeb:battle_arena",
			Pos1:      claim.Vector2{X: 1, Z: 21},
			Pos2:      claim.Vector2{X: 2, Z: 22},
		},
	}
	snapshot, err := claim.BuildSnapshot(claims, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(snapshot.Candidates(dimension, 0, 16)); got != 2 {
		t.Fatalf("expected both claims intersecting chunk, got %d", got)
	}
	dimensionRange, ok := dimensionRangeByID(1000, definitions)
	if !ok || dimensionRange.Min() != -64 || dimensionRange.Max() != 320 {
		t.Fatalf("unexpected data-driven dimension range: %v, found=%v", dimensionRange, ok)
	}
}

func TestClaimPolicyIdentityAndFailOpen(t *testing.T) {
	cl := testClaim()
	position := mgl32.Vec3{5, 5, 5}
	for name, actor := range map[string]ClaimActor{
		"owner":    {XUID: "owner"},
		"trusted":  {XUID: "trusted"},
		"operator": {XUID: "stranger", Operator: true},
	} {
		t.Run(name, func(t *testing.T) {
			if !ClaimActionPermitted(cl, actor, ClaimActionBlockBreak, position) {
				t.Fatal("privileged actor should be permitted")
			}
		})
	}

	if !ClaimActionPermitted(cl, ClaimActor{}, ClaimActionBlockBreak, position) {
		t.Fatal("missing XUID must fail open")
	}
	if !ClaimActionPermitted(cl, ClaimActor{XUID: "stranger"}, ClaimAction(255), position) {
		t.Fatal("unsupported action must pass through")
	}
	if !ClaimActionPermitted(cl, ClaimActor{XUID: "stranger"}, ClaimActionBlockBreak, "malformed") {
		t.Fatal("malformed action data must fail open")
	}
	for name, malformed := range map[string]claim.PlayerClaim{
		"empty":             {},
		"missing id":        {OwnerXUID: "owner", Location: cl.Location},
		"missing owner":     {ID: "claim", Location: cl.Location},
		"missing dimension": {ID: "claim", OwnerXUID: "owner"},
	} {
		t.Run(name, func(t *testing.T) {
			if !ClaimActionPermitted(malformed, ClaimActor{XUID: "stranger"}, ClaimActionBlockBreak, position) {
				t.Fatal("malformed claim must fail open")
			}
		})
	}
}

func TestClaimActionObserveOnlyRecordsForwardedWithoutLookup(t *testing.T) {
	factory := claim.NewFactory(
		service.Config{},
		"TEST",
		time.Minute,
		time.Minute,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	s := &Session{claimFactory: factory, claimPrefilter: false}
	if !s.claimActionPermitted(ClaimActionBlockBreak, "malformed") {
		t.Fatal("disabled prefilter must preserve forwarding")
	}

	var output bytes.Buffer
	factory.Metrics().WriteDelta(&output, time.Minute, nil)
	var record struct {
		Seen       []uint64 `json:"seen"`
		Forwarded  []uint64 `json:"forwarded"`
		Reasons    []uint64 `json:"reasons"`
		Candidates uint64   `json:"candidates"`
	}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Seen[ClaimActionBlockBreak] != 1 || record.Forwarded[ClaimActionBlockBreak] != 1 {
		t.Fatalf("observe-only action not recorded as forwarded: %+v", record)
	}
	if record.Candidates != 0 {
		t.Fatalf("observe-only action performed claim lookup: %+v", record)
	}
	for _, count := range record.Reasons {
		if count != 0 {
			t.Fatalf("observe-only action evaluated claim status: %+v", record)
		}
	}
}

func TestClaimPolicyFeatureParity(t *testing.T) {
	position := mgl32.Vec3{5, 5, 5}
	actor := ClaimActor{XUID: "stranger"}
	tests := []struct {
		name    string
		feature claim.Feature
		action  ClaimAction
		typeID  string
	}{
		{"mineable", claim.Feature{Type: claim.FeatureTypeMineable}, ClaimActionBlockBreak, "minecraft:stone"},
		{"placeable", claim.Feature{Type: claim.FeatureTypeBlockPlaceable}, ClaimActionBlockPlace, "minecraft:stone"},
		{"block interactable", claim.Feature{Type: claim.FeatureTypeBlockInteractable}, ClaimActionBlockInteract, "minecraft:chest"},
		{"entity interactable", claim.Feature{Type: claim.FeatureTypeEntityInteractable}, ClaimActionEntityInteract, "minecraft:armor_stand"},
		{"entity hurt", claim.Feature{Type: claim.FeatureTypeEntityHurt}, ClaimActionEntityHurt, "minecraft:zombie"},
		{"drop items", claim.Feature{Type: claim.FeatureTypeDropItems}, ClaimActionItemDrop, "minecraft:stone"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := testClaim()
			cl.Features = []claim.Feature{test.feature}
			data := claimActionData{position: position, typeID: test.typeID}
			if !ClaimActionPermitted(cl, actor, test.action, data) {
				t.Fatal("matching feature should permit action")
			}
		})
	}
}

func TestClaimPolicyFiltersBoundsAndAdminExceptions(t *testing.T) {
	actor := ClaimActor{XUID: "stranger"}
	cl := testClaim()
	from := claim.Vector3{X: 8, Y: 10, Z: 8}
	to := claim.Vector3{X: 2, Y: 2, Z: 2}
	cl.Features = []claim.Feature{{
		Type:         claim.FeatureTypeMineable,
		FromLocation: &from,
		ToLocation:   &to,
		BlockTypeIDs: []string{"minecraft:stone"},
	}}
	if !ClaimActionPermitted(
		cl,
		actor,
		ClaimActionBlockBreak,
		claimActionData{position: mgl32.Vec3{5, 5, 5}, typeID: "minecraft:stone"},
	) {
		t.Fatal("reversed feature bounds and matching filter should permit")
	}
	if ClaimActionPermitted(
		cl,
		actor,
		ClaimActionBlockBreak,
		claimActionData{position: mgl32.Vec3{5, 5, 5}, typeID: "minecraft:dirt"},
	) {
		t.Fatal("non-matching block filter should deny")
	}

	admin := testClaim()
	admin.OwnerXUID = "*"
	if !ClaimActionPermitted(admin, actor, ClaimActionEntityInteract, claimActionData{}) {
		t.Fatal("admin claims should allow entity interaction")
	}
	if !ClaimActionPermitted(admin, actor, ClaimActionItemDrop, claimActionData{}) {
		t.Fatal("admin claims should allow item drops")
	}
	if !ClaimActionPermitted(
		admin,
		actor,
		ClaimActionBlockInteract,
		claimActionData{typeID: "minecraft:ender_chest"},
	) {
		t.Fatal("admin claim block exception should be permitted")
	}
}

func TestClaimPolicyItemFiltersAndRelease(t *testing.T) {
	cl := testClaim()
	cl.Features = []claim.Feature{{
		Type:        claim.FeatureTypeDropItems,
		ItemTypeIDs: []string{"minecraft:stone"},
	}}
	actor := ClaimActor{XUID: "stranger"}
	position := mgl32.Vec3{5, 5, 5}
	if !ClaimActionPermitted(
		cl,
		actor,
		ClaimActionItemDrop,
		claimActionData{position: position, typeID: "minecraft:stone"},
	) {
		t.Fatal("listed item should be droppable")
	}
	if ClaimActionPermitted(
		cl,
		actor,
		ClaimActionItemDrop,
		claimActionData{position: position, typeID: "minecraft:dirt"},
	) {
		t.Fatal("unlisted item should not be droppable")
	}
	if !ClaimActionPermitted(cl, actor, ClaimActionItemRelease, position) {
		t.Fatal("item-use release must fail open")
	}
	if !ClaimActionPermitted(
		cl,
		actor,
		ClaimActionItemThrow,
		claimActionData{position: position, typeID: "minecraft:snowball"},
	) {
		t.Fatal("click-air throwable use must fail open")
	}
}

func TestClaimsAtHandlesReversedAndOverlappingClaims(t *testing.T) {
	first, second := testClaim(), testClaim()
	first.Location.Pos1, first.Location.Pos2 = first.Location.Pos2, first.Location.Pos1
	second.ID = "second"
	second.Location.Pos1 = claim.Vector2{X: 5, Z: 5}
	second.Location.Pos2 = claim.Vector2{X: 15, Z: 15}
	claims := map[string]claim.PlayerClaim{"first": first, "second": second}
	snapshot, err := claim.BuildSnapshot(claims, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	candidates := snapshot.Candidates("minecraft:overworld", 7, 7)
	if len(candidates) != 2 {
		t.Fatalf("expected both overlapping claims, got %d", len(candidates))
	}
	if matched, ambiguous := singleClaimAt(candidates, 7, 7); matched != nil || !ambiguous {
		t.Fatal("overlap must be ambiguous so proxy can pass through")
	}
	first.OwnerXUID = "stranger"
	if claimActionsPermitted(
		[]claim.PlayerClaim{first, second},
		ClaimActor{XUID: "stranger"},
		ClaimActionBlockBreak,
		mgl32.Vec3{7, 0, 7},
	) {
		t.Fatal("one overlapping denied claim must deny action")
	}
}

func claimActionsPermitted(claims []claim.PlayerClaim, actor ClaimActor, action ClaimAction, data any) bool {
	for _, cl := range claims {
		if !ClaimActionPermitted(cl, actor, action, data) {
			return false
		}
	}
	return true
}

func testClaim() claim.PlayerClaim {
	return claim.PlayerClaim{
		ID:           "claim",
		OwnerXUID:    "owner",
		TrustedXUIDS: []string{"trusted"},
		Location: claim.Location{
			Dimension: "minecraft:overworld",
			Pos1:      claim.Vector2{X: 0, Z: 0},
			Pos2:      claim.Vector2{X: 10, Z: 10},
		},
	}
}
