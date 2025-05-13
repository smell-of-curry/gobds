package infra

import (
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

var (
	WorldBorder = area.NilArea2D
)

// WorldBorderEnabled ...
func WorldBorderEnabled() bool {
	return WorldBorder != area.NilArea2D
}
