package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCommandNameHandlesEmptyAndMalformedWithoutSlicing(t *testing.T) {
	for _, line := range []string{"", " ", "/"} {
		if _, empty, err := commandName(line, 16); err != nil || !empty {
			t.Fatalf("line %q: empty=%v err=%v", line, empty, err)
		}
	}
	if name, empty, err := commandName("/HeLp arg", 16); err != nil || empty || name != "help" {
		t.Fatalf("valid command: name=%q empty=%v err=%v", name, empty, err)
	}
	if name, empty, err := commandName("uncertain", 16); err != nil || empty || name != "" {
		t.Fatalf("uncertain command should pass through: name=%q empty=%v err=%v", name, empty, err)
	}
	_, _, err := commandName("/this-is-too-long", 4)
	var malformed malformedPacketError
	if !errors.As(err, &malformed) {
		t.Fatalf("oversized command returned %v", err)
	}
}

func TestTrafficDefaultsApplyToMissingSection(t *testing.T) {
	got := (TrafficConfig{}).WithDefaults()
	want := DefaultTrafficConfig()
	if got.Enforce {
		t.Fatal("traffic enforcement must default off")
	}
	if got.Chat != want.Chat || got.Commands != want.Commands ||
		got.MaxTextBytes != want.MaxTextBytes || got.MaxTotalStackActions != want.MaxTotalStackActions {
		t.Fatalf("missing config did not receive defaults: got=%+v want=%+v", got, want)
	}
}

func TestTokenBucketObserveOnlyAndEnforcement(t *testing.T) {
	config := DefaultTrafficConfig()
	config.Chat = RateLimit{Rate: 0.0001, Burst: 1}
	observe := newTrafficState(config, nil)
	if !observe.allow(trafficChat) || !observe.allow(trafficChat) {
		t.Fatal("observe-only bucket must preserve forwarding")
	}

	config.Enforce = true
	enforce := newTrafficState(config, nil)
	if !enforce.allow(trafficChat) || enforce.allow(trafficChat) {
		t.Fatal("enforced bucket must drop only excess")
	}

	var output bytes.Buffer
	enforce.session.WriteDelta(&output, "TEST", "xuid", time.Minute)
	var record trafficMetricRecord
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Observed[trafficChat] != 2 || record.Exceeded[trafficChat] != 1 ||
		record.Enforced[trafficChat] != 1 {
		t.Fatalf("unexpected metrics: %+v", record)
	}
}

func TestFormResponseBoundsAndStructure(t *testing.T) {
	config := DefaultTrafficConfig()
	config.MaxFormResponseBytes = 8
	config.MaxFormResponseValues = 1
	for _, response := range [][]byte{
		{},
		[]byte("{"),
		[]byte(`"12345678"`),
		[]byte(`[0,1]`),
	} {
		var malformed malformedPacketError
		if err := validateFormResponse(response, config); !errors.As(err, &malformed) {
			t.Fatalf("response %q returned %v", response, err)
		}
	}
	for _, response := range [][]byte{[]byte(`true`), []byte(`[0]`), []byte(`null`)} {
		if err := validateFormResponse(response, config); err != nil {
			t.Fatalf("valid response %q rejected: %v", response, err)
		}
	}
}
