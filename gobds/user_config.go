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
		ServerName string

		LocalAddress  string
		RemoteAddress string

		PlayerManagerPath string

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
	ClaimService struct {
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

// provider returns default provider.
func (c UserConfig) provider() (minecraft.ServerStatusProvider, error) {
	return minecraft.NewForeignStatusProvider(c.Network.RemoteAddress)
}

// afkTimer returns new AFKTimer instance.
func (c UserConfig) afkTimer() *infra.AFKTimer {
	if !c.AFKTimer.Enabled {
		return nil
	}
	d, err := time.ParseDuration(c.AFKTimer.TimeoutDuration)
	if err != nil {
		// Fallback to a sensible default to avoid crash on invalid config
		d = 10 * time.Minute
	}
	return &infra.AFKTimer{TimeoutDuration: d}
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

// dialerFunc returns default dialer func.
func (c UserConfig) dialerFunc(log *slog.Logger) DialerFunc {
	return func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error) {
		d := minecraft.Dialer{
			ClientData:   clientData,
			IdentityData: identityData,

			DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool { return false },

			FlushRate:           time.Millisecond * time.Duration(c.Network.FlushRate),
			ErrorLog:            log,
			KeepXBLIdentityData: true,
		}
		return d.DialContext(ctx, "raknet", c.Network.RemoteAddress)
	}
}

// listenerFunc returns default listener func.
func (c UserConfig) listenerFunc(conf Config) (Listener, error) {
	cfg := minecraft.ListenConfig{
		ErrorLog:       conf.Log,
		StatusProvider: conf.StatusProvider,

		FlushRate:            time.Millisecond * time.Duration(c.Network.FlushRate),
		ResourcePacks:        c.packs(conf.Log),
		TexturePacksRequired: c.Resources.PacksRequired,
	}

	if conf.Log.Enabled(context.Background(), slog.LevelDebug) {
		cfg.ErrorLog = conf.Log.With("net origin", "gophertunnel")
	}
	l, err := cfg.Listen("raknet", c.Network.LocalAddress)
	if err != nil {
		return nil, fmt.Errorf("create listener: %w", err)
	}
	conf.Log.Info("listener running.", "addr", l.Addr())
	return listener{l}, nil
}

// DefaultConfig ...
func DefaultConfig() UserConfig {
	c := UserConfig{}

	c.Network.ServerName = "Some server"

	c.Network.LocalAddress = "127.0.0.1:19132"
	c.Network.RemoteAddress = "127.0.0.1:19133"

	c.Network.PlayerManagerPath = "players/manager.json"

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

	c.Resources.PacksRequired = false
	c.Resources.CommandPath = "resources/commands.json"

	c.AuthenticationService.Enabled = false
	c.AuthenticationService.URL = "http://127.0.0.1:8080/authentication"
	const defaultKey = "secret-key"
	c.AuthenticationService.Key = defaultKey

	c.ClaimService.Enabled = false
	c.ClaimService.URL = "http://127.0.0.1:8080/fetch/claims"
	c.ClaimService.Key = defaultKey

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
