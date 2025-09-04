package infra

import (
	"sync"

	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

// MappedClaims ...
type MappedClaims map[string]claim.PlayerClaim

var (
	claims   MappedClaims = make(MappedClaims)
	claimsMu sync.RWMutex
)

// Claims ...
func Claims() MappedClaims {
	claimsMu.RLock()
	defer claimsMu.RUnlock()
	return claims
}

// SetClaims ...
func SetClaims(c MappedClaims) {
	claimsMu.Lock()
	defer claimsMu.Unlock()
	claims = c
}
