package gobds

import (
	"os"

	"github.com/restartfu/gophig"
	"github.com/smell-of-curry/gobds/gobds/util"
)

// Config ...
type Config struct {
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
		TimeoutDuration util.Duration
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

// DefaultConfig ...
func DefaultConfig() Config {
	c := Config{}

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

	c.Resources.PacksRequired = false
	c.Resources.CommandPath = "resources/commands.json"

	c.AuthenticationService.Enabled = false
	c.AuthenticationService.URL = "http://127.0.0.1:8080/authentication"
	c.AuthenticationService.Key = "secret-key"

	c.ClaimService.Enabled = false
	c.ClaimService.URL = "http://127.0.0.1:8080/fetch/claims"
	c.ClaimService.Key = "secret-key"

	c.VPNService.Enabled = false
	c.VPNService.URL = "http://ip-api.com/json"

	c.Encryption.Key = "secret-key"
	return c
}

// ReadConfig ...
func ReadConfig() (Config, error) {
	g := gophig.NewGophig[Config]("./config.toml", gophig.TOMLMarshaler{}, os.ModePerm)
	_, err := g.LoadConf()
	if os.IsNotExist(err) {
		err = g.SaveConf(DefaultConfig())
		if err != nil {
			return Config{}, err
		}
	}
	c, err := g.LoadConf()
	return c, err
}
