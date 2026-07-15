package entity

import (
	"maps"
	"slices"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

// Factory ...
type Factory struct {
	entities            map[uint64]Entity
	runtimeIDByUniqueID map[int64]uint64
	mu                  sync.RWMutex
}

// NewFactory ...
func NewFactory() *Factory {
	return &Factory{
		entities:            make(map[uint64]Entity),
		runtimeIDByUniqueID: make(map[int64]uint64),
	}
}

// Add ...
func (f *Factory) Add(e Entity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if previous, ok := f.entities[e.RuntimeID()]; ok {
		delete(f.runtimeIDByUniqueID, previous.UniqueID())
	}
	if previousRuntimeID, ok := f.runtimeIDByUniqueID[e.UniqueID()]; ok {
		delete(f.entities, previousRuntimeID)
	}
	f.entities[e.RuntimeID()] = e
	f.runtimeIDByUniqueID[e.UniqueID()] = e.RuntimeID()
}

// RemoveFromRuntimeID ...
func (f *Factory) RemoveFromRuntimeID(runtimeID uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.entities[runtimeID]; ok {
		delete(f.runtimeIDByUniqueID, e.UniqueID())
	}
	delete(f.entities, runtimeID)
}

// RemoveFromUniqueID ...
func (f *Factory) RemoveFromUniqueID(uniqueID int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	runtimeID, ok := f.runtimeIDByUniqueID[uniqueID]
	if !ok {
		return
	}
	delete(f.runtimeIDByUniqueID, uniqueID)
	delete(f.entities, runtimeID)
}

// UpdatePosition updates selected absolute position components.
func (f *Factory) UpdatePosition(runtimeID uint64, position mgl32.Vec3, updateX, updateY, updateZ bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.entities[runtimeID]
	if !ok {
		return
	}
	if updateX {
		e.position[0] = position[0]
	}
	if updateY {
		e.position[1] = position[1]
	}
	if updateZ {
		e.position[2] = position[2]
	}
	f.entities[runtimeID] = e
}

// ByRuntimeID ...
func (f *Factory) ByRuntimeID(runtimeID uint64) (Entity, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	e, ok := f.entities[runtimeID]
	return e, ok
}

// All ...
func (f *Factory) All() []Entity {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return slices.Collect(maps.Values(f.entities))
}
