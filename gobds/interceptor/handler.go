package interceptor

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// Handler ...
type Handler interface {
	Handle(Client, packet.Packet, *session.Context)
}
