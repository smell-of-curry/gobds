// Package gobds implements a bedrock proxy server for Minecraft with authentication and claim support.
package gobds

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service"
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
	AFKTimer              *infra.AFKTimer
	Whitelist             *whitelist.Whitelist
	Border                *area.Area2D
	ClaimPrefilter        bool
	ClaimDenyRendering    bool
	ClaimPollInterval     time.Duration
	ClaimMaxSnapshotAge   time.Duration
	TrafficProtection     session.TrafficConfig
	DuplicateXUIDEnabled  bool
	Log                   *slog.Logger
}

// Config converts the user configuration into a runtime configuration.
func (c UserConfig) Config(log *slog.Logger) (Config, error) {
	if len(c.Network.Servers) == 0 {
		return Config{}, fmt.Errorf("no servers configured")
	}

	pollInterval, err := claimDuration(c.Claims.PollInterval, claim.DefaultPollInterval)
	if err != nil {
		return Config{}, fmt.Errorf("claims poll interval: %w", err)
	}
	maxSnapshotAge, err := claimDuration(c.Claims.MaxSnapshotAge, claim.DefaultMaxSnapshotAge)
	if err != nil {
		return Config{}, fmt.Errorf("claims max snapshot age: %w", err)
	}
	if maxSnapshotAge < pollInterval {
		return Config{}, fmt.Errorf("claims max snapshot age must be at least poll interval")
	}

	conf := Config{
		SecuredSlots:          c.Network.SecuredSlots,
		EncryptionKey:         c.Encryption.Key,
		AuthenticationService: authentication.NewService(log, c.AuthenticationService),
		VPNService: vpn.NewService(log, service.Config{
			Enabled: c.VPNService.Enabled,
			URL:     c.VPNService.URL,
			Key:     c.VPNService.Key,
		}, c.VPNService.WhitelistedCIDRs),
		AFKTimer:             c.afkTimer(),
		Whitelist:            c.whiteList(log),
		Border:               c.makeBorder(),
		ClaimPrefilter:       c.Claims.PrefilterEnabled,
		ClaimDenyRendering:   c.Claims.DenyRenderingEnabled,
		ClaimPollInterval:    pollInterval,
		ClaimMaxSnapshotAge:  maxSnapshotAge,
		TrafficProtection:    c.TrafficProtection.WithDefaults(),
		DuplicateXUIDEnabled: c.DuplicateXUID.Enabled,
		Log:                  log,
	}
	session.SetCommandPath(c.Resources.CommandPath)

	err = c.loadCommands(log)
	if err != nil {
		return conf, fmt.Errorf("error loading commands: %w", err)
	}

	for _, server := range c.Network.Servers {
		srv := &Server{
			Name:          server.Name,
			LocalAddress:  server.LocalAddress,
			RemoteAddress: server.RemoteAddress,

			ClaimFactory: claim.NewFactory(
				server.ClaimService,
				server.Name,
				pollInterval,
				maxSnapshotAge,
				log.With(slog.String("srv", server.Name)),
			),
			TrafficMetrics: &session.TrafficMetrics{},

			DialerFunc: c.dialerFunc(server.RemoteAddress, log),

			Log: log.With(slog.String("srv", server.Name)),
		}
		motd := server.MOTD
		if motd == "" {
			motd = server.Name
		}
		maxPlayers := server.MaxPlayers
		if maxPlayers <= 0 {
			maxPlayers = 85
		}
		srv.StatusProviderFunc = func() (minecraft.ServerStatusProvider, error) {
			return newProxyStatusProvider(srv, motd, maxPlayers), nil
		}
		srv.ListenerFunc = func() (Listener, error) {
			return c.listenerFunc(srv)
		}
		conf.Servers = append(conf.Servers, srv)
	}
	return conf, nil
}

func claimDuration(value string, fallback time.Duration) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("must be a positive duration")
	}
	return duration, nil
}
