package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

// Button ...
type Button struct {
	Type    ButtonType
	Facing  cube.Face
	Pressed bool
}

// EncodeBlock ...
func (b Button) EncodeBlock() (string, map[string]any) {
	return "minecraft:" + b.Type.String() + "_button", map[string]any{"facing_direction": int32(b.Facing), "button_pressed_bit": b.Pressed}
}

// Model ...
func (b Button) Model() world.BlockModel {
	return model.Solid{}
}

// Hash ...
func (b Button) Hash() (uint64, uint64) {
	return hashButton, uint64(b.Type.Uint8())<<8 | uint64(b.Facing)<<14 | uint64(boolByte(b.Pressed))<<17
}

// Buttons ...
func Buttons() (buttons []world.Block) {
	for _, w := range ButtonTypes() {
		for _, f := range cube.Faces() {
			buttons = append(buttons, Button{Type: w, Facing: f})
			buttons = append(buttons, Button{Type: w, Facing: f, Pressed: true})
		}
	}
	return
}
