package gobds

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/df-mc/go-nethernet"
)

// loadOrCreateServerKey returns the persistent P-384 key used to sign the
// NetherNet server identity. go-nethernet generates a fresh key per process by
// default, which surfaces a TOFU ("trust this server?") prompt to every player
// after each restart; persisting the key keeps the fingerprint stable.
func loadOrCreateServerKey(path string) (*ecdsa.PrivateKey, error) {
	if b, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(b)
		if block == nil {
			return nil, fmt.Errorf("nethernet key %s: not PEM", path)
		}
		return x509.ParseECPrivateKey(block.Bytes)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	b := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// serverIdentityIssuer adapts a persistent key to nethernet.ListenConfig.IssueServerIdentity.
// "self" is the domain BDS itself advertises for self-signed server identities.
func serverIdentityIssuer(key *ecdsa.PrivateKey) func(context.Context) (*nethernet.Identity, error) {
	return func(context.Context) (*nethernet.Identity, error) {
		return nethernet.GenerateServerIdentity(key, "self")
	}
}
