package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

type packetHandler interface {
	Handle(s *Session, packet packet.Packet, ctx *Context) error
}
