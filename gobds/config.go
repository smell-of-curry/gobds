// Package gobds implements a bedrock proxy server for Minecraft with authentication and claim support.
package gobds

import (
	"fmt"
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/whitelist"
)

// Config ...
type Config struct {
	Servers               []*Server
	SecuredSlots          int
	EncryptionKey         string
	AuthenticationService *authentication.Service
	VPNService            *vpn.Service
	PingIndicator         *infra.PingIndicator
	AFKTimer              *infra.AFKTimer
	Whitelist             *whitelist.Whitelist
	Border                *area.Area2D
	PlayerManager         *PlayerManager
	Log                   *slog.Logger
}

// Config converts the user configuration into a runtime configuration.
func (c UserConfig) Config(log *slog.Logger) (Config, error) {
	if len(c.Network.Servers) == 0 {
		return Config{}, fmt.Errorf("no servers configured")
	}

	conf := Config{
		SecuredSlots:          c.Network.SecuredSlots,
		EncryptionKey:         c.Encryption.Key,
		AuthenticationService: authentication.NewService(log, c.AuthenticationService),
		VPNService:            vpn.NewService(log, c.VPNService),
		PingIndicator:         c.pingIndicator(),
		AFKTimer:              c.afkTimer(),
		Whitelist:             c.whiteList(log),
		Border:                c.makeBorder(),
		Log:                   log,
	}
	session.SetupRuntimeIDs(c.Network.HashedBlockIDS)
	session.SetCommandPath(c.Resources.CommandPath)

	err := c.loadCommands(log)
	if err != nil {
		return conf, fmt.Errorf("error loading commands: %w", err)
	}

	conf.PlayerManager, err = NewPlayerManager(c.Network.PlayerManagerPath, log)
	if err != nil {
		return conf, fmt.Errorf("error creating player manager: %w", err)
	}

	for _, server := range c.Network.Servers {
		srv := &Server{
			Name:          server.Name,
			LocalAddress:  server.LocalAddress,
			RemoteAddress: server.RemoteAddress,

			EntityFactory: entity.NewFactory(),
			ClaimFactory:  claim.NewFactory(server.ClaimService, log),

			DialerFunc: c.dialerFunc(server.RemoteAddress, log),

			Log: log.With(slog.String("srv", server.Name)),
		}
		srv.StatusProviderFunc = func() (minecraft.ServerStatusProvider, error) {
			return minecraft.NewForeignStatusProvider(server.RemoteAddress)
		}
		srv.ListenerFunc = func() (Listener, error) {
			return c.listenerFunc(srv)
		}
		conf.Servers = append(conf.Servers, srv)
	}
	return conf, nil
}
