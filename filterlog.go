package main

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// noiseLogWindow is how long a given kind of "backend unreachable" error is
// suppressed after it has been logged once. When a backend BDS server is down
// or restarting, gophertunnel's dialer/listener emit the same transient error
// ("connection refused", "discover mtu", "i/o timeout", "handle packet ...")
// for every packet, which floods the proxy log. Keeping one line per window
// preserves the signal without the flood.
const noiseLogWindow = 15 * time.Second

// rateLimiterState is shared across every handler derived via WithAttrs /
// WithGroup so the throttle is global rather than reset per child logger.
type rateLimiterState struct {
	window time.Duration
	mu     sync.Mutex
	last   map[string]time.Time
}

// allow reports whether a record with the given throttle key may be logged now,
// recording the time when it returns true.
func (s *rateLimiterState) allow(key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if last, ok := s.last[key]; ok && now.Sub(last) < s.window {
		return false
	}
	s.last[key] = now
	return true
}

// rateLimitHandler wraps an slog.Handler and throttles known high-volume
// "backend unreachable" error spam to one line per noiseLogWindow. Every other
// record passes through untouched.
type rateLimitHandler struct {
	inner slog.Handler
	state *rateLimiterState
}

// newRateLimitHandler wraps inner so that recognized backend-down log spam is
// throttled to one line per window.
//
// @param inner The underlying handler to forward non-suppressed records to.
// @param window How long to suppress repeats of the same noise kind.
// @returns an slog.Handler that throttles backend-unreachable spam.
func newRateLimitHandler(inner slog.Handler, window time.Duration) slog.Handler {
	return &rateLimitHandler{
		inner: inner,
		state: &rateLimiterState{window: window, last: make(map[string]time.Time)},
	}
}

func (h *rateLimitHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *rateLimitHandler) Handle(ctx context.Context, r slog.Record) error {
	if key := noiseKey(r); key != "" && !h.state.allow(key) {
		return nil
	}
	return h.inner.Handle(ctx, r)
}

func (h *rateLimitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &rateLimitHandler{inner: h.inner.WithAttrs(attrs), state: h.state}
}

func (h *rateLimitHandler) WithGroup(name string) slog.Handler {
	return &rateLimitHandler{inner: h.inner.WithGroup(name), state: h.state}
}

// noiseKey returns a stable throttle key for known backend-unreachable log
// spam, or "" if the record should always pass through. The key intentionally
// drops the variable parts (ephemeral ports, addresses) so all repeats collapse
// onto one throttle bucket.
//
// @param r The log record to classify.
// @returns a throttle key, or "" when the record is not backend-down noise.
func noiseKey(r slog.Record) string {
	if strings.Contains(r.Message, "handle packet") {
		return "handle-packet"
	}
	if strings.Contains(r.Message, "error dialing connection") {
		return "error-dialing"
	}
	key := ""
	r.Attrs(func(a slog.Attr) bool {
		v := a.Value.String()
		if strings.Contains(v, "connection refused") ||
			strings.Contains(v, "discover mtu") ||
			strings.Contains(v, "i/o timeout") {
			key = "dial-transient"
			return false
		}
		return true
	})
	return key
}
