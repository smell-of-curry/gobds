package claim

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestMetricsEmitPerServerDeltaJSON(t *testing.T) {
	var output bytes.Buffer
	metrics := NewMetrics("GOLD")
	metrics.RefreshAttempt()
	metrics.RefreshSuccess()
	metrics.Packet()
	metrics.Action(1, false)
	metrics.Reason(QueryOverlap)
	metrics.Candidates(2)
	metrics.Latency(20 * time.Microsecond)
	metrics.Correction(true)
	metrics.SubchunkDecoded()
	metrics.SubchunkModified()

	metrics.WriteDelta(&output, time.Minute, &Snapshot{
		PolicyVersion: PolicyVersion,
		SchemaVersion: SchemaVersion,
		Generation:    4,
		FetchedAt:     time.Now(),
		ClaimCount:    3,
		CellCount:     5,
	})
	var record metricRecord
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Server != "GOLD" || record.Packets != 1 || record.Refresh != [3]uint64{1, 1, 0} ||
		record.Seen[1] != 1 || record.Denied[1] != 1 || record.Reasons[QueryOverlap] != 1 ||
		record.Snapshot.Generation != 4 {
		t.Fatalf("unexpected metric record: %+v", record)
	}

	output.Reset()
	metrics.WriteDelta(&output, time.Minute, nil)
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Refresh != [3]uint64{} || record.Seen[1] != 0 {
		t.Fatal("delta counters did not reset")
	}
}
