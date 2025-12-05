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
	Pos1      Vector2 `json:"pos2"`
	Pos2      Vector2 `json:"pos1"`
}

const (
	FeatureTypeMineable          = "mineable"
	FeatureTypeBlockPlaceable    = "blockPlaceable"
	FeatureTypeBlockIntractable  = "blockIntractable"
	FeatureTypeEntityIntractable = "entityIntractable"
	FeatureTypeEntityHurt        = "entityHurt"
)

// Feature ...
type Feature struct {
	Type         string  `json:"type"`
	FromLocation Vector2 `json:"fromLocation"`
	ToLocation   Vector2 `json:"toLocation"`
}

// Vector2 ...
type Vector2 struct {
	X float32 `json:"x"`
	Z float32 `json:"z"`
}
