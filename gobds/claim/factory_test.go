package claim

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smell-of-curry/gobds/gobds/service"
)

func TestFactoryFreshnessFailsOpenAfterMaximumAge(t *testing.T) {
	factory := NewFactory(service.Config{}, "test", time.Second, 3*time.Second, slog.Default())
	now := time.Unix(100, 0)
	snapshot, err := BuildSnapshot(map[string]PlayerClaim{
		"one": testSnapshotClaim("one", 0, 15),
	}, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	factory.snapshot.Store(snapshot)

	if _, status := factory.Snapshot(now.Add(3 * time.Second)); status != QueryReady {
		t.Fatalf("snapshot at max age should remain ready, got %v", status)
	}
	if _, status := factory.Snapshot(now.Add(3*time.Second + time.Nanosecond)); status != QueryStale {
		t.Fatalf("old snapshot should fail open as stale, got %v", status)
	}
	factory.failureStatus.Store(uint32(QueryRefreshFailed))
	if _, status := factory.Snapshot(now.Add(4 * time.Second)); status != QueryRefreshFailed {
		t.Fatalf("failed refresh beyond max age should be explicit, got %v", status)
	}
}

func TestFactoryKeepsPreviousSnapshotWhenReplacementInvalid(t *testing.T) {
	valid := true
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if valid {
			_, _ = writer.Write([]byte(`[{"_key":"one","data":{"claimId":"one","playerXUID":"owner","location":{"dimension":"minecraft:overworld","pos1":{"x":0,"z":0},"pos2":{"x":1,"z":1}}}}]`))
			return
		}
		_, _ = writer.Write([]byte(`[{"_key":"bad","data":{"claimId":"bad","playerXUID":"owner","location":{"dimension":"invalid","pos1":{"x":0,"z":0},"pos2":{"x":1,"z":1}}}}]`))
	}))
	defer server.Close()

	factory := NewFactory(
		service.Config{Enabled: true, URL: server.URL},
		"test",
		time.Second,
		time.Minute,
		slog.Default(),
	)
	if err := factory.Fetch(); err != nil {
		t.Fatal(err)
	}
	previous, status := factory.Snapshot(time.Now())
	if status != QueryReady {
		t.Fatalf("initial status = %v", status)
	}
	valid = false
	if err := factory.Fetch(); err == nil {
		t.Fatal("invalid replacement must fail")
	}
	current, status := factory.Snapshot(time.Now())
	if status != QueryReady || current != previous {
		t.Fatal("fresh previous snapshot must survive invalid replacement without partial swap")
	}
}

func TestFactoryMissingAndUnsupportedSnapshotsFailOpen(t *testing.T) {
	factory := NewFactory(service.Config{}, "test", time.Second, 3*time.Second, slog.Default())
	if _, status := factory.Snapshot(time.Now()); status != QueryMissing {
		t.Fatalf("missing snapshot status = %v", status)
	}
	factory.snapshot.Store(&Snapshot{PolicyVersion: PolicyVersion + 1, FetchedAt: time.Now()})
	if _, status := factory.Snapshot(time.Now()); status != QueryUnsupported {
		t.Fatalf("unsupported snapshot status = %v", status)
	}
}
