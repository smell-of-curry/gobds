package gobds

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/session"
)

func TestBackendDisconnectThroughProxy(t *testing.T) {
	const kick = "§cTemporary kick: contact staff.\nReference 83"
	tests := []struct {
		name string
		pkt  *packet.Disconnect
		want string
	}{
		{name: "visible kick", pkt: &packet.Disconnect{Message: kick}, want: kick},
		{name: "hidden kick", pkt: &packet.Disconnect{HideDisconnectionScreen: true, Message: kick}},
		{name: "empty kick", pkt: &packet.Disconnect{}},
		{name: "transport close"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			upstream, backend := disconnectConnectionPair(t, ctx, log)
			client, downstream := disconnectConnectionPair(t, ctx, log)
			proxy := session.Config{Client: downstream, Server: upstream, Log: log}.New()

			sessionDone := make(chan struct{})
			go func() {
				defer close(sessionDone)
				proxy.ReadPackets(ctx)
			}()

			// Prove these are spawned connections actually forwarding packets
			// before measuring the disconnect delivered to the downstream client.
			const ready = "backend connected"
			if err := backend.WritePacket(&packet.Text{TextType: packet.TextTypeRaw, Message: ready}); err != nil {
				t.Fatal(err)
			}
			if err := backend.Flush(); err != nil {
				t.Fatal(err)
			}
			if deadline, ok := ctx.Deadline(); ok {
				if err := client.SetReadDeadline(deadline); err != nil {
					t.Fatal(err)
				}
			}
			for {
				pk, err := client.ReadPacket()
				if err != nil {
					t.Fatalf("read forwarded backend greeting: %v", err)
				}
				if msg, ok := pk.(*packet.Text); ok && msg.Message == ready {
					break
				}
			}

			if tt.pkt != nil {
				if err := backend.WritePacket(tt.pkt); err != nil {
					t.Fatal(err)
				}
			}
			if err := backend.Close(); err != nil {
				t.Fatal(err)
			}

			var err error
			for err == nil {
				_, err = client.ReadPacket()
			}
			if errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timed out waiting for downstream disconnect: %v", err)
			}
			var reason minecraft.DisconnectError
			hasReason := errors.As(err, &reason)
			switch {
			case tt.want == "" && hasReason:
				t.Fatalf("expected a silent disconnect, received %q", string(reason))
			case tt.want != "" && (!hasReason || string(reason) != tt.want):
				t.Fatalf("expected player-facing kick %q, received %v", tt.want, err)
			}
			select {
			case <-sessionDone:
			case <-ctx.Done():
				t.Fatal("proxy session did not close")
			}
		})
	}
}

// disconnectConnectionPair completes a normal offline login and game spawn over
// RakNet and returns the dialing connection and the accepted connection.
func disconnectConnectionPair(t *testing.T, ctx context.Context, log *slog.Logger) (*minecraft.Conn, *minecraft.Conn) {
	t.Helper()
	listener, err := (minecraft.ListenConfig{AuthenticationDisabled: true, ErrorLog: log}).Listen("raknet", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	type acceptResult struct {
		conn *minecraft.Conn
		err  error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		c, err := listener.Accept()
		if err != nil {
			accepted <- acceptResult{err: err}
			return
		}
		conn := c.(*minecraft.Conn)
		accepted <- acceptResult{conn: conn, err: conn.StartGameContext(ctx, minecraft.GameData{})}
	}()

	dial := minecraft.Dialer{ErrorLog: log, IdentityData: login.IdentityData{DisplayName: "TestClient"}}
	client, err := dial.DialContext(ctx, "raknet", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.DoSpawnContext(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-accepted:
		if result.conn != nil {
			t.Cleanup(func() { _ = result.conn.Close() })
		}
		if result.err != nil {
			t.Fatal(result.err)
		}
		return client, result.conn
	case <-ctx.Done():
		t.Fatalf("waiting for RakNet server spawn: %v", ctx.Err())
		return nil, nil
	}
}
