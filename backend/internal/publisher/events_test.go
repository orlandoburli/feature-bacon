package publisher

import (
	"testing"
	"time"
)

func TestNewEvent_Fields(t *testing.T) {
	before := time.Now().Unix()
	event := NewEvent(EventFlagCreated, "tenant-1", map[string]string{"key": "dark-mode"})
	after := time.Now().Unix()

	if event.EventId == "" {
		t.Error("expected non-empty EventId")
	}
	if event.EventType != EventFlagCreated {
		t.Errorf("EventType = %q, want %q", event.EventType, EventFlagCreated)
	}
	if event.TenantId != "tenant-1" {
		t.Errorf("TenantId = %q, want %q", event.TenantId, "tenant-1")
	}
	if event.Timestamp < before || event.Timestamp > after {
		t.Errorf("Timestamp = %d, expected between %d and %d", event.Timestamp, before, after)
	}
	if event.PayloadJson == "" {
		t.Error("expected non-empty PayloadJson")
	}
}

func TestNewEvent_PayloadSerialization(t *testing.T) {
	event := NewEvent(EventExperimentCreated, "t1", struct {
		Name string `json:"name"`
	}{Name: "test"})

	want := `{"name":"test"}`
	if event.PayloadJson != want {
		t.Errorf("PayloadJson = %q, want %q", event.PayloadJson, want)
	}
}

func TestNewEvent_NilPayload(t *testing.T) {
	event := NewEvent(EventFlagDeleted, "t1", nil)
	if event.PayloadJson != "null" {
		t.Errorf("PayloadJson = %q, want %q", event.PayloadJson, "null")
	}
}

func TestNewEvent_UniqueIDs(t *testing.T) {
	e1 := NewEvent(EventFlagCreated, "t1", nil)
	e2 := NewEvent(EventFlagCreated, "t1", nil)
	if e1.EventId == e2.EventId {
		t.Error("expected unique EventIds across calls")
	}
}

func TestEventConstants(t *testing.T) {
	constants := []string{
		EventFlagCreated,
		EventFlagUpdated,
		EventFlagDeleted,
		EventExperimentCreated,
		EventExperimentUpdated,
		EventExperimentStarted,
		EventExperimentPaused,
		EventExperimentCompleted,
		EventExposure,
	}
	seen := make(map[string]bool, len(constants))
	for _, c := range constants {
		if c == "" {
			t.Error("found empty event constant")
		}
		if seen[c] {
			t.Errorf("duplicate event constant: %q", c)
		}
		seen[c] = true
	}
}
