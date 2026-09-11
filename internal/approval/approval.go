package approval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
	"github.com/Cosmess/mcpshield/internal/risk"
)

type Status string

const (
	Pending  Status = "PENDING"
	Approved Status = "APPROVED"
	Denied   Status = "DENIED"
	Expired  Status = "EXPIRED"
	Consumed Status = "CONSUMED"
)

var (
	ErrNotFound        = errors.New("approval not found")
	ErrInvalidState    = errors.New("invalid approval state transition")
	ErrUnauthorized    = errors.New("reviewer is not authorized")
	ErrFingerprint     = errors.New("approval fingerprint mismatch")
	ErrExpired         = errors.New("approval expired")
	ErrAlreadyConsumed = errors.New("approval already consumed")
)

type Request struct {
	Principal  identity.Principal
	UpstreamID string
	Method     string
	Tool       string
	Arguments  map[string]any
	PolicyIDs  []string
	Decision   policy.Decision
	Risk       risk.Result
	ExpiresAt  time.Time
}

type Reviewer struct {
	Subject string
	Roles   []string
	Scopes  []string
}

type Record struct {
	ID             string
	Status         Status
	Subject        string
	TenantID       string
	UpstreamID     string
	Method         string
	Tool           string
	ArgumentsHash  string
	PolicyIDs      []string
	PolicyDecision policy.Decision
	RiskScore      int
	RiskSeverity   risk.Severity
	RiskSignals    []string
	Fingerprint    string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	Reviewer       string
	DecisionReason string
	ReviewedAt     time.Time
	ConsumedAt     time.Time
}

type Repository interface {
	Create(Record) error
	Get(string) (Record, error)
	Review(string, Status, Reviewer, string, time.Time) (Record, error)
	Consume(string, string, time.Time) (Record, error)
}

type MemoryRepository struct {
	mu      sync.Mutex
	records map[string]Record
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: make(map[string]Record)}
}

func (repository *MemoryRepository) Create(record Record) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.records[record.ID]; exists {
		return fmt.Errorf("approval %q already exists", record.ID)
	}
	repository.records[record.ID] = record
	return nil
}

func (repository *MemoryRepository) Get(id string) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, exists := repository.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	return record, nil
}

func (repository *MemoryRepository) Review(id string, status Status, reviewer Reviewer, reason string, now time.Time) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, exists := repository.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	if record.Status != Pending {
		return Record{}, ErrInvalidState
	}
	if !now.Before(record.ExpiresAt) {
		record.Status = Expired
		repository.records[id] = record
		return Record{}, ErrExpired
	}
	if !authorizedReviewer(record, reviewer) {
		return Record{}, ErrUnauthorized
	}
	if status != Approved && status != Denied {
		return Record{}, ErrInvalidState
	}
	record.Status = status
	record.Reviewer = reviewer.Subject
	record.DecisionReason = reason
	record.ReviewedAt = now
	repository.records[id] = record
	return record, nil
}

func (repository *MemoryRepository) Consume(id, fingerprint string, now time.Time) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, exists := repository.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	if record.Status == Consumed {
		return Record{}, ErrAlreadyConsumed
	}
	if !now.Before(record.ExpiresAt) {
		record.Status = Expired
		repository.records[id] = record
		return Record{}, ErrExpired
	}
	if record.Status != Approved {
		return Record{}, ErrInvalidState
	}
	if record.Fingerprint != fingerprint {
		return Record{}, ErrFingerprint
	}
	record.Status = Consumed
	record.ConsumedAt = now
	repository.records[id] = record
	return record, nil
}

type Service struct {
	repository Repository
	clock      func() time.Time
	created    atomic.Uint64
	approved   atomic.Uint64
	denied     atomic.Uint64
	expired    atomic.Uint64
	consumed   atomic.Uint64
}

func (service *Service) Repository() Repository { return service.repository }

func (service *Service) Review(id string, status Status, reviewer Reviewer, reason string) (Record, error) {
	record, err := service.repository.Review(id, status, reviewer, reason, service.clock())
	if err == nil {
		if status == Approved {
			service.approved.Add(1)
		} else if status == Denied {
			service.denied.Add(1)
		}
	}
	if errors.Is(err, ErrExpired) {
		service.expired.Add(1)
	}
	return record, err
}

func (service *Service) Consume(id, fingerprint string) (Record, error) {
	record, err := service.repository.Consume(id, fingerprint, service.clock())
	if err == nil {
		service.consumed.Add(1)
	}
	if errors.Is(err, ErrExpired) {
		service.expired.Add(1)
	}
	return record, err
}

func (service *Service) Metrics() string {
	return fmt.Sprintf("mcpshield_approval_created_total %d\nmcpshield_approval_approved_total %d\nmcpshield_approval_denied_total %d\nmcpshield_approval_expired_total %d\nmcpshield_approval_consumed_total %d\n", service.created.Load(), service.approved.Load(), service.denied.Load(), service.expired.Load(), service.consumed.Load())
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, clock: time.Now}
}

func (service *Service) CreatePending(request Request, id string) (Record, error) {
	if request.Decision != policy.RequireApproval {
		return Record{}, ErrInvalidState
	}
	if request.ExpiresAt.IsZero() || !request.ExpiresAt.After(service.clock()) {
		return Record{}, ErrExpired
	}
	argumentsHash, fingerprint, err := Fingerprint(request)
	if err != nil {
		return Record{}, err
	}
	record := Record{ID: id, Status: Pending, Subject: request.Principal.Subject, TenantID: request.Principal.TenantID, UpstreamID: request.UpstreamID, Method: request.Method, Tool: request.Tool, ArgumentsHash: argumentsHash, PolicyIDs: append([]string(nil), request.PolicyIDs...), PolicyDecision: request.Decision, RiskScore: request.Risk.Score, RiskSeverity: request.Risk.Severity, RiskSignals: riskIDs(request.Risk), Fingerprint: fingerprint, CreatedAt: service.clock(), ExpiresAt: request.ExpiresAt}
	if err := service.repository.Create(record); err != nil {
		return Record{}, err
	}
	service.created.Add(1)
	return record, nil
}

func Fingerprint(request Request) (string, string, error) {
	arguments, err := json.Marshal(request.Arguments)
	if err != nil {
		return "", "", fmt.Errorf("marshal approval arguments: %w", err)
	}
	argumentsHash := sha256.Sum256(arguments)
	binding := struct {
		Subject       string          `json:"subject"`
		TenantID      string          `json:"tenant_id"`
		UpstreamID    string          `json:"upstream_id"`
		Method        string          `json:"method"`
		Tool          string          `json:"tool"`
		ArgumentsHash string          `json:"arguments_hash"`
		PolicyIDs     []string        `json:"policy_ids"`
		Decision      policy.Decision `json:"decision"`
		RiskScore     int             `json:"risk_score"`
		RiskSeverity  risk.Severity   `json:"risk_severity"`
		RiskSignals   []string        `json:"risk_signals"`
	}{request.Principal.Subject, request.Principal.TenantID, request.UpstreamID, request.Method, request.Tool, hex.EncodeToString(argumentsHash[:]), append([]string(nil), request.PolicyIDs...), request.Decision, request.Risk.Score, request.Risk.Severity, riskIDs(request.Risk)}
	canonical, err := json.Marshal(binding)
	if err != nil {
		return "", "", fmt.Errorf("marshal approval binding: %w", err)
	}
	fingerprint := sha256.Sum256(canonical)
	return hex.EncodeToString(argumentsHash[:]), hex.EncodeToString(fingerprint[:]), nil
}

func authorizedReviewer(record Record, reviewer Reviewer) bool {
	if reviewer.Subject == "" || reviewer.Subject == record.Subject {
		return false
	}
	for _, scope := range reviewer.Scopes {
		if scope == "approvals:review" {
			return true
		}
	}
	for _, role := range reviewer.Roles {
		if role == "approver" || role == "security-admin" {
			return true
		}
	}
	return false
}

func riskIDs(result risk.Result) []string {
	ids := make([]string, 0, len(result.Signals))
	for _, signal := range result.Signals {
		ids = append(ids, signal.ID)
	}
	sort.Strings(ids)
	return ids
}
