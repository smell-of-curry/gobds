package session

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

func TestFloorDenyMaskMatchesLegacyRenderDecisions(t *testing.T) {
	claims := []*claim.PlayerClaim{
		{
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
		},
	}
	actor := ClaimActor{XUID: "stranger"}
	var mask [16]uint16
	forwarded, denied := floorDenyMask(claims, actor, 0, 0, &mask)
	if forwarded+denied != 256 {
		t.Fatalf("mask cells=%d want 256", forwarded+denied)
	}

	for z := 0; z < 16; z++ {
		for x := 0; x < 16; x++ {
			pos := mgl32.Vec3{float32(x), 0, float32(z)}
			legacyDenied := !ClaimActionPermitted(*claims[0], actor, ClaimActionRender, pos)
			got := mask[z]&(1<<uint(x)) != 0
			if got != legacyDenied {
				t.Fatalf("mismatch at %d,%d: mask=%v legacyDenied=%v", x, z, got, legacyDenied)
			}
		}
	}
	if denied == 0 || forwarded == 0 {
		t.Fatalf("expected mixed mask, forwarded=%d denied=%d", forwarded, denied)
	}
}

func TestFloorDenyMaskOwnerSeesNoDeny(t *testing.T) {
	claims := []*claim.PlayerClaim{{
		ID:        "home",
		OwnerXUID: "owner",
		Location: claim.Location{
			Dimension: "minecraft:overworld",
			Pos1:      claim.Vector2{X: 0, Z: 0},
			Pos2:      claim.Vector2{X: 15, Z: 15},
		},
	}}
	var mask [16]uint16
	forwarded, denied := floorDenyMask(claims, ClaimActor{XUID: "owner"}, 0, 0, &mask)
	if denied != 0 || forwarded != 256 || mask != [16]uint16{} {
		t.Fatalf("owner must see no deny: forwarded=%d denied=%d mask=%v", forwarded, denied, mask)
	}
}

func TestFloorDenyMaskFullChunkDenyFastPath(t *testing.T) {
	claims := []*claim.PlayerClaim{{
		ID:        "home",
		OwnerXUID: "owner",
		Location: claim.Location{
			Dimension: "minecraft:overworld",
			Pos1:      claim.Vector2{X: 0, Z: 0},
			Pos2:      claim.Vector2{X: 15, Z: 15},
		},
	}}
	var mask [16]uint16
	forwarded, denied := floorDenyMask(claims, ClaimActor{XUID: "stranger"}, 0, 0, &mask)
	if forwarded != 0 || denied != 256 {
		t.Fatalf("full-chunk stranger deny: forwarded=%d denied=%d", forwarded, denied)
	}
	for z := 0; z < 16; z++ {
		if mask[z] != 0xFFFF {
			t.Fatalf("mask[%d]=%#x want 0xffff", z, mask[z])
		}
	}
}

func TestDropHoldInteract(t *testing.T) {
	hold := &packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:  protocol.UseItemActionClickBlock,
			TriggerType: protocol.TriggerTypeSimulationTick,
		},
	}
	if !dropHoldInteract(hold) {
		t.Fatal("simulation-tick click-block must drop")
	}
	first := &packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:  protocol.UseItemActionClickBlock,
			TriggerType: protocol.TriggerTypePlayerInput,
		},
	}
	if dropHoldInteract(first) {
		t.Fatal("player-input click-block must pass")
	}
}
