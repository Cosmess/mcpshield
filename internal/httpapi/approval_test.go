package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cosmess/mcpshield/internal/approval"
	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
)

type approvalAuthenticator struct{}

func (approvalAuthenticator) Authenticate(context.Context, *http.Request) (identity.Principal, error) {
	return identity.Principal{Subject: "reviewer", Roles: []string{"approver"}}, nil
}

func TestApprovalRoutesReviewAndExposeMetrics(t *testing.T) {
	service := approval.NewService(approval.NewMemoryRepository())
	request := approval.Request{
		Principal:  identity.Principal{Subject: "requester", TenantID: "tenant-a"},
		UpstreamID: "mock", Method: "tools/call", Tool: "github.merge_pr",
		Arguments: map[string]any{"id": "42"}, PolicyIDs: []string{"review"},
		Decision: policy.RequireApproval, ExpiresAt: time.Now().Add(time.Minute),
	}
	record, err := service.CreatePending(request, "approval-http")
	if err != nil {
		t.Fatal(err)
	}

	server := New(slog.Default(), time.Second, 1024)
	server.SetApprovalService(service)
	server.SetAuthenticator(approvalAuthenticator{})
	handler := server.Handler()

	approve := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+record.ID+"/approve", strings.NewReader(`{"reason":"incident"}`))
	approve.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, approve)
	if response.Code != http.StatusOK {
		t.Fatalf("approve status = %d, body = %s", response.Code, response.Body.String())
	}

	metrics := httptest.NewRecorder()
	handler.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metrics.Body.String(), "mcpshield_approval_created_total 1") || !strings.Contains(metrics.Body.String(), "mcpshield_approval_approved_total 1") {
		t.Fatalf("metrics = %s", metrics.Body.String())
	}
}
