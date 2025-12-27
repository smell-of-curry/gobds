package infra

import "github.com/smell-of-curry/gobds/gobds/entity"

var (
	// EntityFactory is the global entity factory instance.
	EntityFactory = entity.NewFactory()
)
