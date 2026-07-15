package session

import (
	"bytes"
	"encoding/json"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ModalFormResponseHandler bounds and rate-checks client form responses.
type ModalFormResponseHandler struct{}

// Handle validates a response before forwarding it to BDS.
func (*ModalFormResponseHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.client {
		return nil
	}
	pkt := pk.(*packet.ModalFormResponse)
	response, present := pkt.ResponseData.Value()
	if present {
		if err := validateFormResponse(response, s.traffic.config); err != nil {
			s.traffic.malformed(trafficForm)
			return err
		}
	}
	if _, cancelled := pkt.CancelReason.Value(); !present && !cancelled {
		s.traffic.malformed(trafficForm)
		return malformedPacketError{reason: "form response has no response or cancellation"}
	}
	if !s.traffic.allow(trafficForm) {
		ctx.Cancel()
	}
	return nil
}

func validateFormResponse(response []byte, config TrafficConfig) error {
	if len(response) > config.MaxFormResponseBytes {
		return malformedPacketError{reason: "form response exceeds maximum length"}
	}
	response = bytes.TrimSpace(response)
	if len(response) == 0 || !json.Valid(response) {
		return malformedPacketError{reason: "form response is not valid JSON"}
	}
	if response[0] != '[' {
		return nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(response, &values); err != nil {
		return malformedPacketError{reason: "form response array is invalid"}
	}
	if len(values) > config.MaxFormResponseValues {
		return malformedPacketError{reason: "form response has too many values"}
	}
	return nil
}
