package gobds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/restartfu/gophig"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/smell-of-curry/gobds/gobds/cmd"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
	"github.com/smell-of-curry/gobds/gobds/whitelist"
)

// UserConfig ...
type UserConfig struct {
	Network struct {
		ServerRegion string

		Servers []ServerConfig

		Whitelisted   bool
		WhitelistPath string

		SecuredSlots      int
		MaxRenderDistance int
		FlushRate         int

		SentryDSN string
	}
	Border struct {
		Enabled    bool
		MinX, MinZ int32
		MaxX, MaxZ int32
	}
	PingIndicator struct {
		Enabled    bool
		Identifier string
	}
	AFKTimer struct {
		Enabled         bool
		TimeoutDuration string
		// WarnApproaching is how long a player must be idle before receiving
		// an "AFK in 1 minute" soft warning. Always sent regardless of fullness.
		WarnApproaching string
		// MarkAFK is how long a player must be idle before being told they
		// are now AFK. Always sent regardless of fullness.
		MarkAFK string
		// FinalWarning is how long a player must be idle before receiving the
		// near-capacity hard warning. Only sent when fullness >= FullnessThreshold.
		FinalWarning string
		// FullnessThreshold is the fraction (0..1) of MaxPlayers at or above
		// which the proxy will start kicking AFK players, longest-AFK first.
		FullnessThreshold float64
	}
	Resources struct {
		PacksRequired bool

		CommandPath   string
		URLResources  []string
		PathResources []string
	}
	AuthenticationService struct {
		Enabled bool
		URL     string
		Key     string
	}
	VPNService struct {
		Enabled bool
		URL     string
		Key     string
	}
	Encryption struct {
		Key string
	}
}

// packs loads and returns all packs.
func (c UserConfig) packs(log *slog.Logger) []*resource.Pack {
	packs := make([]*resource.Pack, 0, len(c.Resources.URLResources)+len(c.Resources.PathResources))

	for _, url := range c.Resources.URLResources {
		pack, err := resource.ReadURL(url)
		if err != nil {
			log.Error("failed to load url pack", "err", err)
			continue
		}
		packs = append(packs, pack)
	}
	for _, path := range c.Resources.PathResources {
		pack, err := resource.ReadPath(path)
		if err != nil {
			log.Error("failed to load path pack", "err", err)
			continue
		}
		packs = append(packs, pack)
	}

	for _, pack := range packs {
		err := translator.Setup(pack)
		if err != nil {
			log.Error("failed to setup translator", "err", err)
		}
	}
	return packs
}

// loadCommands loads commands from the file.
func (c UserConfig) loadCommands(log *slog.Logger) error {
	rawBytes, err := os.ReadFile(c.Resources.CommandPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dir := filepath.Dir(c.Resources.CommandPath)
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				panic(err)
			}
			if createErr := os.WriteFile(c.Resources.CommandPath, []byte("{}"), os.ModePerm); createErr != nil {
				return createErr
			}
			rawBytes, err = os.ReadFile(c.Resources.CommandPath)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	var commands map[string]cmd.EngineResponseCommand
	if err = json.Unmarshal(rawBytes, &commands); err != nil {
		log.Error("failed to unmarshal commands", "err", err)
		return err
	}
	cmd.LoadFrom(commands)
	log.Info("loaded commands", "count", len(commands))

	return nil
}

// makeBorder returns new border instance.
func (c UserConfig) makeBorder() *area.Area2D {
	if !c.Border.Enabled {
		return nil
	}
	return area.NewArea2D(c.Border.MinX, c.Border.MinZ, c.Border.MaxX, c.Border.MaxZ)
}

// afkTimer returns new AFKTimer instance.
func (c UserConfig) afkTimer() *infra.AFKTimer {
	if !c.AFKTimer.Enabled {
		return nil
	}
	parse := func(s string, fallback time.Duration) time.Duration {
		if s == "" {
			return fallback
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return fallback
		}
		return d
	}

	timeout := parse(c.AFKTimer.TimeoutDuration, 10*time.Minute)
	warn := parse(c.AFKTimer.WarnApproaching, 4*time.Minute)
	mark := parse(c.AFKTimer.MarkAFK, 5*time.Minute)
	final := parse(c.AFKTimer.FinalWarning, 9*time.Minute)

	threshold := c.AFKTimer.FullnessThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.9
	}

	return &infra.AFKTimer{
		TimeoutDuration:   timeout,
		WarnApproaching:   warn,
		MarkAFK:           mark,
		FinalWarning:      final,
		FullnessThreshold: threshold,
	}
}

// pingIndicator returns new PingIndicator instance.
func (c UserConfig) pingIndicator() *infra.PingIndicator {
	if !c.PingIndicator.Enabled {
		return nil
	}
	return &infra.PingIndicator{Identifier: c.PingIndicator.Identifier}
}

// whiteList returns new Whitelist instance.
func (c UserConfig) whiteList(log *slog.Logger) *whitelist.Whitelist {
	if !c.Network.Whitelisted {
		return nil
	}
	conf, err := whitelist.ReadConfig(c.Network.WhitelistPath)
	if err != nil {
		log.Error("failed to read whitelist config", "err", err)
		return nil
	}
	return whitelist.NewWhitelist(conf.Entries)
}

// dialerFunc returns a dialer func for a specific server.
func (c UserConfig) dialerFunc(remoteAddress string, log *slog.Logger) DialerFunc {
	return func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error) {
		d := minecraft.Dialer{
			ClientData:   clientData,
			IdentityData: identityData,

			DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool { return false },

			FlushRate:           time.Millisecond * time.Duration(c.Network.FlushRate),
			ErrorLog:            log,
			KeepXBLIdentityData: true,
		}
		return d.DialContext(ctx, "raknet", remoteAddress)
	}
}

// listenerFunc returns a listener func for a specific server.
func (c UserConfig) listenerFunc(srv *Server) (Listener, error) {
	cfg := minecraft.ListenConfig{
		ErrorLog:       srv.Log,
		StatusProvider: srv.StatusProvider,

		FlushRate:            time.Millisecond * time.Duration(c.Network.FlushRate),
		ResourcePacks:        c.packs(srv.Log),
		TexturePacksRequired: c.Resources.PacksRequired,
	}

	if srv.Log.Enabled(context.Background(), slog.LevelDebug) {
		cfg.ErrorLog = srv.Log.With("net origin", "gophertunnel")
	}
	l, err := cfg.Listen("raknet", srv.LocalAddress)
	if err != nil {
		return nil, fmt.Errorf("create listener: %w", err)
	}
	srv.Log.Info("listener running.", "addr", l.Addr())
	return listener{l}, nil
}

// DefaultConfig ...
func DefaultConfig() UserConfig {
	c := UserConfig{}

	c.Network.ServerRegion = "Some region"

	const defaultKey = "secret-key"
	c.Network.Servers = []ServerConfig{
		{
			Name:          "Some server",
			LocalAddress:  "127.0.0.1:19132",
			RemoteAddress: "127.0.0.1:19133",
			ClaimService: struct {
				Enabled bool
				URL     string
				Key     string
			}{
				Enabled: false,
				URL:     "http://127.0.0.1:8080/fetch/claims",
				Key:     defaultKey,
			},
		},
	}

	c.Network.Whitelisted = false
	c.Network.WhitelistPath = "whitelist.json"

	c.Network.SecuredSlots = 0
	c.Network.MaxRenderDistance = 16
	c.Network.FlushRate = 20

	c.Border.Enabled = false

	c.PingIndicator.Enabled = true
	c.PingIndicator.Identifier = "&_playerPing:"

	c.AFKTimer.Enabled = true
	c.AFKTimer.TimeoutDuration = "10m"
	c.AFKTimer.WarnApproaching = "4m"
	c.AFKTimer.MarkAFK = "5m"
	c.AFKTimer.FinalWarning = "9m"
	c.AFKTimer.FullnessThreshold = 0.9

	c.Resources.PacksRequired = false
	c.Resources.CommandPath = "resources/commands.json"

	c.AuthenticationService.Enabled = false
	c.AuthenticationService.URL = "http://127.0.0.1:8080/authentication"
	c.AuthenticationService.Key = defaultKey

	c.VPNService.Enabled = false
	c.VPNService.URL = "http://ip-api.com/json"

	c.Encryption.Key = defaultKey
	return c
}

// ReadConfig ...
func ReadConfig() (UserConfig, error) {
	g := gophig.NewGophig[UserConfig]("./config.toml", gophig.TOMLMarshaler{}, os.ModePerm)
	_, err := g.LoadConf()
	if os.IsNotExist(err) {
		err = g.SaveConf(DefaultConfig())
		if err != nil {
			return UserConfig{}, err
		}
	}
	c, err := g.LoadConf()
	return c, err
}
