package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/util/formutil"
)

// ModalFormRequestHandler ...
type ModalFormRequestHandler struct{}

// Handle ...
func (*ModalFormRequestHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}

	pkt := pk.(*packet.ModalFormRequest)
	form, err := formutil.ParseMenuForm(pkt.FormData)
	if err != nil || form == nil {
		return nil
	}

	modified := false
	for i := range form.Elements {
		xuid, exists := form.Elements[i].KeyFaceXUID()
		if !exists {
			continue
		}
		form.Elements[i].SetImageURL(s.skinConfig.HeadURL(xuid))
		modified = true
	}
	if !modified {
		return nil
	}

	formData, err := form.Marshal()
	if err != nil {
		s.log.Error("error marshaling form", "error", err)
		return nil
	}

	ctx.Cancel()
	s.WriteToClient(&packet.ModalFormRequest{
		FormID:   pkt.FormID,
		FormData: formData,
	})
	return nil
}
