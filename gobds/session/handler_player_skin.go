package session

import (
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/util/skinutil"
)

// PlayerSkinHandler ...
type PlayerSkinHandler struct{}

// Handle ...
func (h *PlayerSkinHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.PlayerSkin)

	if ctx.Val() == s.client &&
		!s.data.CanChangeSkin(s.skinConfig.SkinChangeCooldown) {
		ctx.Cancel()
		h.sendCurrentSkin(s)
		return nil
	}

	sk, err := skinutil.ProtocolToSkin(pkt.Skin)
	if err != nil {
		s.log.Error("error parsing skin", "error", err)
		ctx.Cancel()
		return nil
	}
	s.data.SetSkin(sk)

	xuid := s.IdentityData().XUID
	head := skinutil.ExtractHead(sk)
	if err = skinutil.SaveHeadPNG(xuid, head, s.skinConfig.HeadsDirectory); err != nil {
		s.log.Error("error saving head", "error", err, "xuid", xuid)
	}
	return nil
}

// sendCurrentSkin ...
func (h *PlayerSkinHandler) sendCurrentSkin(s *Session) {
	currentSkin := s.data.Skin()
	protocolSkin := protocol.Skin{
		SkinID:                    currentSkin.FullID,
		SkinResourcePatch:         currentSkin.ModelConfig.Encode(),
		SkinImageWidth:            uint32(currentSkin.Bounds().Dx()),
		SkinImageHeight:           uint32(currentSkin.Bounds().Dy()),
		SkinData:                  currentSkin.Pix,
		SkinGeometry:              currentSkin.Model,
		PersonaSkin:               currentSkin.Persona,
		PlayFabID:                 currentSkin.PlayFabID,
		CapeID:                    uuid.New().String(),
		FullID:                    currentSkin.FullID,
		CapeImageWidth:            uint32(currentSkin.Cape.Bounds().Dx()),
		CapeImageHeight:           uint32(currentSkin.Cape.Bounds().Dy()),
		CapeData:                  currentSkin.Cape.Pix,
		Trusted:                   true,
		OverrideAppearance:        true,
		GeometryDataEngineVersion: []byte(protocol.CurrentVersion),
	}
	for _, animation := range currentSkin.Animations {
		protocolAnim := protocol.SkinAnimation{
			ImageWidth:  uint32(animation.Bounds().Max.X),
			ImageHeight: uint32(animation.Bounds().Max.Y),
			ImageData:   animation.Pix,
			FrameCount:  float32(animation.FrameCount),
		}
		switch animation.Type() {
		case skin.AnimationHead:
			protocolAnim.AnimationType = protocol.SkinAnimationHead
		case skin.AnimationBody32x32:
			protocolAnim.AnimationType = protocol.SkinAnimationBody32x32
		case skin.AnimationBody128x128:
			protocolAnim.AnimationType = protocol.SkinAnimationBody128x128
		}
		protocolAnim.ExpressionType = uint32(animation.AnimationExpression)
		protocolSkin.Animations = append(protocolSkin.Animations, protocolAnim)
	}

	uid, _ := uuid.Parse(s.IdentityData().Identity)
	s.WriteToClient(&packet.PlayerSkin{
		UUID: uid,
		Skin: protocolSkin,
	})
}
