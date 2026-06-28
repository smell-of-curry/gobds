package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRateLimitHandlerThrottlesBackendNoise(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(newRateLimitHandler(slog.NewTextHandler(&buf, nil), time.Hour))

	// Repeated per-packet spam (different ephemeral ports) collapses to 1 line.
	log.Error("handle packet: read udp 127.0.0.1:53908->127.0.0.1:19134: connection refused", "src", "dialer")
	log.Error("handle packet: read udp 127.0.0.1:53999->127.0.0.1:19134: connection refused", "src", "dialer")
	log.Error("error dialing connection", "err", "dial raknet: connection refused")
	log.Error("error dialing connection", "err", "dial raknet: connection refused")
	// Noise detected only via an attr value (message itself is generic).
	log.Error("read failed", "err", "discover mtu: i/o timeout")
	log.Error("read failed", "err", "discover mtu: i/o timeout")

	out := buf.String()
	if got := strings.Count(out, "handle packet"); got != 1 {
		t.Fatalf("handle packet logged %d times, want 1", got)
	}
	if got := strings.Count(out, "error dialing connection"); got != 1 {
		t.Fatalf("error dialing logged %d times, want 1", got)
	}
	if got := strings.Count(out, "read failed"); got != 1 {
		t.Fatalf("attr-detected noise logged %d times, want 1", got)
	}

	// Normal, non-noise records must never be throttled.
	buf.Reset()
	log.Info("player connected", "srv", "GOLD")
	log.Info("player connected", "srv", "SILVER")
	if got := strings.Count(buf.String(), "player connected"); got != 2 {
		t.Fatalf("normal logs throttled: got %d, want 2", got)
	}
}

func TestRateLimitHandlerWindowExpiry(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(newRateLimitHandler(slog.NewTextHandler(&buf, nil), time.Millisecond))

	log.Error("handle packet: connection refused")
	time.Sleep(3 * time.Millisecond)
	log.Error("handle packet: connection refused")

	if got := strings.Count(buf.String(), "handle packet"); got != 2 {
		t.Fatalf("after window expiry want 2 lines, got %d", got)
	}
}
