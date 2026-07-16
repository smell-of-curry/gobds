package gobds

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/session"
)

func TestClaimHardeningDefaultsOff(t *testing.T) {
	config := DefaultConfig()
	if config.Claims.PrefilterEnabled || config.Claims.DenyRenderingEnabled {
		t.Fatal("claim prefilter and deny rendering must default off")
	}
}

func TestClaimDurationsDefaultDuringConfigConversion(t *testing.T) {
	config := UserConfig{}
	config.Network.Servers = []ServerConfig{{
		Name:          "test",
		LocalAddress:  "127.0.0.1:19132",
		RemoteAddress: "127.0.0.1:19133",
	}}
	config.Resources.CommandPath = filepath.Join(t.TempDir(), "commands.json")
	if err := os.WriteFile(config.Resources.CommandPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime, err := config.Config(slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.ClaimPollInterval != claim.DefaultPollInterval ||
		runtime.ClaimMaxSnapshotAge != claim.DefaultMaxSnapshotAge {
		t.Fatalf(
			"unexpected defaults: poll=%s maxAge=%s",
			runtime.ClaimPollInterval,
			runtime.ClaimMaxSnapshotAge,
		)
	}
	if runtime.Servers[0].ClaimFactory.PollInterval() != claim.DefaultPollInterval {
		t.Fatal("factory did not receive default poll interval")
	}
	defaultTraffic := session.DefaultTrafficConfig()
	if runtime.TrafficProtection.Enforce ||
		runtime.TrafficProtection.Chat != defaultTraffic.Chat ||
		runtime.TrafficProtection.MaxInventoryActions != defaultTraffic.MaxInventoryActions {
		t.Fatalf("missing traffic section did not receive defaults: %+v", runtime.TrafficProtection)
	}
}
