CREATE TABLE IF NOT EXISTS audit_event (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT NOT NULL DEFAULT '',
    upstream_id TEXT NOT NULL DEFAULT '',
    mcp_method TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL DEFAULT '',
    protocol_version TEXT NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    risk_score INTEGER NOT NULL DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    risk_severity TEXT NOT NULL DEFAULT '',
    risk_signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    dlp_action TEXT NOT NULL DEFAULT '',
    dlp_detectors JSONB NOT NULL DEFAULT '[]'::jsonb,
    dlp_paths JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS audit_event_occurred_at_idx ON audit_event (occurred_at);
CREATE INDEX IF NOT EXISTS audit_event_upstream_idx ON audit_event (upstream_id, occurred_at);
CREATE INDEX IF NOT EXISTS audit_event_outcome_idx ON audit_event (outcome, occurred_at);
