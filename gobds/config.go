package gobds

import (
	"os"

	"github.com/restartfu/gophig"
)

// Config ...
type Config struct {
	Network struct {
		LocalAddress  string
		RemoteAddress string

		Whitelisted bool

		SecuredSlots      int
		MaxRenderDistance int
		FlushRate         int
	}
	Border struct {
		Enabled    bool
		MinX, MinZ int32
		MaxX, MaxZ int32
	}
	Resources struct {
		PacksRequired bool

		CommandPath   string
		URLResources  []string
		PathResources []string
	}
	AuthenticationService struct {
		URL string
		Key string
	}
	ClaimService struct {
		URL string
		Key string
	}
	Encryption struct {
		Key string
	}
}

// DefaultConfig ...
func DefaultConfig() Config {
	c := Config{}

	c.Network.LocalAddress = "127.0.0.1:19132"
	c.Network.RemoteAddress = "127.0.0.1:19133"

	c.Network.Whitelisted = false

	c.Network.SecuredSlots = 10
	c.Network.MaxRenderDistance = 16
	c.Network.FlushRate = 20

	c.Border.Enabled = false

	c.Resources.PacksRequired = false
	c.Resources.CommandPath = "resources/commands.json"

	c.AuthenticationService.URL = "http://127.0.0.1:8080/authentication"
	c.AuthenticationService.Key = "secret-key"

	c.ClaimService.URL = "http://127.0.0.1:8080/fetch/claims"
	c.ClaimService.Key = "secret-key"

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
