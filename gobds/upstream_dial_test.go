package gobds

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TestUpstreamDialAcceptsLargeBatch is the prod failure from BDS 1.26.60:
// "decompress batch: decompress flate: size exceeds limit 16777216".
func TestUpstreamDialAcceptsLargeBatch(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	l, err := minecraft.ListenConfig{AuthenticationDisabled: true, ErrorLog: log}.Listen("raknet", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	const payloadLen = 20 << 20
	serverErr := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		conn := c.(*minecraft.Conn)
		defer func() { _ = conn.Close() }()
		if err := conn.StartGame(minecraft.GameData{}); err != nil {
			serverErr <- err
			return
		}
		if err := conn.WritePacket(&packet.LevelChunk{RawPayload: make([]byte, payloadLen)}); err != nil {
			serverErr <- err
			return
		}
		serverErr <- conn.Flush()
		time.Sleep(5 * time.Second)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dial := UserConfig{}.dialerFunc(l.Addr().String(), "raknet", log)
	conn, err := dial(login.IdentityData{DisplayName: "batch"}, login.ClientData{}, ctx)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if chunk, ok := pk.(*packet.LevelChunk); ok {
			if len(chunk.RawPayload) != payloadLen {
				t.Fatalf("payload %d bytes, want %d", len(chunk.RawPayload), payloadLen)
			}
			break
		}
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server: %v", err)
	}
}
