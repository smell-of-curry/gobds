// Package gobds implements a bedrock proxy server for Minecraft with authentication and claim support.
package gobds

import (
	"fmt"
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/whitelist"
)

// Config ...
type Config struct {
	Listeners             []func(conf Config) (Listener, error)
	StatusProvider        minecraft.ServerStatusProvider
	SecuredSlots          int
	EncryptionKey         string
	AuthenticationService *authentication.Service
	ClaimService          *claim.Service
	VPNService            *vpn.Service
	PingIndicator         *infra.PingIndicator
	AFKTimer              *infra.AFKTimer
	Whitelist             *whitelist.Whitelist
	Border                *area.Area2D
	PlayerManager         *PlayerManager
	DialerFunction        DialerFunc
	Log                   *slog.Logger
}

// Config converts the user configuration into a runtime configuration.
func (c UserConfig) Config(log *slog.Logger) (Config, error) {
	conf := Config{
		SecuredSlots:          c.Network.SecuredSlots,
		EncryptionKey:         c.Encryption.Key,
		AuthenticationService: authentication.NewService(log, c.AuthenticationService),
		ClaimService:          claim.NewService(log, c.ClaimService),
		VPNService:            vpn.NewService(log, c.VPNService),
		PingIndicator:         c.pingIndicator(),
		AFKTimer:              c.afkTimer(),
		Whitelist:             c.whiteList(log),
		Border:                c.makeBorder(),
		DialerFunction:        c.dialerFunc(log),
		Log:                   log,
	}
	session.SetCommandPath(c.Resources.CommandPath)

	err := c.loadCommands(log)
	if err != nil {
		return conf, fmt.Errorf("error loading commands: %w", err)
	}
	conf.StatusProvider, err = c.provider()
	if err != nil {
		return conf, fmt.Errorf("error creating status provider: %w", err)
	}
	conf.PlayerManager, err = NewPlayerManager(c.Network.PlayerManagerPath, log)
	if err != nil {
		return conf, fmt.Errorf("error creating player mamanger: %w", err)
	}
	conf.Listeners = append(conf.Listeners, c.listenerFunc)
	return conf, nil
}
