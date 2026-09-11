package approval

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
)

func testRequest(expiresAt time.Time) Request {
	return Request{Principal: identity.Principal{Subject: "requester", TenantID: "tenant-a"}, UpstreamID: "mock", Method: "tools/call", Tool: "github.merge_pr", Arguments: map[string]any{"id": "42"}, PolicyIDs: []string{"prod-write"}, Decision: policy.RequireApproval, ExpiresAt: expiresAt}
}

func TestApprovalLifecycleAndFingerprintBinding(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	service := NewService(repository)
	record, err := service.CreatePending(testRequest(now.Add(time.Minute)), "approval-1")
	if err != nil || record.Status != Pending || record.Fingerprint == "" {
		t.Fatalf("CreatePending() = %#v, %v", record, err)
	}
	if _, err = repository.Review(record.ID, Approved, Reviewer{Subject: "reviewer", Scopes: []string{"approvals:review"}}, "approved for incident", now); err != nil {
		t.Fatal(err)
	}
	request := testRequest(now.Add(time.Minute))
	_, fingerprint, _ := Fingerprint(request)
	if _, err = repository.Consume(record.ID, fingerprint+"changed", now); !errors.Is(err, ErrFingerprint) {
		t.Fatalf("Consume() error = %v", err)
	}
	consumed, err := repository.Consume(record.ID, record.Fingerprint, now)
	if err != nil || consumed.Status != Consumed {
		t.Fatalf("Consume() = %#v, %v", consumed, err)
	}
	if _, err = repository.Consume(record.ID, record.Fingerprint, now); !errors.Is(err, ErrAlreadyConsumed) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestApprovalRejectsSelfReviewAndExpires(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	service := NewService(repository)
	record, err := service.CreatePending(testRequest(now.Add(time.Minute)), "approval-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Review(record.ID, Approved, Reviewer{Subject: "requester", Scopes: []string{"approvals:review"}}, "self", now); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("self-review error = %v", err)
	}
	expired, err := service.CreatePending(testRequest(now.Add(-time.Second)), "approval-3")
	if !errors.Is(err, ErrExpired) || expired.Status != "" {
		t.Fatalf("expired create = %#v, %v", expired, err)
	}
}

func TestApprovalConsumeIsAtomic(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	service := NewService(repository)
	record, err := service.CreatePending(testRequest(now.Add(time.Minute)), "approval-4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Review(record.ID, Approved, Reviewer{Subject: "reviewer", Roles: []string{"approver"}}, "approved", now); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	var successes int
	var mu sync.Mutex
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, consumeErr := repository.Consume(record.ID, record.Fingerprint, now); consumeErr == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	group.Wait()
	if successes != 1 {
		t.Fatalf("successful consumes = %d, want 1", successes)
	}
}

func TestApprovalTransitionHookReceivesLifecycleEvents(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(NewMemoryRepository())
	var statuses []Status
	service.SetTransitionHook(func(status Status, _ Record) { statuses = append(statuses, status) })
	record, err := service.CreatePending(testRequest(now.Add(time.Minute)), "approval-hook")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Review(record.ID, Approved, Reviewer{Subject: "reviewer", Roles: []string{"approver"}}, "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Consume(record.ID, record.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 3 || statuses[0] != Pending || statuses[1] != Approved || statuses[2] != Consumed {
		t.Fatalf("statuses = %#v", statuses)
	}
}
