package session

import (
	"log/slog"
	"testing"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type pingTestConn struct {
	Conn
	latency time.Duration
	written []packet.Packet
}

func (c *pingTestConn) GameData() minecraft.GameData { return minecraft.GameData{} }
func (c *pingTestConn) Latency() time.Duration       { return c.latency }
func (c *pingTestConn) ClientData() login.ClientData {
	return login.ClientData{ThirdPartyName: "Fixture"}
}
func (c *pingTestConn) IdentityData() login.IdentityData { return login.IdentityData{XUID: "100"} }
func (c *pingTestConn) WritePacket(pk packet.Packet) error {
	c.written = append(c.written, pk)
	return nil
}

func TestPingForwardingFromAuthInput(t *testing.T) {
	for _, tc := range []struct {
		name     string
		disabled bool
		messages []string
	}{
		{name: "enabled by default", messages: []string{"[PROXY_PING] 17", "[PROXY_PING] 24"}},
		{name: "disabled", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, backend := &pingTestConn{latency: 17 * time.Millisecond}, &pingTestConn{}
			s := (Config{Client: client, Server: backend, DisablePingIndicator: tc.disabled, Log: slog.Default()}).New()
			s.SetWarnedApproaching(true)
			s.SetMarkedAFK(true)
			s.SetWarnedFinal(true)
			pos := mgl32.Vec3{4, 63, 7}
			for _, tick := range []uint64{1, 20, 40, 60} {
				if tick == 60 {
					client.latency = 24 * time.Millisecond
				}
				input := &packet.PlayerAuthInput{Position: pos, Yaw: 90, Pitch: 10, Tick: tick}
				if forward, err := s.handlePacket(input, client); !forward || err != nil {
					t.Fatalf("movement packet %d was rejected: forward=%t, err=%v", tick, forward, err)
				}
				if tick == 1 && len(backend.written) != 0 {
					t.Fatal("ping sent outside the sampling tick")
				}
			}
			if yaw, pitch := s.Look(); s.Position() != pos || yaw != 90 || pitch != 10 ||
				s.WarnedApproaching() || s.MarkedAFK() || s.WarnedFinal() {
				t.Fatal("movement must update the sampled position and clear AFK flags")
			}
			if got := len(backend.written); got != len(tc.messages) {
				t.Fatalf("ping chat packet count = %d, want %d", got, len(tc.messages))
			}
			for i, expected := range tc.messages {
				text, ok := backend.written[i].(*packet.Text)
				if !ok || text.Message != expected || text.TextType != packet.TextTypeChat ||
					text.SourceName != "Fixture" || text.XUID != "100" {
					t.Fatalf("chat packet %d = %#v, want %q and session identity", i, backend.written[i], expected)
				}
			}
		})
	}
}
