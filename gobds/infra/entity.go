package infra

import "github.com/smell-of-curry/gobds/gobds/entity"

var (
	EntityFactory *entity.Factory
)

func init() {
	EntityFactory = entity.NewFactory()
}
