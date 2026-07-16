package session

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

type policyGolden struct {
	SchemaVersion int                `json:"schemaVersion"`
	PolicyVersion int                `json:"policyVersion"`
	Cases         []policyGoldenCase `json:"cases"`
}

type policyGoldenCase struct {
	Name  string            `json:"name"`
	Claim claim.PlayerClaim `json:"claim"`
	Actor struct {
		XUID     string `json:"xuid"`
		Operator bool   `json:"operator"`
	} `json:"actor"`
	Action    string        `json:"action"`
	Position  claim.Vector3 `json:"position"`
	TypeID    string        `json:"typeId"`
	Permitted bool          `json:"permitted"`
}

func TestClaimPolicyGoldenFixture(t *testing.T) {
	raw, err := os.ReadFile("../../policy/claim_policy.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden policyGolden
	if err = json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if golden.SchemaVersion != claim.SchemaVersion || golden.PolicyVersion != claim.PolicyVersion {
		t.Fatalf("unsupported golden versions: schema=%d policy=%d", golden.SchemaVersion, golden.PolicyVersion)
	}
	for _, test := range golden.Cases {
		t.Run(test.Name, func(t *testing.T) {
			action := claimActionFromGolden(t, test.Action)
			got := ClaimActionPermitted(
				test.Claim,
				ClaimActor{XUID: test.Actor.XUID, Operator: test.Actor.Operator},
				action,
				claimActionData{
					position: mgl32.Vec3{test.Position.X, test.Position.Y, test.Position.Z},
					typeID:   test.TypeID,
				},
			)
			if got != test.Permitted {
				t.Fatalf("permitted = %v, want %v", got, test.Permitted)
			}
		})
	}
}

func claimActionFromGolden(t *testing.T, value string) ClaimAction {
	t.Helper()
	switch value {
	case "render":
		return ClaimActionRender
	case "blockBreak":
		return ClaimActionBlockBreak
	case "blockPlace":
		return ClaimActionBlockPlace
	case "blockInteract":
		return ClaimActionBlockInteract
	case "entityInteract":
		return ClaimActionEntityInteract
	case "entityHurt":
		return ClaimActionEntityHurt
	case "itemRelease":
		return ClaimActionItemRelease
	case "itemThrow":
		return ClaimActionItemThrow
	case "itemDrop":
		return ClaimActionItemDrop
	default:
		t.Fatalf("unknown golden action: %q", value)
		return 0
	}
}
