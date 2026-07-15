package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ItemStackRequestHandler ...
type ItemStackRequestHandler struct{}

// Handle ...
func (*ItemStackRequestHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.ItemStackRequest)
	if ctx.Val() == s.client {
		drop, err := validateItemStackRequest(pkt, s.traffic.config)
		if drop {
			s.traffic.malformed(trafficStack)
			ctx.Cancel()
			return nil
		}
		if err != nil {
			s.traffic.malformed(trafficStack)
			return err
		}
		if !s.traffic.allow(trafficStack) {
			ctx.Cancel()
			return nil
		}
	}
	for _, request := range pkt.Requests {
		for _, requestAction := range request.Actions {
			switch action := requestAction.(type) {
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

func validateItemStackRequest(pkt *packet.ItemStackRequest, config TrafficConfig) (drop bool, err error) {
	if len(pkt.Requests) == 0 {
		return true, nil
	}
	if len(pkt.Requests) > config.MaxStackRequests {
		return false, malformedPacketError{reason: "item stack packet has too many requests"}
	}
	totalActions := 0
	for _, request := range pkt.Requests {
		if len(request.Actions) == 0 {
			return true, nil
		}
		if len(request.Actions) > config.MaxStackActions {
			return false, malformedPacketError{reason: "item stack request has too many actions"}
		}
		totalActions += len(request.Actions)
		if totalActions > config.MaxTotalStackActions {
			return false, malformedPacketError{reason: "item stack packet has too many total actions"}
		}
	}
	return false, nil
}
