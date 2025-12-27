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
	Servers               []*Server
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
	Log                   *slog.Logger
}

func (c UserConfig) Config(log *slog.Logger) (Config, error) {
	if len(c.Network.Servers) == 0 {
		return Config{}, fmt.Errorf("no servers configured")
	}

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
		Log:                   log,
	}
	session.SetCommandPath(c.Resources.CommandPath)

	err := c.loadCommands(log)
	if err != nil {
		return conf, fmt.Errorf("error loading commands: %w", err)
	}

	conf.PlayerManager, err = NewPlayerManager(c.Network.PlayerManagerPath, log)
	if err != nil {
		return conf, fmt.Errorf("error creating player mamanger: %w", err)
	}

	for _, server := range c.Network.Servers {
		localAddr := server.LocalAddress
		remoteAddr := server.RemoteAddress

		prov, err := minecraft.NewForeignStatusProvider(remoteAddr)
		if err != nil {
			return conf, fmt.Errorf("error creating status provider for %s: %w", remoteAddr, err)
		}

		name := server.Name
		srv := &Server{
			Name:          name,
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,

			DialerFunc:     c.dialerFunc(remoteAddr, log),
			StatusProvider: prov,
			Log:            log.With(slog.String("srv", name)),
		}

		srv.Listener, err = c.listenerFunc(srv)
		if err != nil {
			return conf, fmt.Errorf("error creating listener for %s: %w", localAddr, err)
		}

		conf.Servers = append(conf.Servers, srv)
	}
	return conf, nil
}
