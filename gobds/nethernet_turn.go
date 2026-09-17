package gobds

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/df-mc/go-nethernet"
)

// turnCredentialsFunc returns an endpoint.HandlerConfig.Credentials function that
// mints short-lived coturn REST credentials (use-auth-secret / static-auth-secret).
// The credentials make the NetherNet listener gather a public TURN relay
// candidate, which is what lets players behind symmetric NAT / CGNAT connect
// (their private host candidate is unreachable otherwise).
//
// address is host:port of the TURN server (e.g. "15.204.44.68:3478"); secret is
// the coturn static-auth-secret. When either is empty the returned function is
// nil, so the caller falls back to go-nethernet's default (no ICE servers).
func turnCredentialsFunc(address, secret string) func(context.Context) (*nethernet.Credentials, error) {
	if address == "" || secret == "" {
		return nil
	}
	const ttl = 5 * time.Minute
	return func(context.Context) (*nethernet.Credentials, error) {
		username := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
		mac := hmac.New(sha1.New, []byte(secret))
		_, _ = mac.Write([]byte(username))
		password := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		return &nethernet.Credentials{
			ExpirationInSeconds: int(ttl.Seconds()),
			ICEServers: []nethernet.ICEServer{{
				Username: username,
				Password: password,
				URLs: []string{
					fmt.Sprintf("stun:%s", address),
					fmt.Sprintf("turn:%s?transport=udp", address),
				},
			}},
		}, nil
	}
}
