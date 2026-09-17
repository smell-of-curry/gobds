package gobds

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
)

func TestNetherNetListenerSmoke(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ep, listenCfg, err := newNetherNetStack(netherNetListenOpts{
		tcpAddr:        "127.0.0.1:0",
		udpPort:        0,
		udpIP:          net.ParseIP("127.0.0.1"),
		publicIP:       "127.0.0.1",
		allowAnonymous: true,
		log:            log,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ep.close() }()

	joinURL := "http://" + ep.tcp.Addr().String() + "/v1/join"
	var resp *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err = http.Get(joinURL)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET /v1/join: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		t.Fatalf("GET /v1/join status = %d", resp.StatusCode)
	}

	l, err := listenCfg.Listen(ep.handler)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	accepted := make(chan net.Conn, 1)
	errc := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			errc <- err
			return
		}
		accepted <- c
	}()

	conn, err := nethernet.Dialer{DisableTrickleICE: true}.DialContext(ctx, "http://"+ep.tcp.Addr().String(), endpoint.NewClient())
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	defer func() { _ = conn.Close() }()

	select {
	case ac := <-accepted:
		_ = ac.Close()
	case err := <-errc:
		t.Fatalf("Accept: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for Accept")
	}
}

func TestNetherNetDialAddressRewritesUnspecified(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"127.0.0.1:19133", "http://127.0.0.1:19133"},
		{"0.0.0.0:19133", "http://127.0.0.1:19133"},
		{":19133", "http://127.0.0.1:19133"},
		{"[::]:19133", "http://127.0.0.1:19133"},
	}
	for _, tt := range tests {
		if got := netherNetDialAddress(tt.in); got != tt.want {
			t.Errorf("netherNetDialAddress(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
