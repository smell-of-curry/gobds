package whitelist

import (
	"strings"
)

// Whitelist ...
type Whitelist struct {
	entries []string
}

// NewWhitelist ...
func NewWhitelist(entries []string) *Whitelist {
	return &Whitelist{entries: entries}
}

// Has ...
func (w *Whitelist) Has(name string) bool {
	for _, entry := range w.entries {
		if strings.EqualFold(entry, name) {
			return true
		}
	}
	return false
}
