package whitelist

import (
	"os"

	"github.com/restartfu/gophig"
)

// Config ...
type Config struct {
	Entries []string
}

// defaultConfig ...
func defaultConfig() Config {
	return Config{
		Entries: []string{"the glancist", "Smell of curry"},
	}
}

// ReadConfig ...
func ReadConfig() (Config, error) {
	g := gophig.NewGophig[Config]("./whitelists.toml", gophig.TOMLMarshaler{}, os.ModePerm)
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
