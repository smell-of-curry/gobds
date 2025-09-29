package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// packetHandler ...
type packetHandler interface {
	Handle(s *Session, pk packet.Packet, ctx *Context) error
}
