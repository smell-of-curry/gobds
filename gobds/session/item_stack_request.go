package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ItemStackRequest ...
type ItemStackRequest struct{}

// Handle ...
func (*ItemStackRequest) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.ItemStackRequest)
	for _, request := range pkt.Requests {
		for _, action := range request.Actions {
			switch action := action.(type) {
			case *protocol.PlaceStackRequestAction:
				if action.Source.Container.ContainerID == protocol.ContainerDynamic ||
					action.Destination.Container.ContainerID == protocol.ContainerDynamic {
					s.WriteToClient(&packet.ItemStackResponse{Responses: []protocol.ItemStackResponse{
						{
							Status:    protocol.ItemStackResponseStatusError,
							RequestID: request.RequestID,
						},
					}})
					ctx.Cancel()
				}
			case *protocol.TakeStackRequestAction:
				if action.Source.Container.ContainerID == protocol.ContainerDynamic ||
					action.Destination.Container.ContainerID == protocol.ContainerDynamic {
					s.WriteToClient(&packet.ItemStackResponse{Responses: []protocol.ItemStackResponse{
						{
							Status:    protocol.ItemStackResponseStatusError,
							RequestID: request.RequestID,
						},
					}})
					ctx.Cancel()
				}
			}
		}
	}
	return nil
}
