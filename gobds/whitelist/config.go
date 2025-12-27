// Package whitelist provides whitelist management functionality for the GoBDS proxy.
package whitelist

import (
	"os"

	"github.com/restartfu/gophig"
)

// Config ...
type Config struct {
	Entries []string `json:"entries"`
}

// defaultConfig ...
func defaultConfig() Config {
	return Config{
		Entries: []string{"the glancist", "Smell of curry"},
	}
}

// ReadConfig ...
func ReadConfig(path string) (Config, error) {
	g := gophig.NewGophig[Config](path, gophig.JSONMarshaler{}, os.ModePerm)
	_, err := g.LoadConf()
	if os.IsNotExist(err) {
		err = g.SaveConf(defaultConfig())
		if err != nil {
			return Config{}, err
		}
	}
	c, err := g.LoadConf()
	return c, err
}
