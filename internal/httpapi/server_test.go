package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer() *Server {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, 16)
}

func TestHealthEndpoints(t *testing.T) {
	server := newTestServer()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "alive") {
		t.Fatalf("live response = %d %q", recorder.Code, recorder.Body.String())
	}
	server.SetReady(false)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("not-ready status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newTestServer()
	handler := server.Handler()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health/live", nil))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "mcpshield_http_requests_total") {
		t.Fatalf("metrics response = %d %q", recorder.Code, recorder.Body.String())
	}
}
