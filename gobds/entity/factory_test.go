package entity

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestFactoryRemovesEntityByUniqueID(t *testing.T) {
	factory := NewFactory()
	factory.Add(NewEntity(7, 42, "minecraft:armor_stand", mgl32.Vec3{}))
	factory.RemoveFromUniqueID(7)
	if _, ok := factory.ByRuntimeID(42); ok {
		t.Fatal("entity remained after removal by unique ID")
	}
}

func TestFactoryUpdatesPartialPosition(t *testing.T) {
	factory := NewFactory()
	factory.Add(NewEntity(7, 42, "minecraft:armor_stand", mgl32.Vec3{1, 2, 3}))
	factory.UpdatePosition(42, mgl32.Vec3{4, 0, 6}, true, false, true)
	entity, ok := factory.ByRuntimeID(42)
	if !ok {
		t.Fatal("entity missing after position update")
	}
	if entity.Position() != (mgl32.Vec3{4, 2, 6}) {
		t.Fatalf("unexpected entity position: %v", entity.Position())
	}
}
