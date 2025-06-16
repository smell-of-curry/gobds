package block

import "github.com/df-mc/dragonfly/server/world"

func init() {
	for _, b := range Buttons() {
		world.RegisterBlock(b)
	}
}
