package approval

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
)

func TestPostgresRepositoryApprovalLifecycle(t *testing.T) {
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

	repository := NewPostgresRepository(pool)
	now := time.Now().UTC()
	service := NewService(repository)
	record, err := service.CreatePending(Request{Principal: identity.Principal{Subject: "requester", TenantID: "tenant-a"}, UpstreamID: "mock", Method: "tools/call", Tool: "dangerous", Arguments: map[string]any{"id": "42"}, PolicyIDs: []string{"review"}, Decision: policy.RequireApproval, ExpiresAt: now.Add(time.Minute)}, "pg-approval")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.Get(record.ID)
	if err != nil || loaded.Fingerprint != record.Fingerprint {
		t.Fatalf("loaded = %#v, %v", loaded, err)
	}
	if _, err := service.Review(record.ID, Approved, Reviewer{Subject: "reviewer", Roles: []string{"approver"}}, "approved"); err != nil {
		t.Fatal(err)
	}
	consumed, err := service.Consume(record.ID, record.Fingerprint)
	if err != nil || consumed.Status != Consumed {
		t.Fatalf("consumed = %#v, %v", consumed, err)
	}
	if _, err := service.Consume(record.ID, record.Fingerprint); !errors.Is(err, ErrAlreadyConsumed) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestPostgresRepositoryAllowsOneConcurrentConsume(t *testing.T) {
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
	repository := NewPostgresRepository(pool)
	now := time.Now().UTC()
	service := NewService(repository)
	record, err := service.CreatePending(Request{Principal: identity.Principal{Subject: "requester", TenantID: "tenant-a"}, UpstreamID: "mock", Method: "tools/call", Tool: "dangerous", Arguments: map[string]any{"id": "43"}, Decision: policy.RequireApproval, ExpiresAt: now.Add(time.Minute)}, "pg-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Review(record.ID, Approved, Reviewer{Subject: "reviewer", Roles: []string{"approver"}}, "approved"); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	var successes int
	var mu sync.Mutex
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, consumeErr := service.Consume(record.ID, record.Fingerprint); consumeErr == nil {
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
