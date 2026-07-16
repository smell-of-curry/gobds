package block

import (
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

// Deny ...
type Deny struct{}

// EncodeItem ...
func (Deny) EncodeItem() (name string, meta int16) {
	return "minecraft:deny", 0
}

// EncodeBlock ...
func (Deny) EncodeBlock() (string, map[string]any) {
	return "minecraft:deny", nil
}

// Model ...
func (Deny) Model() world.BlockModel {
	return model.Solid{}
}

// Hash ...
func (Deny) Hash() (uint64, uint64) {
	return hashDeny, 0
}
