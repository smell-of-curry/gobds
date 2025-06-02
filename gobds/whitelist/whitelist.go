package whitelist

import (
	"os"
	"strings"
	"sync"

	"github.com/restartfu/gophig"
)

// Whitelist ...
type Whitelist struct {
	entries []string
	mu      sync.RWMutex
}

// NewWhitelist ...
func NewWhitelist(entries []string) *Whitelist {
	return &Whitelist{entries: entries}
}

// Has ...
func (w *Whitelist) Has(name string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, entry := range w.entries {
		if strings.ToLower(entry) == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// defaultConfig ...
func defaultConfig() []string {
	return []string{"the glancist", "Smell of curry"}
}

// ReadConfig ...
func ReadConfig() ([]string, error) {
	g := gophig.NewGophig[[]string]("./whitelists.toml", gophig.TOMLMarshaler{}, os.ModePerm)
	_, err := g.LoadConf()
	if os.IsNotExist(err) {
		err = g.SaveConf(defaultConfig())
		if err != nil {
			return nil, err
		}
	}
	c, err := g.LoadConf()
	return c, err
}
