package claim

import (
	"math"
	"testing"
	"time"
)

func TestBuildSnapshotIndexesCanonicalCellsAndClones(t *testing.T) {
	source := map[string]PlayerClaim{
		"one": {
			OwnerXUID: "owner",
			Location: Location{
				Dimension: "OVERWORLD",
				Pos1:      Vector2{X: -17, Z: -1},
				Pos2:      Vector2{X: 16, Z: 16},
			},
			Features: []Feature{{
				Type:         FeatureTypeMineable,
				BlockTypeIDs: []string{"minecraft:stone"},
			}},
			TrustedXUIDS: []string{"trusted"},
		},
	}
	now := time.Unix(100, 0)
	snapshot, err := BuildSnapshot(source, 7, now)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != 7 || snapshot.ClaimCount != 1 || snapshot.CellCount != 12 {
		t.Fatalf("unexpected metadata: %+v", snapshot)
	}
	if candidates := snapshot.Candidates("minecraft:overworld", -16, 0); len(candidates) != 1 {
		t.Fatalf("expected indexed claim, got %d candidates", len(candidates))
	}
	source["one"].Features[0].BlockTypeIDs[0] = "minecraft:dirt"
	source["one"].TrustedXUIDS[0] = "changed"
	if snapshot.claims[0].Features[0].BlockTypeIDs[0] != "minecraft:stone" ||
		snapshot.claims[0].TrustedXUIDS[0] != "trusted" {
		t.Fatal("snapshot retained mutable source slices")
	}
}

func TestBuildSnapshotAcceptsPickupItemsWithoutEnforcement(t *testing.T) {
	snapshot, err := BuildSnapshot(map[string]PlayerClaim{
		"one": {
			ID:        "one",
			OwnerXUID: "owner",
			Location: Location{
				Dimension: "minecraft:overworld",
				Pos1:      Vector2{X: 0, Z: 0},
				Pos2:      Vector2{X: 15, Z: 15},
			},
			Features: []Feature{{
				Type:        FeatureTypePickupItems,
				ItemTypeIDs: []string{"minecraft:apple"},
			}},
		},
	}, 1, time.Now())
	if err != nil {
		t.Fatalf("pickupItems must be allowed for BEH parity: %v", err)
	}
	if len(snapshot.claims) != 1 || snapshot.claims[0].Features[0].Type != FeatureTypePickupItems {
		t.Fatalf("unexpected snapshot: %+v", snapshot.claims)
	}
}

func TestBuildSnapshotRejectsInvalidDataAtomically(t *testing.T) {
	_, err := BuildSnapshot(map[string]PlayerClaim{
		"bad": {
			OwnerXUID: "owner",
			Location: Location{
				Dimension: "unknown",
				Pos1:      Vector2{X: float32(math.NaN())},
			},
		},
	}, 1, time.Now())
	if err == nil {
		t.Fatal("invalid claim must reject entire snapshot")
	}
}

func TestSnapshotCellsExposeOverlapWithoutChoosingWinner(t *testing.T) {
	claims := map[string]PlayerClaim{
		"one": testSnapshotClaim("one", 0, 15),
		"two": testSnapshotClaim("two", 8, 20),
	}
	snapshot, err := BuildSnapshot(claims, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if candidates := snapshot.Candidates("minecraft:overworld", 9, 9); len(candidates) != 2 {
		t.Fatalf("overlap must retain both candidates, got %d", len(candidates))
	}
}

func testSnapshotClaim(id string, minPosition, maxPosition float32) PlayerClaim {
	return PlayerClaim{
		ID:        id,
		OwnerXUID: "owner",
		Location: Location{
			Dimension: "minecraft:overworld",
			Pos1:      Vector2{X: minPosition, Z: minPosition},
			Pos2:      Vector2{X: maxPosition, Z: maxPosition},
		},
	}
}
