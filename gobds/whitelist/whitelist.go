package whitelist

import (
	"strings"
	"sync"
)

// Whitelist ...
type Whitelist struct {
	entries []string
	mu      sync.RWMutex
}

// NewWhitelist ...
func NewWhitelist(entries []string) *Whitelist {
	return &Whitelist{entries: entries}
}

// Has ...
func (w *Whitelist) Has(name string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, entry := range w.entries {
		if strings.ToLower(entry) == strings.ToLower(name) {
			return true
		}
	}
	return false
}
