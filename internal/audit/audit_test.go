package audit

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
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

func TestPostgresSinkRecordsAuditEvent(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("mcpshield"), postgres.WithUsername("mcpshield"), postgres.WithPassword("mcpshield"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Skipf("PostgreSQL integration unavailable: %v", err)
	}
	defer container.Terminate(ctx)
	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := ApplyMigration(ctx, pool); err != nil {
		t.Fatal(err)
	}

	sink := NewPostgresSink(pool)
	occurredAt := time.Now().UTC()
	sink.Record(Event{RequestID: "req-1", UpstreamID: "orders", MCPMethod: "tools/call", Outcome: "allowed", ProtocolVersion: "2026-07-28", Duration: 25 * time.Millisecond, OccurredAt: occurredAt, RiskScore: 42, RiskSeverity: "medium", RiskSignals: []string{"tool_write"}, DLPAction: "redact", DLPDetectors: []string{"email"}, DLPPaths: []string{"arguments.email"}})

	var requestID, upstreamID, outcome string
	var durationMS int64
	var riskSignals, dlpDetectors, dlpPaths []string
	if err := pool.QueryRow(ctx, `SELECT request_id, upstream_id, outcome, duration_ms, risk_signals, dlp_detectors, dlp_paths FROM audit_event WHERE request_id=$1`, "req-1").Scan(&requestID, &upstreamID, &outcome, &durationMS, &riskSignals, &dlpDetectors, &dlpPaths); err != nil {
		t.Fatal(err)
	}
	if requestID != "req-1" || upstreamID != "orders" || outcome != "allowed" || durationMS != 25 {
		t.Fatalf("stored audit event = %q %q %q %d", requestID, upstreamID, outcome, durationMS)
	}
	if len(riskSignals) != 1 || riskSignals[0] != "tool_write" || len(dlpDetectors) != 1 || dlpDetectors[0] != "email" || len(dlpPaths) != 1 || dlpPaths[0] != "arguments.email" {
		t.Fatalf("stored JSON fields = %#v %#v %#v", riskSignals, dlpDetectors, dlpPaths)
	}
}
