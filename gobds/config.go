package gobds

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/head"
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
	HeadService           *head.Service
	SkinConfig            *infra.SkinConfig
	PingIndicator         *infra.PingIndicator
	AFKTimer              *infra.AFKTimer
	Whitelist             *whitelist.Whitelist
	Border                *area.Area2D
	PlayerManager         *PlayerManager
	DialerFunction        DialerFunc
	Log                   *slog.Logger
}

func (c UserConfig) Config(log *slog.Logger) (Config, error) {
	skinConf := c.skinConfig()
	conf := Config{
		SecuredSlots:          c.Network.SecuredSlots,
		EncryptionKey:         c.Encryption.Key,
		AuthenticationService: authentication.NewService(log, c.AuthenticationService),
		ClaimService:          claim.NewService(log, c.ClaimService),
		VPNService:            vpn.NewService(log, c.VPNService),
		HeadService:           head.NewService(log, c.HeadService, skinConf.HeadsDirectory),
		SkinConfig:            skinConf,
		PingIndicator:         c.pingIndicator(),
		AFKTimer:              c.afkTimer(),
		Whitelist:             c.whiteList(log),
		Border:                c.makeBorder(),
		DialerFunction:        c.dialerFunc(log),
		Log:                   log,
	}
	session.SetupRuntimeIDs(c.Network.HashedBlockIDS)
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
		return conf, fmt.Errorf("error creating player manager: %w", err)
	}
	conf.Listeners = append(conf.Listeners, c.listenerFunc)
	return conf, nil
}

func (c UserConfig) skinConfig() *infra.SkinConfig {
	cooldown, err := time.ParseDuration(c.SkinConfig.Cooldown)
	if err != nil {
		cooldown = 15 * time.Second
	}
	return &infra.SkinConfig{
		SkinChangeCooldown: cooldown,
		HeadsDirectory:     c.SkinConfig.HeadsDirectory,
		HeadServiceURL:     c.SkinConfig.HeadServiceURL,
	}
}
