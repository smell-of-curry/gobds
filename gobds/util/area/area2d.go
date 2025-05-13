package area

// Area2D ...
type Area2D struct {
	MinX, MinZ int32
	MaxX, MaxZ int32
}

// NilArea2D ...
var NilArea2D Area2D

// NewArea2D ...
func NewArea2D(minX, minZ int32, maxX, maxZ int32) Area2D {
	return Area2D{
		MinX: minX,
		MinZ: minZ,
		MaxX: maxX,
		MaxZ: maxZ,
	}
}

// PositionInside ...
func (a Area2D) PositionInside(x, z int32) bool {
	return x > a.MinX && x < a.MaxX &&
		z > a.MinZ && z < a.MaxZ
}

// ArePositionsInside ...
func (a Area2D) ArePositionsInside(v []int32) bool {
	x := v[0] << 4
	z := v[1] << 4

	return x >= a.MinX-16 && x <= a.MaxX &&
		z >= a.MinZ-16 && z <= a.MaxZ
}
