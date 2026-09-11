package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/identity"
)

type fakeAuthenticator struct{ err error }

func (fake fakeAuthenticator) Authenticate(context.Context, *http.Request) (identity.Principal, error) {
	if fake.err != nil {
		return identity.Principal{}, fake.err
	}
	return identity.Principal{Subject: "user-1"}, nil
}

func TestMCPRouteFailsClosedWithoutAuthentication(t *testing.T) {
	server := New(slog.Default(), time.Second, 1024)
	server.SetMCPHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) }))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/mock", nil)
	request.Header.Set("X-Subject", "spoofed-user")
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestMCPRoutePropagatesAuthenticatedPrincipal(t *testing.T) {
	server := New(slog.Default(), time.Second, 1024)
	sink := audit.NewMemorySink(8)
	server.SetAuthAudit(sink)
	server.SetAuthenticator(fakeAuthenticator{})
	server.SetMCPHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := identity.FromContext(request.Context())
		if !ok || principal.Subject != "user-1" {
			t.Errorf("principal = %#v, %t", principal, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/mock", nil)
	request.Header.Set("X-Subject", "spoofed-user")
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if events := sink.Events(); len(events) != 1 || events[0].Outcome != "authenticated" || events[0].MCPMethod != "authentication" {
		t.Fatalf("auth audit events = %#v", events)
	}
	metrics := httptest.NewRecorder()
	server.Handler().ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metrics.Body.String(), "mcpshield_auth_success_total 1") {
		t.Fatalf("metrics = %q", metrics.Body.String())
	}
}
