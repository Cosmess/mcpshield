package mcpproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
	"github.com/Cosmess/mcpshield/internal/upstream"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ProtocolVersion = "2026-07-28"

type Proxy struct {
	logger   *slog.Logger
	audit    audit.Sink
	servers  map[string]*mcp.StreamableHTTPHandler
	sessions map[string]*mcp.ClientSession
	mu       sync.RWMutex
	policy   *policy.Engine
}

func (proxy *Proxy) SetPolicy(engine *policy.Engine) { proxy.policy = engine }

func New(ctx context.Context, registry *upstream.Registry, logger *slog.Logger, sink audit.Sink) (*Proxy, error) {
	proxy := &Proxy{logger: logger, audit: sink, servers: make(map[string]*mcp.StreamableHTTPHandler), sessions: make(map[string]*mcp.ClientSession)}
	for _, definition := range registry.Definitions() {
		if err := proxy.addUpstream(ctx, definition); err != nil {
			proxy.Close()
			return nil, fmt.Errorf("connect upstream %q: %w", definition.ID, err)
		}
	}
	return proxy, nil
}

func (proxy *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = request.WithContext(context.WithValue(request.Context(), requestIDKey{}, requestID(request)))
		prefix := "/mcp/"
		if !strings.HasPrefix(request.URL.Path, prefix) {
			http.NotFound(writer, request)
			return
		}
		id := strings.Trim(strings.TrimPrefix(request.URL.Path, prefix), "/")
		proxy.mu.RLock()
		handler, found := proxy.servers[id]
		proxy.mu.RUnlock()
		if !found || id == "" {
			proxy.audit.Record(audit.Event{
				RequestID:  requestID(request),
				UpstreamID: id,
				MCPMethod:  "unknown",
				Outcome:    "rejected",
				OccurredAt: time.Now(),
			})
			writeError(writer, http.StatusNotFound, "unknown_upstream")
			return
		}
		handler.ServeHTTP(writer, request)
	})
}

func (proxy *Proxy) Close() {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	for id, session := range proxy.sessions {
		_ = session.Close()
		delete(proxy.sessions, id)
	}
}

func (proxy *Proxy) addUpstream(ctx context.Context, definition upstream.Definition) error {
	client := mcp.NewClient(&mcp.Implementation{Name: "mcpshield", Version: "0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             definition.Endpoint,
		HTTPClient:           &http.Client{Timeout: definition.Timeout},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return fmt.Errorf("list tools: %w", err)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "mcpshield", Version: "0.1.0"}, nil)
	for _, remoteTool := range tools.Tools {
		server.AddTool(remoteTool, proxy.toolHandler(definition, session, remoteTool.Name))
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		MaxRequestBodyBytes:          1 << 20,
		PropagateRequestCancellation: true,
		Logger:                       proxy.logger,
	})
	proxy.mu.Lock()
	proxy.sessions[definition.ID] = session
	proxy.servers[definition.ID] = handler
	proxy.mu.Unlock()
	return nil
}

func (proxy *Proxy) toolHandler(definition upstream.Definition, session *mcp.ClientSession, toolName string) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		started := time.Now()
		if proxy.policy != nil {
			principal, _ := identity.FromContext(ctx)
			arguments := map[string]any{}
			if len(request.Params.Arguments) > 0 {
				if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
					return &mcp.CallToolResult{IsError: true}, fmt.Errorf("decode tool arguments for policy: %w", err)
				}
			}
			result := proxy.policy.Evaluate(policy.Input{Principal: principal, UpstreamID: definition.ID, Method: "tools/call", Tool: toolName, Operation: policy.Classify("tools/call", toolName), Arguments: arguments})
			if result.Decision == policy.Deny {
				proxy.audit.Record(audit.Event{RequestID: requestIDFromContext(ctx), UpstreamID: definition.ID, MCPMethod: "tools/call", Outcome: "policy_denied", Duration: time.Since(started), OccurredAt: time.Now()})
				return &mcp.CallToolResult{IsError: true}, fmt.Errorf("policy denied: %s", result.Reason)
			}
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: toolName, Arguments: request.Params.Arguments})
		outcome := "success"
		if err != nil {
			outcome = "upstream_error"
		}
		if ctx.Err() != nil {
			outcome = "client_canceled"
		}
		proxy.audit.Record(audit.Event{RequestID: requestIDFromContext(ctx), UpstreamID: definition.ID, MCPMethod: "tools/call", Outcome: outcome, ProtocolVersion: ProtocolVersion, Duration: time.Since(started), OccurredAt: time.Now()})
		return result, err
	}
}

type requestIDKey struct{}

func requestID(request *http.Request) string {
	return request.Header.Get("X-Request-ID")
}

func requestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}
	return ""
}

func writeError(writer http.ResponseWriter, status int, code string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": code})
}
