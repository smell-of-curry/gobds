// Package entity provides entity-related functionality for the GoBDS proxy.
package entity

import "github.com/go-gl/mathgl/mgl32"

// Entity ...
type Entity struct {
	runtimeID       uint64
	actorType       string
	initialPosition mgl32.Vec3
}

// NewEntity ...
func NewEntity(runtimeID uint64, actorType string, initialPosition mgl32.Vec3) Entity {
	return Entity{
		runtimeID:       runtimeID,
		actorType:       actorType,
		initialPosition: initialPosition,
	}
}

// RuntimeID ...
func (e Entity) RuntimeID() uint64 {
	return e.runtimeID
}

// ActorType ...
func (e Entity) ActorType() string {
	return e.actorType
}

// InitialPosition ...
func (e Entity) InitialPosition() mgl32.Vec3 {
	return e.initialPosition
}
