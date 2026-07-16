package claim

import (
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

const (
	metricActions  = 9
	metricReasons  = 8
	latencyBuckets = 5
)

var (
	actionMetricNames = [metricActions]string{
		"render", "block_break", "block_place", "block_interact", "entity_interact",
		"entity_hurt", "item_release", "item_throw", "item_drop",
	}
	reasonMetricNames = [metricReasons]string{
		"ready", "missing", "stale", "unsupported", "unknown_dimension", "invalid",
		"overlap", "refresh_failed",
	}
)

// Metrics contains per-server dependency-free atomic claim proxy counters.
type Metrics struct {
	server string

	refreshAttempts atomic.Uint64
	refreshSuccess  atomic.Uint64
	refreshFailure  atomic.Uint64
	packets         atomic.Uint64
	seen            [metricActions]atomic.Uint64
	forwarded       [metricActions]atomic.Uint64
	denied          [metricActions]atomic.Uint64
	reasons         [metricReasons]atomic.Uint64
	candidates      atomic.Uint64
	latency         [latencyBuckets]atomic.Uint64
	correctionsSent atomic.Uint64
	correctionsSkip atomic.Uint64
	subchunkDecode  atomic.Uint64
	subchunkModify  atomic.Uint64
	subchunkError   atomic.Uint64
}

// NewMetrics creates metrics isolated to one configured server.
func NewMetrics(server string) *Metrics {
	return &Metrics{server: server}
}

// RefreshAttempt records one claim-refresh attempt.
func (m *Metrics) RefreshAttempt() { m.refreshAttempts.Add(1) }

// RefreshSuccess records one successful claim refresh.
func (m *Metrics) RefreshSuccess() { m.refreshSuccess.Add(1) }

// RefreshFailure records one failed claim refresh.
func (m *Metrics) RefreshFailure() { m.refreshFailure.Add(1) }

// Packet records one processed claim-related packet.
func (m *Metrics) Packet() { m.packets.Add(1) }

// Action records one policy decision by fixed action index.
func (m *Metrics) Action(action uint8, forwarded bool) {
	if m == nil || int(action) >= metricActions {
		return
	}
	m.seen[action].Add(1)
	if forwarded {
		m.forwarded[action].Add(1)
		return
	}
	m.denied[action].Add(1)
}

// Reason records a fail-open reason by fixed reason index.
func (m *Metrics) Reason(reason QueryStatus) {
	if m == nil || int(reason) >= metricReasons {
		return
	}
	m.reasons[reason].Add(1)
}

// Candidates records number of spatial candidates examined.
func (m *Metrics) Candidates(count int) {
	if m != nil {
		m.candidates.Add(uint64(max(count, 0)))
	}
}

// Latency records handler latency in fixed <=10µs, <=50µs, <=250µs, <=1ms, >1ms buckets.
func (m *Metrics) Latency(elapsed time.Duration) {
	if m == nil {
		return
	}
	index := 4
	switch {
	case elapsed <= 10*time.Microsecond:
		index = 0
	case elapsed <= 50*time.Microsecond:
		index = 1
	case elapsed <= 250*time.Microsecond:
		index = 2
	case elapsed <= time.Millisecond:
		index = 3
	}
	m.latency[index].Add(1)
}

// Correction records whether a corrective packet was sent or skipped.
func (m *Metrics) Correction(sent bool) {
	if sent {
		m.correctionsSent.Add(1)
		return
	}
	m.correctionsSkip.Add(1)
}

// SubchunkDecoded records one successfully decoded subchunk payload.
func (m *Metrics) SubchunkDecoded() { m.subchunkDecode.Add(1) }

// SubchunkModified records one subchunk rewritten with deny blocks.
func (m *Metrics) SubchunkModified() { m.subchunkModify.Add(1) }

// SubchunkError records one subchunk decode or index failure.
func (m *Metrics) SubchunkError() { m.subchunkError.Add(1) }

type metricRecord struct {
	Type        string                 `json:"type"`
	Server      string                 `json:"server"`
	PeriodMS    int64                  `json:"period_ms"`
	ActionNames [metricActions]string  `json:"action_names"`
	ReasonNames [metricReasons]string  `json:"reason_names"`
	Refresh     [3]uint64              `json:"refresh"`
	Snapshot    snapshotMetric         `json:"snapshot"`
	Packets     uint64                 `json:"packets"`
	Seen        [metricActions]uint64  `json:"seen"`
	Forwarded   [metricActions]uint64  `json:"forwarded"`
	Denied      [metricActions]uint64  `json:"denied"`
	Reasons     [metricReasons]uint64  `json:"reasons"`
	Candidates  uint64                 `json:"candidates"`
	Latency     [latencyBuckets]uint64 `json:"latency_us"`
	Corrections [2]uint64              `json:"corrections"`
	Subchunk    [3]uint64              `json:"subchunk"`
}

type snapshotMetric struct {
	AgeMS      int64  `json:"age_ms"`
	Generation uint64 `json:"generation"`
	Claims     int    `json:"claims"`
	Cells      int    `json:"cells"`
}

// WriteDelta emits one compact JSON record and resets interval counters.
func (m *Metrics) WriteDelta(output io.Writer, period time.Duration, snapshot *Snapshot) {
	if m == nil {
		return
	}
	record := metricRecord{
		Type:        "claim_proxy_metrics",
		Server:      m.server,
		PeriodMS:    period.Milliseconds(),
		ActionNames: actionMetricNames,
		ReasonNames: reasonMetricNames,
		Packets:     m.packets.Swap(0),
		Refresh: [3]uint64{
			m.refreshAttempts.Swap(0),
			m.refreshSuccess.Swap(0),
			m.refreshFailure.Swap(0),
		},
		Candidates:  m.candidates.Swap(0),
		Corrections: [2]uint64{m.correctionsSent.Swap(0), m.correctionsSkip.Swap(0)},
		Subchunk:    [3]uint64{m.subchunkDecode.Swap(0), m.subchunkModify.Swap(0), m.subchunkError.Swap(0)},
		Snapshot:    snapshotMetric{AgeMS: -1},
	}
	for i := range metricActions {
		record.Seen[i] = m.seen[i].Swap(0)
		record.Forwarded[i] = m.forwarded[i].Swap(0)
		record.Denied[i] = m.denied[i].Swap(0)
	}
	for i := range metricReasons {
		record.Reasons[i] = m.reasons[i].Swap(0)
	}
	for i := range latencyBuckets {
		record.Latency[i] = m.latency[i].Swap(0)
	}
	if snapshot != nil {
		record.Snapshot = snapshotMetric{
			AgeMS:      snapshot.Age(time.Now()).Milliseconds(),
			Generation: snapshot.Generation,
			Claims:     snapshot.ClaimCount,
			Cells:      snapshot.CellCount,
		}
	}
	raw, err := json.Marshal(record)
	if err == nil {
		_, _ = fmt.Fprintln(output, string(raw))
	}
}
