package session

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const trafficCategories = 6

const (
	trafficChat = iota
	trafficCommand
	trafficForm
	trafficInventory
	trafficStack
	trafficHandler
)

var trafficCategoryNames = [trafficCategories]string{
	"chat", "command", "form", "inventory", "item_stack", "handler",
}

// RateLimit configures one per-session token bucket.
type RateLimit struct {
	Rate  float64
	Burst int
}

// TrafficConfig bounds malformed packets and optionally enforces rate limits.
type TrafficConfig struct {
	Enforce bool

	Chat                  RateLimit
	Commands              RateLimit
	ModalFormResponses    RateLimit
	InventoryTransactions RateLimit
	ItemStackRequests     RateLimit

	MaxTextBytes          int
	MaxCommandBytes       int
	MaxFormResponseBytes  int
	MaxFormResponseValues int
	MaxInventoryActions   int
	MaxStackRequests      int
	MaxStackActions       int
	MaxTotalStackActions  int
}

// DefaultTrafficConfig returns safe observe-only limits.
func DefaultTrafficConfig() TrafficConfig {
	return TrafficConfig{
		Chat:                  RateLimit{Rate: 8, Burst: 16},
		Commands:              RateLimit{Rate: 4, Burst: 8},
		ModalFormResponses:    RateLimit{Rate: 10, Burst: 20},
		InventoryTransactions: RateLimit{Rate: 60, Burst: 120},
		ItemStackRequests:     RateLimit{Rate: 40, Burst: 80},
		MaxTextBytes:          4 << 10,
		MaxCommandBytes:       4 << 10,
		MaxFormResponseBytes:  64 << 10,
		MaxFormResponseValues: 128,
		MaxInventoryActions:   128,
		MaxStackRequests:      64,
		MaxStackActions:       128,
		MaxTotalStackActions:  256,
	}
}

func setDefaultRate(v *float64, def float64) {
	if *v <= 0 {
		*v = def
	}
}

func setDefaultBurst(v *int, def int) {
	if *v <= 0 {
		*v = def
	}
}

// WithDefaults fills zero values, including when the config section is missing.
func (c TrafficConfig) WithDefaults() TrafficConfig {
	defaults := DefaultTrafficConfig()
	setDefaultRate(&c.Chat.Rate, defaults.Chat.Rate)
	setDefaultBurst(&c.Chat.Burst, defaults.Chat.Burst)
	setDefaultRate(&c.Commands.Rate, defaults.Commands.Rate)
	setDefaultBurst(&c.Commands.Burst, defaults.Commands.Burst)
	setDefaultRate(&c.ModalFormResponses.Rate, defaults.ModalFormResponses.Rate)
	setDefaultBurst(&c.ModalFormResponses.Burst, defaults.ModalFormResponses.Burst)
	setDefaultRate(&c.InventoryTransactions.Rate, defaults.InventoryTransactions.Rate)
	setDefaultBurst(&c.InventoryTransactions.Burst, defaults.InventoryTransactions.Burst)
	setDefaultRate(&c.ItemStackRequests.Rate, defaults.ItemStackRequests.Rate)
	setDefaultBurst(&c.ItemStackRequests.Burst, defaults.ItemStackRequests.Burst)
	setDefaultBurst(&c.MaxTextBytes, defaults.MaxTextBytes)
	setDefaultBurst(&c.MaxCommandBytes, defaults.MaxCommandBytes)
	setDefaultBurst(&c.MaxFormResponseBytes, defaults.MaxFormResponseBytes)
	setDefaultBurst(&c.MaxFormResponseValues, defaults.MaxFormResponseValues)
	setDefaultBurst(&c.MaxInventoryActions, defaults.MaxInventoryActions)
	setDefaultBurst(&c.MaxStackRequests, defaults.MaxStackRequests)
	setDefaultBurst(&c.MaxStackActions, defaults.MaxStackActions)
	setDefaultBurst(&c.MaxTotalStackActions, defaults.MaxTotalStackActions)
	return c
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(limit RateLimit, now time.Time) tokenBucket {
	return tokenBucket{rate: limit.Rate, burst: float64(limit.Burst), tokens: float64(limit.Burst), last: now}
}

func (b *tokenBucket) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// TrafficMetrics contains dependency-free atomic traffic counters.
type TrafficMetrics struct {
	observed  [trafficCategories]atomic.Uint64
	exceeded  [trafficCategories]atomic.Uint64
	enforced  [trafficCategories]atomic.Uint64
	malformed [trafficCategories]atomic.Uint64
}

func (m *TrafficMetrics) observe(category int, exceeded, enforced bool) {
	if m == nil || category < 0 || category >= trafficCategories {
		return
	}
	m.observed[category].Add(1)
	if exceeded {
		m.exceeded[category].Add(1)
	}
	if enforced {
		m.enforced[category].Add(1)
	}
}

func (m *TrafficMetrics) malformedPacket(category int) {
	if m != nil && category >= 0 && category < trafficCategories {
		m.malformed[category].Add(1)
	}
}

type trafficMetricRecord struct {
	Type       string                    `json:"type"`
	Server     string                    `json:"server"`
	Session    string                    `json:"session,omitempty"`
	PeriodMS   int64                     `json:"period_ms"`
	Categories [trafficCategories]string `json:"categories"`
	Observed   [trafficCategories]uint64 `json:"observed"`
	Exceeded   [trafficCategories]uint64 `json:"exceeded"`
	Enforced   [trafficCategories]uint64 `json:"enforced"`
	Malformed  [trafficCategories]uint64 `json:"malformed"`
}

// WriteDelta emits one compact JSON record and resets interval counters.
func (m *TrafficMetrics) WriteDelta(output io.Writer, server, session string, period time.Duration) {
	if m == nil {
		return
	}
	record := trafficMetricRecord{
		Type:       "traffic_protection_metrics",
		Server:     server,
		Session:    session,
		PeriodMS:   period.Milliseconds(),
		Categories: trafficCategoryNames,
	}
	for i := range trafficCategories {
		record.Observed[i] = m.observed[i].Swap(0)
		record.Exceeded[i] = m.exceeded[i].Swap(0)
		record.Enforced[i] = m.enforced[i].Swap(0)
		record.Malformed[i] = m.malformed[i].Swap(0)
	}
	raw, err := json.Marshal(record)
	if err == nil {
		_, _ = fmt.Fprintln(output, string(raw))
	}
}

type trafficState struct {
	config    TrafficConfig
	buckets   [trafficCategories]tokenBucket
	session   TrafficMetrics
	aggregate *TrafficMetrics
}

func newTrafficState(config TrafficConfig, aggregate *TrafficMetrics) trafficState {
	config = config.WithDefaults()
	now := time.Now()
	return trafficState{
		config: config,
		buckets: [trafficCategories]tokenBucket{
			newTokenBucket(config.Chat, now),
			newTokenBucket(config.Commands, now),
			newTokenBucket(config.ModalFormResponses, now),
			newTokenBucket(config.InventoryTransactions, now),
			newTokenBucket(config.ItemStackRequests, now),
		},
		aggregate: aggregate,
	}
}

func (t *trafficState) allow(category int) bool {
	exceeded := !t.buckets[category].allow(time.Now())
	enforced := exceeded && t.config.Enforce
	t.session.observe(category, exceeded, enforced)
	t.aggregate.observe(category, exceeded, enforced)
	return !enforced
}

func (t *trafficState) malformed(category int) {
	t.session.malformedPacket(category)
	t.aggregate.malformedPacket(category)
}
