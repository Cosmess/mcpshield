package audit

import (
	"testing"
	"time"
)

func TestMemorySinkIsBounded(t *testing.T) {
	sink := NewMemorySink(2)
	for index := 0; index < 3; index++ {
		sink.Record(Event{RequestID: string(rune('a' + index)), OccurredAt: time.Now()})
	}
	events := sink.Events()
	if len(events) != 2 || events[0].RequestID != "b" || events[1].RequestID != "c" {
		t.Fatalf("Events() = %#v, want last two events", events)
	}
}
