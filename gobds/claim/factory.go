package claim

import (
	"log/slog"
	"sync"

	"github.com/smell-of-curry/gobds/gobds/service"
)

// Factory ...
type Factory struct {
	service *Service
	data    map[string]PlayerClaim

	mu  sync.RWMutex
	log *slog.Logger
}

// NewFactory ...
func NewFactory(c service.Config, log *slog.Logger) *Factory {
	return &Factory{
		service: NewService(c, log),
		data:    make(map[string]PlayerClaim),
		log:     log,
	}
}

// All ...
func (f *Factory) All() map[string]PlayerClaim {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.data
}

// Set ...
func (f *Factory) Set(claims map[string]PlayerClaim) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = claims
}

// Fetch ...
func (f *Factory) Fetch() error {
	if f.service == nil || !f.service.Enabled {
		return nil
	}

	claims, err := f.service.FetchClaims()
	if err != nil {
		return err
	}

	f.Set(claims)
	return nil
}
