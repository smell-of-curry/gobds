// Package entity provides entity-related functionality for the GoBDS proxy.
package entity

import "github.com/go-gl/mathgl/mgl32"

// Entity ...
type Entity struct {
	uniqueID  int64
	runtimeID uint64
	actorType string
	position  mgl32.Vec3
}

// NewEntity ...
func NewEntity(uniqueID int64, runtimeID uint64, actorType string, position mgl32.Vec3) Entity {
	return Entity{
		uniqueID:  uniqueID,
		runtimeID: runtimeID,
		actorType: actorType,
		position:  position,
	}
}

// UniqueID ...
func (e Entity) UniqueID() int64 {
	return e.uniqueID
}

// RuntimeID ...
func (e Entity) RuntimeID() uint64 {
	return e.runtimeID
}

// ActorType ...
func (e Entity) ActorType() string {
	return e.actorType
}

// Position ...
func (e Entity) Position() mgl32.Vec3 {
	return e.position
}
