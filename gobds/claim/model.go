package claim

// ResponseModel ...
type ResponseModel struct {
	Data      PlayerClaim `json:"data"`
	Key       string      `json:"_key"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

// PlayerClaim ...
type PlayerClaim struct {
	ID           string    `json:"claimId"`
	OwnerXUID    string    `json:"playerXUID"`
	Location     Location  `json:"location"`
	Features     []Feature `json:"features"`
	TrustedXUIDS []string  `json:"trusts"`
}

// Location ...
type Location struct {
	Dimension string  `json:"dimension"`
	Pos1      Vector2 `json:"pos1"`
	Pos2      Vector2 `json:"pos2"`
}

const (
	// FeatureTypeMineable ...
	FeatureTypeMineable = "mineable"
	// FeatureTypeBlockPlaceable ...
	FeatureTypeBlockPlaceable = "blockPlaceable"
	// FeatureTypeBlockInteractable preserves the API's historical "intractable" value.
	FeatureTypeBlockInteractable = "blockIntractable"
	// FeatureTypeEntityInteractable preserves the API's historical "intractable" value.
	FeatureTypeEntityInteractable = "entityIntractable"
	// FeatureTypeEntityHurt ...
	FeatureTypeEntityHurt = "entityHurt"
	// FeatureTypeDropItems ...
	FeatureTypeDropItems = "dropItems"
	// FeatureTypePickupItems is stored/accepted for BEH parity; proxy does not enforce pickup yet.
	FeatureTypePickupItems = "pickupItems"
)

// Feature ...
type Feature struct {
	Type          string   `json:"type"`
	FromLocation  *Vector3 `json:"fromLocation,omitempty"`
	ToLocation    *Vector3 `json:"toLocation,omitempty"`
	BlockTypeIDs  []string `json:"blockTypeIds,omitempty"`
	EntityTypeIDs []string `json:"entityTypeIds,omitempty"`
	ItemTypeIDs   []string `json:"itemTypeIds,omitempty"`
}

// Vector2 ...
type Vector2 struct {
	X float32 `json:"x"`
	Z float32 `json:"z"`
}

// Vector3 ...
type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}
