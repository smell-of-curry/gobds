package gobds

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/restartfu/gophig"
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

func TestNetherNetConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	const raw = `
[Network]
[[Network.Servers]]
Name = 'nether'
LocalAddress = '127.0.0.1:19132'
RemoteAddress = '127.0.0.1:19133'
Transport = 'nethernet'
[Network.Servers.NetherNet]
Enabled = true
UDPPort = 19140
PublicIP = '1.2.3.4'
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	g := gophig.NewGophig[UserConfig](path, gophig.TOMLMarshaler{}, os.ModePerm)
	c, err := g.LoadConf()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Network.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(c.Network.Servers))
	}
	s := c.Network.Servers[0]
	if s.Transport != "nethernet" || !s.NetherNet.Enabled || s.NetherNet.UDPPort != 19140 || s.NetherNet.PublicIP != "1.2.3.4" {
		t.Fatalf("parsed %#v", s)
	}
	if err = g.SaveConf(c); err != nil {
		t.Fatal(err)
	}
	c2, err := g.LoadConf()
	if err != nil {
		t.Fatal(err)
	}
	s2 := c2.Network.Servers[0]
	if s2.Transport != s.Transport || s2.NetherNet != s.NetherNet {
		t.Fatalf("round-trip %#v vs %#v", s2, s)
	}

	c.Resources.CommandPath = filepath.Join(t.TempDir(), "commands.json")
	if err = os.WriteFile(c.Resources.CommandPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime, err := c.Config(slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(runtime.Servers[0].ListenerFuncs); got != 2 {
		t.Fatalf("ListenerFuncs = %d, want 2", got)
	}
}

func TestInvalidTransportRejected(t *testing.T) {
	config := UserConfig{}
	config.Network.Servers = []ServerConfig{{
		Name:          "test",
		LocalAddress:  "127.0.0.1:19132",
		RemoteAddress: "127.0.0.1:19133",
		Transport:     "webrtc",
	}}
	config.Resources.CommandPath = filepath.Join(t.TempDir(), "commands.json")
	if err := os.WriteFile(config.Resources.CommandPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Config(slog.Default()); err == nil {
		t.Fatal("expected invalid transport error")
	}
}

func TestNetherNetEnabledRequiresUDPPort(t *testing.T) {
	config := UserConfig{}
	config.Network.Servers = []ServerConfig{{
		Name:          "test",
		LocalAddress:  "127.0.0.1:19132",
		RemoteAddress: "127.0.0.1:19133",
		NetherNet:     NetherNetConfig{Enabled: true, UDPPort: 0},
	}}
	config.Resources.CommandPath = filepath.Join(t.TempDir(), "commands.json")
	if err := os.WriteFile(config.Resources.CommandPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Config(slog.Default()); err == nil {
		t.Fatal("expected UDPPort required error")
	}
}
