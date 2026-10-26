package gobds

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestWriteItemRegistry(t *testing.T) {
	t.Chdir(t.TempDir())
	srv := &Server{Name: "GOLD", Log: slog.Default()}
	items := []protocol.ItemEntry{
		{Name: "minecraft:stick", RuntimeID: 328},
		{Name: "pokeb:aluminum_boots", RuntimeID: 959},
	}

	read := func() map[string]int16 {
		t.Helper()
		data, err := os.ReadFile("item-registry-GOLD.json")
		if err != nil {
			t.Fatal(err)
		}
		var ids map[string]int16
		if err := json.Unmarshal(data, &ids); err != nil {
			t.Fatal(err)
		}
		return ids
	}

	srv.WriteItemRegistry(items)
	if got := read()["pokeb:aluminum_boots"]; got != 959 {
		t.Fatalf("aluminum_boots = %d, want 959", got)
	}

	// An unchanged registry is rewritten if the file was removed.
	if err := os.Remove("item-registry-GOLD.json"); err != nil {
		t.Fatal(err)
	}
	srv.WriteItemRegistry(items)
	read()

	items[1].RuntimeID = 3670
	srv.WriteItemRegistry(items)
	if got := read()["pokeb:aluminum_boots"]; got != 3670 {
		t.Fatalf("aluminum_boots = %d, want 3670", got)
	}
}
