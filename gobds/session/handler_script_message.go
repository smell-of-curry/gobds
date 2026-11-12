package session

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/identity"
)

// ScriptMessageHandler keeps broker challenges off the public client connection.
// At most one request is in flight per session; retries are bounded by the addon.
type ScriptMessageHandler struct{ publishing atomic.Bool }

func (h *ScriptMessageHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	p := pk.(*packet.ScriptMessage)
	if p.Identifier != identity.MessageID {
		return nil
	}
	ctx.Cancel()
	if ctx.Val() != s.server || s.identityPublisher == nil {
		return nil
	}
	// Never use the upstream connection's copied IdentityData as the issuer.
	authenticated := s.client.IdentityData()
	assertion, ok := s.identityPublisher.Decode(p.Data,
		strconv.FormatInt(s.server.GameData().EntityUniqueID, 10), authenticated.DisplayName, authenticated.XUID, time.Now())
	if !ok || !h.publishing.CompareAndSwap(false, true) {
		return nil
	}
	go func() {
		defer h.publishing.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-s.close:
				cancel()
			case <-done:
			}
		}()
		if err := s.identityPublisher.Publish(ctx, assertion); err != nil && s.log != nil {
			// Deliberately omit the payload, credentials, URL and account identifiers.
			s.log.Warn("session identity delivery failed")
		}
	}()
	return nil
}
