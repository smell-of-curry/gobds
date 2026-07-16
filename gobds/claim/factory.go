// Package claim provides claim management including fetching & storage.
package claim

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smell-of-curry/gobds/gobds/service"
)

const (
	// DefaultPollInterval controls claim refresh frequency.
	DefaultPollInterval = 15 * time.Second
	// DefaultMaxSnapshotAge is maximum age at which proxy denials are safe.
	DefaultMaxSnapshotAge = 45 * time.Second
)

// Factory ...
type Factory struct {
	service        *Service
	refreshMu      sync.Mutex
	snapshot       atomic.Pointer[Snapshot]
	generation     atomic.Uint64
	failureStatus  atomic.Uint32
	pollInterval   time.Duration
	maxSnapshotAge time.Duration
	metrics        *Metrics
	log            *slog.Logger
}

// NewFactory ...
func NewFactory(
	c service.Config,
	server string,
	pollInterval, maxSnapshotAge time.Duration,
	log *slog.Logger,
) *Factory {
	return &Factory{
		service:        NewService(c, log),
		pollInterval:   pollInterval,
		maxSnapshotAge: maxSnapshotAge,
		metrics:        NewMetrics(server),
		log:            log,
	}
}

// PollInterval returns configured refresh frequency.
func (f *Factory) PollInterval() time.Duration {
	return f.pollInterval
}

// Metrics returns this server's claim proxy metrics.
func (f *Factory) Metrics() *Metrics {
	return f.metrics
}

// Snapshot returns current immutable snapshot and fail-open status.
func (f *Factory) Snapshot(now time.Time) (*Snapshot, QueryStatus) {
	snapshot := f.snapshot.Load()
	if snapshot == nil {
		if status := QueryStatus(f.failureStatus.Load()); status != QueryReady {
			return nil, status
		}
		return nil, QueryMissing
	}
	if !snapshot.Supported() {
		return snapshot, QueryUnsupported
	}
	if snapshot.Age(now) > f.maxSnapshotAge {
		if status := QueryStatus(f.failureStatus.Load()); status != QueryReady {
			return snapshot, status
		}
		return snapshot, QueryStale
	}
	return snapshot, QueryReady
}

// Fetch ...
func (f *Factory) Fetch() error {
	if f.service == nil || !f.service.Enabled {
		return nil
	}
	f.refreshMu.Lock()
	defer f.refreshMu.Unlock()
	f.metrics.RefreshAttempt()

	result, err := f.service.FetchClaims()
	if err != nil {
		f.failureStatus.Store(uint32(QueryRefreshFailed))
		f.metrics.RefreshFailure()
		return err
	}
	now := time.Now()
	if result.NotModified {
		current := f.snapshot.Load()
		if current == nil {
			f.failureStatus.Store(uint32(QueryRefreshFailed))
			f.metrics.RefreshFailure()
			return fmt.Errorf("claims service returned 304 without a snapshot")
		}
		revalidated := *current
		revalidated.Generation = f.generation.Add(1)
		revalidated.FetchedAt = now
		f.snapshot.Store(&revalidated)
		f.failureStatus.Store(uint32(QueryReady))
		f.metrics.RefreshSuccess()
		return nil
	}
	next, err := BuildSnapshot(result.Claims, f.generation.Add(1), now)
	if err != nil {
		f.failureStatus.Store(uint32(QueryInvalid))
		f.metrics.RefreshFailure()
		return fmt.Errorf("build claim snapshot: %w", err)
	}
	f.snapshot.Store(next)
	f.failureStatus.Store(uint32(QueryReady))
	f.metrics.RefreshSuccess()
	return nil
}
