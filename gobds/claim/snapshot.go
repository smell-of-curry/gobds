package claim

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// PolicyVersion identifies claim decision semantics shared with BEH.
	PolicyVersion = 1
	// SchemaVersion identifies claim payload shape shared with BEH.
	SchemaVersion = 1
	cellSize      = 16
	// ponytail: 262k-cell ceiling bounds hostile payload cost; add coarse cells if world claims exceed it.
	maxCellsPerClaim = 1 << 18
)

// Snapshot is immutable after construction and safe for concurrent lock-free reads.
type Snapshot struct {
	PolicyVersion int
	SchemaVersion int
	Generation    uint64
	GeneratedAt   time.Time
	FetchedAt     time.Time
	ClaimCount    int
	CellCount     int

	claims []PlayerClaim
	cells  map[string]map[cellKey][]*PlayerClaim
}

type cellKey struct {
	x int32
	z int32
}

// QueryStatus explains whether proxy policy can safely act on a query.
type QueryStatus uint8

const (
	// QueryReady means the snapshot can be used for policy decisions.
	QueryReady QueryStatus = iota
	// QueryMissing means no snapshot has been published yet.
	QueryMissing
	// QueryStale means the snapshot exceeded the configured max age.
	QueryStale
	// QueryUnsupported means policy/schema versions are not understood.
	QueryUnsupported
	// QueryUnknownDimension means the packet dimension could not be mapped.
	QueryUnknownDimension
	// QueryInvalid means the query inputs were malformed.
	QueryInvalid
	// QueryOverlap means multiple claims matched the same position.
	QueryOverlap
	// QueryRefreshFailed means the last refresh attempt failed.
	QueryRefreshFailed
)

// Candidates returns immutable claims from the fixed spatial cell containing x,z.
func (s *Snapshot) Candidates(dimension string, x, z float32) []*PlayerClaim {
	if s == nil {
		return nil
	}
	return s.cells[dimension][cellFor(x, z)]
}

// Supported reports whether this process understands snapshot policy and schema.
func (s *Snapshot) Supported() bool {
	return s != nil && s.PolicyVersion == PolicyVersion && s.SchemaVersion == SchemaVersion
}

// Age returns elapsed time since claims were last fetched or revalidated.
func (s *Snapshot) Age(now time.Time) time.Duration {
	if s == nil || s.FetchedAt.IsZero() {
		return time.Duration(math.MaxInt64)
	}
	return max(now.Sub(s.FetchedAt), 0)
}

// BuildSnapshot fully validates, clones, and indexes claims before publication.
func BuildSnapshot(claims map[string]PlayerClaim, generation uint64, now time.Time) (*Snapshot, error) {
	snapshot := &Snapshot{
		PolicyVersion: PolicyVersion,
		SchemaVersion: SchemaVersion,
		Generation:    generation,
		GeneratedAt:   now,
		FetchedAt:     now,
		ClaimCount:    len(claims),
		claims:        make([]PlayerClaim, 0, len(claims)),
		cells:         make(map[string]map[cellKey][]*PlayerClaim),
	}
	ids := make(map[string]struct{}, len(claims))
	for key, source := range claims {
		cl, err := cloneAndValidateClaim(key, source)
		if err != nil {
			return nil, err
		}
		if _, exists := ids[cl.ID]; exists {
			return nil, fmt.Errorf("duplicate claim id %q", cl.ID)
		}
		ids[cl.ID] = struct{}{}
		snapshot.claims = append(snapshot.claims, cl)
	}
	for i := range snapshot.claims {
		cl := &snapshot.claims[i]
		dimensionCells := snapshot.cells[cl.Location.Dimension]
		if dimensionCells == nil {
			dimensionCells = make(map[cellKey][]*PlayerClaim)
			snapshot.cells[cl.Location.Dimension] = dimensionCells
		}
		minCell, ok := checkedCellFor(
			min(cl.Location.Pos1.X, cl.Location.Pos2.X),
			min(cl.Location.Pos1.Z, cl.Location.Pos2.Z),
		)
		if !ok {
			return nil, fmt.Errorf("claim %q bounds exceed spatial index", cl.ID)
		}
		maxCell, ok := checkedCellFor(
			max(cl.Location.Pos1.X, cl.Location.Pos2.X),
			max(cl.Location.Pos1.Z, cl.Location.Pos2.Z),
		)
		if !ok {
			return nil, fmt.Errorf("claim %q bounds exceed spatial index", cl.ID)
		}
		width := int64(maxCell.x) - int64(minCell.x) + 1
		depth := int64(maxCell.z) - int64(minCell.z) + 1
		if width > maxCellsPerClaim || depth > maxCellsPerClaim ||
			width*depth > maxCellsPerClaim {
			return nil, fmt.Errorf("claim %q spans too many spatial cells", cl.ID)
		}
		for x := int64(minCell.x); x <= int64(maxCell.x); x++ {
			for z := int64(minCell.z); z <= int64(maxCell.z); z++ {
				key := cellKey{x: int32(x), z: int32(z)}
				if len(dimensionCells[key]) == 0 {
					snapshot.CellCount++
				}
				dimensionCells[key] = append(dimensionCells[key], cl)
			}
		}
	}
	return snapshot, nil
}

func cloneAndValidateClaim(key string, source PlayerClaim) (PlayerClaim, error) {
	cl := source
	if cl.ID == "" {
		cl.ID = key
	}
	if cl.ID == "" || cl.OwnerXUID == "" {
		return PlayerClaim{}, fmt.Errorf("claim %q missing id or owner", key)
	}
	dimension, ok := CanonicalDimension(cl.Location.Dimension)
	if !ok {
		return PlayerClaim{}, fmt.Errorf("claim %q has invalid dimension %q", cl.ID, cl.Location.Dimension)
	}
	cl.Location.Dimension = dimension
	if !finite2(cl.Location.Pos1) || !finite2(cl.Location.Pos2) {
		return PlayerClaim{}, fmt.Errorf("claim %q has non-finite bounds", cl.ID)
	}
	cl.TrustedXUIDS = append([]string(nil), source.TrustedXUIDS...)
	cl.Features = make([]Feature, len(source.Features))
	for i, feature := range source.Features {
		if err := validateFeature(cl.ID, feature); err != nil {
			return PlayerClaim{}, err
		}
		cl.Features[i] = cloneFeature(feature)
	}
	return cl, nil
}

func validateFeature(claimID string, feature Feature) error {
	switch feature.Type {
	case FeatureTypeMineable, FeatureTypeBlockPlaceable, FeatureTypeBlockInteractable,
		FeatureTypeEntityInteractable, FeatureTypeEntityHurt, FeatureTypeDropItems,
		FeatureTypePickupItems:
	default:
		return fmt.Errorf("claim %q has unsupported feature %q", claimID, feature.Type)
	}
	if feature.FromLocation != nil && !finite3(*feature.FromLocation) {
		return fmt.Errorf("claim %q has non-finite feature start", claimID)
	}
	if feature.ToLocation != nil && !finite3(*feature.ToLocation) {
		return fmt.Errorf("claim %q has non-finite feature end", claimID)
	}
	return nil
}

func cloneFeature(source Feature) Feature {
	feature := source
	if source.FromLocation != nil {
		from := *source.FromLocation
		feature.FromLocation = &from
	}
	if source.ToLocation != nil {
		to := *source.ToLocation
		feature.ToLocation = &to
	}
	feature.BlockTypeIDs = append([]string(nil), source.BlockTypeIDs...)
	feature.EntityTypeIDs = append([]string(nil), source.EntityTypeIDs...)
	feature.ItemTypeIDs = append([]string(nil), source.ItemTypeIDs...)
	return feature
}

// CanonicalDimension normalizes vanilla aliases and validates namespaced IDs.
func CanonicalDimension(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "overworld":
		value = "minecraft:overworld"
	case "nether":
		value = "minecraft:nether"
	case "end", "the_end":
		value = "minecraft:end"
	}
	parts := strings.Split(value, ":")
	return value, len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func cellFor(x, z float32) cellKey {
	return cellKey{
		x: int32(math.Floor(float64(x) / cellSize)),
		z: int32(math.Floor(float64(z) / cellSize)),
	}
}

func checkedCellFor(x, z float32) (cellKey, bool) {
	cellX := math.Floor(float64(x) / cellSize)
	cellZ := math.Floor(float64(z) / cellSize)
	if cellX < math.MinInt32 || cellX > math.MaxInt32 ||
		cellZ < math.MinInt32 || cellZ > math.MaxInt32 {
		return cellKey{}, false
	}
	return cellKey{x: int32(cellX), z: int32(cellZ)}, true
}

func finite2(v Vector2) bool {
	return !float32Invalid(v.X) && !float32Invalid(v.Z)
}

func finite3(v Vector3) bool {
	return !float32Invalid(v.X) && !float32Invalid(v.Y) && !float32Invalid(v.Z)
}

func float32Invalid(v float32) bool {
	return math.IsNaN(float64(v)) || math.IsInf(float64(v), 0)
}
