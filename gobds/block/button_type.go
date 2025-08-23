package block

import "github.com/df-mc/dragonfly/server/block"

// ButtonType ...
type ButtonType struct {
	button
	wood block.WoodType
}

type button uint8

// WoodButton ...
func WoodButton(w block.WoodType) ButtonType {
	return ButtonType{0, w}
}

// StoneButton ...
func StoneButton() ButtonType {
	return ButtonType{button: 1}
}

// PolishedBlackstoneButton ...
func PolishedBlackstoneButton() ButtonType {
	return ButtonType{button: 2}
}

// Uint8 ...
func (b ButtonType) Uint8() uint8 {
	return b.wood.Uint8() | uint8(b.button)<<4
}

// Name ...
func (b ButtonType) Name() string {
	switch b.button {
	case 0:
		return b.wood.Name() + " Button"
	case 1:
		return "Stone Button"
	case 2:
		return "Polished Blackstone Button"
	}
	panic("unknown button type")
}

// String ...
func (b ButtonType) String() string {
	switch b.button {
	case 0:
		if b.wood == block.OakWood() {
			return "wooden"
		}
		return b.wood.String()
	case 1:
		return "stone"
	case 2:
		return "polished_blackstone"
	}
	panic("unknown button type")
}

// ButtonTypes ...
func ButtonTypes() []ButtonType {
	types := []ButtonType{StoneButton(), PolishedBlackstoneButton()}
	for _, w := range block.WoodTypes() {
		types = append(types, WoodButton(w))
	}
	return types
}
