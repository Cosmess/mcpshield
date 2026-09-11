package audit

import (
	"sync"
	"time"
)

type Event struct {
	RequestID       string
	UpstreamID      string
	MCPMethod       string
	Outcome         string
	ProtocolVersion string
	Duration        time.Duration
	OccurredAt      time.Time
	RiskScore       int
	RiskSeverity    string
	RiskSignals     []string
	DLPAction       string
	DLPDetectors    []string
	DLPPaths        []string
}

type Sink interface {
	Record(Event)
}

type MemorySink struct {
	mu     sync.Mutex
	limit  int
	events []Event
}

func NewMemorySink(limit int) *MemorySink {
	if limit < 1 {
		limit = 1
	}
	return &MemorySink{limit: limit}
}

func (sink *MemorySink) Record(event Event) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.events = append(sink.events, event)
	if len(sink.events) > sink.limit {
		sink.events = sink.events[len(sink.events)-sink.limit:]
	}
}

func (sink *MemorySink) Events() []Event {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return append([]Event(nil), sink.events...)
}
