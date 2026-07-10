package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// ChangeDimensionHandler tracks server-authoritative dimension changes.
type ChangeDimensionHandler struct{}

// Handle ...
func (*ChangeDimensionHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() == s.server {
		s.Data().SetDimension(pk.(*packet.ChangeDimension).Dimension)
	}
	return nil
}
