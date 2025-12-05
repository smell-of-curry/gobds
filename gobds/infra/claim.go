package infra

import (
	"maps"
	"sync"

	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

// MappedClaims ...
type MappedClaims map[string]claim.PlayerClaim

var (
	claims   = make(MappedClaims)
	claimsMu sync.RWMutex
)

// Claims ...
func Claims() MappedClaims {
	claimsMu.RLock()
	defer claimsMu.RUnlock()
	return maps.Clone(claims)
}

// SetClaims ...
func SetClaims(c MappedClaims) {
	claimsMu.Lock()
	defer claimsMu.Unlock()
	claims = maps.Clone(c)
}
