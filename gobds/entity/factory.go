package entity

import (
	"maps"
	"slices"
	"sync"
)

// Factory ...
type Factory struct {
	entities map[uint64]Entity
	mu       sync.RWMutex
}

// NewFactory ...
func NewFactory() *Factory {
	return &Factory{
		entities: make(map[uint64]Entity),
	}
}

// Add ...
func (f *Factory) Add(e Entity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entities[e.RuntimeID()] = e
}

// RemoveFromRuntimeID ...
func (f *Factory) RemoveFromRuntimeID(runtimeID uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.entities, runtimeID)
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
