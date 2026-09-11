package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/mcp/mock", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestMCPRoutePropagatesAuthenticatedPrincipal(t *testing.T) {
	server := New(slog.Default(), time.Second, 1024)
	server.SetAuthenticator(fakeAuthenticator{})
	server.SetMCPHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := identity.FromContext(request.Context())
		if !ok || principal.Subject != "user-1" {
			t.Errorf("principal = %#v, %t", principal, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/mcp/mock", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}
