package gobds

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// itemRegistryPath is where bds-manager reads the registry for this server.
func (s *Server) itemRegistryPath() string {
	return "item-registry-" + s.Name + ".json"
}

// WriteItemRegistry writes the item name to runtime ID map BDS sent this
// session to item-registry-<name>.json in the working directory. BDS numbers
// add-on items in the host filesystem's directory order, so every host has its
// own layout; bds-manager serves this file to the behavior pack, which needs
// the real IDs to render item icons in forms.
func (s *Server) WriteItemRegistry(items []protocol.ItemEntry) {
	ids := make(map[string]int16, len(items))
	for _, item := range items {
		ids[item.Name] = item.RuntimeID
	}
	// encoding/json sorts map keys, so an unchanged registry is byte-identical.
	data, err := json.Marshal(ids)
	if err != nil {
		s.Log.Error("failed to encode item registry", "err", err)
		return
	}

	s.itemRegistryMu.Lock()
	defer s.itemRegistryMu.Unlock()
	path := s.itemRegistryPath()
	if bytes.Equal(data, s.itemRegistry) {
		if _, err := os.Stat(path); err == nil {
			return
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		s.Log.Error("failed to write item registry", "err", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		s.Log.Error("failed to write item registry", "err", err)
		return
	}
	s.itemRegistry = data
}
