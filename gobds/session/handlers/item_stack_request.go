package handlers

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// ItemStackRequest ...
type ItemStackRequest struct{}

// Handle ...
func (ItemStackRequest) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.ItemStackRequest)
	for _, request := range pkt.Requests {
		for _, requestAction := range request.Actions {
			switch action := requestAction.(type) {
			case *protocol.PlaceStackRequestAction:
				if action.Source.Container.ContainerID == protocol.ContainerDynamic ||
					action.Destination.Container.ContainerID == protocol.ContainerDynamic {
					c.WriteToClient(&packet.ItemStackResponse{Responses: []protocol.ItemStackResponse{
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
					c.WriteToClient(&packet.ItemStackResponse{Responses: []protocol.ItemStackResponse{
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
}
