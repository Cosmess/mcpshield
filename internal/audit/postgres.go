package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSink struct {
	pool *pgxpool.Pool
}

func NewPostgresSink(pool *pgxpool.Pool) *PostgresSink {
	return &PostgresSink{pool: pool}
}

func (sink *PostgresSink) Record(event Event) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	riskSignals, err := json.Marshal(event.RiskSignals)
	if err != nil {
		return
	}
	dlpDetectors, err := json.Marshal(event.DLPDetectors)
	if err != nil {
		return
	}
	dlpPaths, err := json.Marshal(event.DLPPaths)
	if err != nil {
		return
	}
	_, _ = sink.pool.Exec(context.Background(), `INSERT INTO audit_event (request_id,upstream_id,mcp_method,outcome,protocol_version,duration_ms,occurred_at,risk_score,risk_severity,risk_signals,dlp_action,dlp_detectors,dlp_paths) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, event.RequestID, event.UpstreamID, event.MCPMethod, event.Outcome, event.ProtocolVersion, event.Duration.Milliseconds(), event.OccurredAt, event.RiskScore, event.RiskSeverity, riskSignals, event.DLPAction, dlpDetectors, dlpPaths)
}

var _ Sink = (*PostgresSink)(nil)
