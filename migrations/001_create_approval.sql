CREATE TABLE IF NOT EXISTS approval (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'APPROVED', 'DENIED', 'EXPIRED', 'CONSUMED')),
    subject TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    upstream_id TEXT NOT NULL,
    method TEXT NOT NULL,
    tool TEXT NOT NULL,
    arguments_hash TEXT NOT NULL,
    policy_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    policy_decision TEXT NOT NULL,
    risk_score INTEGER NOT NULL CHECK (risk_score BETWEEN 0 AND 100),
    risk_severity TEXT NOT NULL,
    risk_signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    fingerprint TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    reviewer TEXT,
    decision_reason TEXT,
    reviewed_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS approval_status_expires_idx ON approval (status, expires_at);
CREATE INDEX IF NOT EXISTS approval_tenant_idx ON approval (tenant_id);
