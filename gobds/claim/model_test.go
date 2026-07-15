package claim

import (
	"encoding/json"
	"testing"
)

func TestPlayerClaimJSONMatchesBehaviorPackSchema(t *testing.T) {
	var cl PlayerClaim
	err := json.Unmarshal([]byte(`{
		"location":{"dimension":"minecraft:overworld","pos1":{"x":1,"z":2},"pos2":{"x":3,"z":4}},
		"features":[{"type":"blockIntractable","fromLocation":{"x":5,"y":6,"z":7}}]
	}`), &cl)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Location.Pos1 != (Vector2{X: 1, Z: 2}) || cl.Location.Pos2 != (Vector2{X: 3, Z: 4}) {
		t.Fatalf("unexpected claim bounds: %+v", cl.Location)
	}
	if len(cl.Features) != 1 || cl.Features[0].FromLocation == nil || cl.Features[0].ToLocation != nil {
		t.Fatalf("unexpected feature bounds: %+v", cl.Features)
	}
}
