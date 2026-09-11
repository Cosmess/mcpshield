package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is the authoritative approval repository. Consume uses a
// single conditional UPDATE so concurrent callers cannot replay an approval.
type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Create(record Record) error {
	policyIDs, _ := json.Marshal(record.PolicyIDs)
	riskSignals, _ := json.Marshal(record.RiskSignals)
	_, err := repository.pool.Exec(context.Background(), `INSERT INTO approval (id,status,subject,tenant_id,upstream_id,method,tool,arguments_hash,policy_ids,policy_decision,risk_score,risk_severity,risk_signals,fingerprint,created_at,expires_at,reviewer,decision_reason,reviewed_at,consumed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`, record.ID, record.Status, record.Subject, record.TenantID, record.UpstreamID, record.Method, record.Tool, record.ArgumentsHash, policyIDs, record.PolicyDecision, record.RiskScore, record.RiskSeverity, riskSignals, record.Fingerprint, record.CreatedAt, record.ExpiresAt, record.Reviewer, record.DecisionReason, record.ReviewedAt, record.ConsumedAt)
	if err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) Get(id string) (Record, error) {
	return repository.scan(repository.pool.QueryRow(context.Background(), `SELECT id,status,subject,tenant_id,upstream_id,method,tool,arguments_hash,policy_ids,policy_decision,risk_score,risk_severity,risk_signals,fingerprint,created_at,expires_at,reviewer,decision_reason,reviewed_at,consumed_at FROM approval WHERE id=$1`, id))
}

func (repository *PostgresRepository) Review(id string, status Status, reviewer Reviewer, reason string, now time.Time) (Record, error) {
	if status != Approved && status != Denied {
		return Record{}, ErrInvalidState
	}
	current, err := repository.Get(id)
	if err != nil {
		return Record{}, err
	}
	if current.Status != Pending {
		return Record{}, ErrInvalidState
	}
	if !now.Before(current.ExpiresAt) {
		_, _ = repository.pool.Exec(context.Background(), `UPDATE approval SET status='EXPIRED' WHERE id=$1 AND status='PENDING'`, id)
		return Record{}, ErrExpired
	}
	if !authorizedReviewer(current, reviewer) {
		return Record{}, ErrUnauthorized
	}
	return repository.scan(repository.pool.QueryRow(context.Background(), `UPDATE approval SET status=$2, reviewer=$3, decision_reason=$4, reviewed_at=$5 WHERE id=$1 AND status='PENDING' RETURNING id,status,subject,tenant_id,upstream_id,method,tool,arguments_hash,policy_ids,policy_decision,risk_score,risk_severity,risk_signals,fingerprint,created_at,expires_at,reviewer,decision_reason,reviewed_at,consumed_at`, id, status, reviewer.Subject, reason, now))
}

func (repository *PostgresRepository) Consume(id, fingerprint string, now time.Time) (Record, error) {
	return repository.scan(repository.pool.QueryRow(context.Background(), `UPDATE approval SET status='CONSUMED', consumed_at=$3 WHERE id=$1 AND fingerprint=$2 AND status='APPROVED' AND expires_at>$3 RETURNING id,status,subject,tenant_id,upstream_id,method,tool,arguments_hash,policy_ids,policy_decision,risk_score,risk_severity,risk_signals,fingerprint,created_at,expires_at,reviewer,decision_reason,reviewed_at,consumed_at`, id, fingerprint, now))
}

type rowScanner interface{ Scan(...any) error }

func (repository *PostgresRepository) scan(row rowScanner) (Record, error) {
	var record Record
	var policyIDs, riskSignals []byte
	var reviewer, reason *string
	var reviewedAt, consumedAt *time.Time
	err := row.Scan(&record.ID, &record.Status, &record.Subject, &record.TenantID, &record.UpstreamID, &record.Method, &record.Tool, &record.ArgumentsHash, &policyIDs, &record.PolicyDecision, &record.RiskScore, &record.RiskSeverity, &riskSignals, &record.Fingerprint, &record.CreatedAt, &record.ExpiresAt, &reviewer, &reason, &reviewedAt, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("scan approval: %w", err)
	}
	if err := json.Unmarshal(policyIDs, &record.PolicyIDs); err != nil {
		return Record{}, fmt.Errorf("decode policy IDs: %w", err)
	}
	if err := json.Unmarshal(riskSignals, &record.RiskSignals); err != nil {
		return Record{}, fmt.Errorf("decode risk signals: %w", err)
	}
	if reviewer != nil {
		record.Reviewer = *reviewer
	}
	if reason != nil {
		record.DecisionReason = *reason
	}
	if reviewedAt != nil {
		record.ReviewedAt = *reviewedAt
	}
	if consumedAt != nil {
		record.ConsumedAt = *consumedAt
	}
	return record, nil
}

var _ Repository = (*PostgresRepository)(nil)
