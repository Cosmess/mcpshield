package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Cosmess/mcpshield/internal/approval"
	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/identity"
)

type Authenticator interface {
	Authenticate(context.Context, *http.Request) (identity.Principal, error)
}

type Server struct {
	logger          *slog.Logger
	requestTimeout  time.Duration
	maxBodyBytes    int64
	ready           atomic.Bool
	requests        atomic.Uint64
	mcpHandler      http.Handler
	authenticator   Authenticator
	authAudit       audit.Sink
	authSuccess     atomic.Uint64
	authFailures    atomic.Uint64
	extraMetrics    func() string
	approvalService *approval.Service
}

func New(logger *slog.Logger, requestTimeout time.Duration, maxBodyBytes int64) *Server {
	server := &Server{logger: logger, requestTimeout: requestTimeout, maxBodyBytes: maxBodyBytes}
	server.ready.Store(true)
	return server
}

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", server.live)
	mux.HandleFunc("GET /health/ready", server.readyHandler)
	mux.HandleFunc("GET /metrics", server.metrics)
	mux.Handle("/mcp/", http.HandlerFunc(server.mcp))
	mux.HandleFunc("GET /api/v1/approvals/{id}", server.getApproval)
	mux.HandleFunc("POST /api/v1/approvals/{id}/approve", server.approve)
	mux.HandleFunc("POST /api/v1/approvals/{id}/deny", server.deny)
	return server.instrument(mux)
}

func (server *Server) SetMCPHandler(handler http.Handler) { server.mcpHandler = handler }

func (server *Server) SetAuthenticator(authenticator Authenticator) {
	server.authenticator = authenticator
}

func (server *Server) SetAuthAudit(sink audit.Sink) { server.authAudit = sink }

func (server *Server) SetExtraMetrics(metrics func() string) { server.extraMetrics = metrics }

func (server *Server) SetApprovalService(service *approval.Service) { server.approvalService = service }

func (server *Server) reviewer(request *http.Request) (approval.Reviewer, bool) {
	if server.authenticator == nil {
		return approval.Reviewer{}, false
	}
	principal, err := server.authenticator.Authenticate(request.Context(), request)
	if err != nil {
		return approval.Reviewer{}, false
	}
	return approval.Reviewer{Subject: principal.Subject, Roles: principal.Roles, Scopes: principal.Scopes}, true
}

func (server *Server) getApproval(writer http.ResponseWriter, request *http.Request) {
	if server.approvalService == nil {
		writeJSON(writer, http.StatusNotImplemented, map[string]string{"error": "approval_service_not_configured"})
		return
	}
	record, err := server.approvalService.Repository().Get(request.PathValue("id"))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "approval_not_found"})
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (server *Server) reviewApproval(writer http.ResponseWriter, request *http.Request, status approval.Status) {
	if server.approvalService == nil {
		writeJSON(writer, http.StatusNotImplemented, map[string]string{"error": "approval_service_not_configured"})
		return
	}
	reviewer, ok := server.reviewer(request)
	if !ok {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "authentication_required"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if request.Body != nil {
		_ = json.NewDecoder(request.Body).Decode(&body)
	}
	record, err := server.approvalService.Review(request.PathValue("id"), status, reviewer, body.Reason)
	if err != nil {
		writeJSON(writer, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (server *Server) approve(writer http.ResponseWriter, request *http.Request) {
	server.reviewApproval(writer, request, approval.Approved)
}
func (server *Server) deny(writer http.ResponseWriter, request *http.Request) {
	server.reviewApproval(writer, request, approval.Denied)
}

func (server *Server) mcp(writer http.ResponseWriter, request *http.Request) {
	if server.mcpHandler == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "mcp_upstream_not_configured"})
		return
	}
	if server.authenticator == nil {
		server.recordAuth(request, "authentication_required")
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "authentication_required"})
		return
	}
	principal, err := server.authenticator.Authenticate(request.Context(), request)
	if err != nil {
		server.authFailures.Add(1)
		server.recordAuth(request, "authentication_failed")
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "authentication_failed"})
		return
	}
	server.authSuccess.Add(1)
	server.recordAuth(request, "authenticated")
	request = request.WithContext(identity.WithPrincipal(request.Context(), principal))
	server.mcpHandler.ServeHTTP(writer, request)
}

func (server *Server) SetReady(ready bool) { server.ready.Store(ready) }

func (server *Server) live(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "alive"})
}

func (server *Server) readyHandler(writer http.ResponseWriter, _ *http.Request) {
	if !server.ready.Load() {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (server *Server) metrics(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("# HELP mcpshield_http_requests_total Total HTTP requests handled by MCPShield.\n"))
	_, _ = writer.Write([]byte("# TYPE mcpshield_http_requests_total counter\n"))
	_, _ = writer.Write([]byte("mcpshield_http_requests_total " + strconv.FormatUint(server.requests.Load(), 10) + "\n"))
	_, _ = writer.Write([]byte("mcpshield_auth_success_total " + strconv.FormatUint(server.authSuccess.Load(), 10) + "\n"))
	_, _ = writer.Write([]byte("mcpshield_auth_failures_total " + strconv.FormatUint(server.authFailures.Load(), 10) + "\n"))
	if server.extraMetrics != nil {
		_, _ = writer.Write([]byte(server.extraMetrics()))
	}
	if server.approvalService != nil {
		_, _ = writer.Write([]byte(server.approvalService.Metrics()))
	}
}

func (server *Server) recordAuth(request *http.Request, outcome string) {
	if server.authAudit != nil {
		server.authAudit.Record(audit.Event{RequestID: request.Header.Get("X-Request-ID"), MCPMethod: "authentication", Outcome: outcome, OccurredAt: time.Now()})
	}
}

func (server *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		server.requests.Add(1)
		if request.ContentLength > server.maxBodyBytes {
			writeJSON(writer, http.StatusRequestEntityTooLarge, map[string]string{"error": "request_body_too_large"})
			return
		}
		request = request.WithContext(context.WithValue(request.Context(), bodyLimitKey{}, server.maxBodyBytes))
		ctx, cancel := context.WithTimeout(request.Context(), server.requestTimeout)
		defer cancel()
		request = request.WithContext(ctx)
		if request.Body != nil {
			request.Body = http.MaxBytesReader(writer, request.Body, server.maxBodyBytes)
		}
		next.ServeHTTP(writer, request)
		server.logger.Info("http_request_completed", "method", request.Method, "path", request.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

type bodyLimitKey struct{}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
