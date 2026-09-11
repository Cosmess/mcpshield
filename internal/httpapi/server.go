package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type Server struct {
	logger         *slog.Logger
	requestTimeout time.Duration
	maxBodyBytes   int64
	ready          atomic.Bool
	requests       atomic.Uint64
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
	return server.instrument(mux)
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
