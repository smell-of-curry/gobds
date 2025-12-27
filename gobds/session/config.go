package session

import (
	"encoding/base64"
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/util/skinutil"
)

// Config ...
type Config struct {
	Client        Conn
	Server        Conn
	SkinConfig    *infra.SkinConfig
	PingIndicator *infra.PingIndicator
	AfkTimer      *infra.AFKTimer
	Border        *area.Area2D
	Log           *slog.Logger
}

// New ...
func (c Config) New() *Session {
	s := &Session{
		client: c.Client,
		server: c.Server,

		skinConfig:    c.SkinConfig,
		pingIndicator: c.PingIndicator,
		afkTimer:      c.AfkTimer,
		border:        c.Border,

		close: make(chan struct{}),

		data: NewData(c.Client),
		log:  c.Log,
	}
	s.registerHandlers()
	go s.parseLoginSkin()
	return s
}

// parseLoginSkin ...
func (s *Session) parseLoginSkin() {
	clientData := s.client.ClientData()
	xuid := s.IdentityData().XUID

	skinData, _ := base64.StdEncoding.DecodeString(clientData.SkinData)
	skinResourcePatch, _ := base64.StdEncoding.DecodeString(clientData.SkinResourcePatch)
	skinGeometry, _ := base64.StdEncoding.DecodeString(clientData.SkinGeometry)
	capeData, _ := base64.StdEncoding.DecodeString(clientData.CapeData)
	protocolSkin := protocol.Skin{
		SkinID:            clientData.SkinID,
		PlayFabID:         clientData.PlayFabID,
		SkinResourcePatch: skinResourcePatch,
		SkinImageWidth:    uint32(clientData.SkinImageWidth),
		SkinImageHeight:   uint32(clientData.SkinImageHeight),
		SkinData:          skinData,
		CapeImageWidth:    uint32(clientData.CapeImageWidth),
		CapeImageHeight:   uint32(clientData.CapeImageHeight),
		CapeData:          capeData,
		SkinGeometry:      skinGeometry,
		PersonaSkin:       clientData.PersonaSkin,
	}

	playerSkin, err := skinutil.ProtocolToSkin(protocolSkin)
	if err != nil {
		s.log.Error("error parsing skin at login", "error", err, "xuid", xuid)
		return
	}

	s.data.skin.Store(playerSkin)

	head := skinutil.ExtractHead(playerSkin)
	if err = skinutil.SaveHeadPNG(xuid, head, s.skinConfig.HeadsDirectory); err != nil {
		s.log.Error("error saving head at login", "error", err, "xuid", xuid)
	}
}
