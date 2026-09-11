package mcpproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/dlp"
	"github.com/Cosmess/mcpshield/internal/identity"
	"github.com/Cosmess/mcpshield/internal/policy"
	"github.com/Cosmess/mcpshield/internal/risk"
	"github.com/Cosmess/mcpshield/internal/upstream"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ProtocolVersion = "2026-07-28"

type Proxy struct {
	logger          *slog.Logger
	audit           audit.Sink
	servers         map[string]*mcp.StreamableHTTPHandler
	sessions        map[string]*mcp.ClientSession
	mu              sync.RWMutex
	policy          *policy.Engine
	riskEvaluations atomic.Uint64
	riskHigh        atomic.Uint64
	riskCritical    atomic.Uint64
	dlpInspections  atomic.Uint64
	dlpBlocks       atomic.Uint64
	dlpRedactions   atomic.Uint64
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
		arguments := request.Params.Arguments
		dlpResult, dlpErr := dlp.InspectJSON(arguments)
		proxy.dlpInspections.Add(1)
		if dlpErr != nil {
			proxy.audit.Record(dlpEvent(ctx, definition.ID, "dlp_error", dlpResult, time.Since(started)))
			return &mcp.CallToolResult{IsError: true}, fmt.Errorf("inspect tool arguments: %w", dlpErr)
		}
		if dlpResult.Action == dlp.Block {
			proxy.dlpBlocks.Add(1)
			proxy.audit.Record(dlpEvent(ctx, definition.ID, "dlp_blocked", dlpResult, time.Since(started)))
			return &mcp.CallToolResult{IsError: true}, fmt.Errorf("tool arguments blocked by DLP")
		}
		if dlpResult.Action == dlp.Redact {
			proxy.dlpRedactions.Add(1)
			arguments = dlpResult.Payload
			proxy.audit.Record(dlpEvent(ctx, definition.ID, "dlp_redacted", dlpResult, time.Since(started)))
		}
		operation := policy.Classify("tools/call", toolName)
		signalIDs := riskSignals(operation)
		riskResult, riskErr := risk.Evaluate(risk.Input{Signals: signalIDs})
		if riskErr != nil {
			proxy.audit.Record(audit.Event{RequestID: requestIDFromContext(ctx), UpstreamID: definition.ID, MCPMethod: "tools/call", Outcome: "risk_error", Duration: time.Since(started), OccurredAt: time.Now()})
			return &mcp.CallToolResult{IsError: true}, fmt.Errorf("evaluate risk: %w", riskErr)
		}
		proxy.riskEvaluations.Add(1)
		if riskResult.Severity == risk.High {
			proxy.riskHigh.Add(1)
		}
		if riskResult.Severity == risk.Critical {
			proxy.riskCritical.Add(1)
		}
		if proxy.policy != nil {
			principal, _ := identity.FromContext(ctx)
			argumentsMap := map[string]any{}
			if len(arguments) > 0 {
				if err := json.Unmarshal(arguments, &argumentsMap); err != nil {
					return &mcp.CallToolResult{IsError: true}, fmt.Errorf("decode tool arguments for policy: %w", err)
				}
			}
			result := proxy.policy.Evaluate(policy.Input{Principal: principal, UpstreamID: definition.ID, Method: "tools/call", Tool: toolName, Operation: operation, Arguments: argumentsMap, Risk: &riskResult})
			if result.Decision == policy.Deny {
				proxy.audit.Record(riskEvent(ctx, definition.ID, "policy_denied", riskResult, time.Since(started)))
				return &mcp.CallToolResult{IsError: true}, fmt.Errorf("policy denied: %s", result.Reason)
			}
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: toolName, Arguments: arguments})
		outcome := "success"
		if err != nil {
			outcome = "upstream_error"
		}
		if ctx.Err() != nil {
			outcome = "client_canceled"
		}
		if err == nil && result != nil {
			responsePayload, marshalErr := json.Marshal(result)
			if marshalErr == nil {
				responseDLP, inspectErr := dlp.InspectJSON(responsePayload)
				if inspectErr == nil && responseDLP.Action == dlp.Block {
					proxy.dlpBlocks.Add(1)
					proxy.audit.Record(dlpEvent(ctx, definition.ID, "dlp_response_blocked", responseDLP, time.Since(started)))
					return &mcp.CallToolResult{IsError: true}, fmt.Errorf("tool response blocked by DLP")
				}
			}
		}
		proxy.audit.Record(riskEvent(ctx, definition.ID, outcome, riskResult, time.Since(started)))
		return result, err
	}
}

func dlpEvent(ctx context.Context, upstreamID, outcome string, result dlp.Result, duration time.Duration) audit.Event {
	detectors := make([]string, 0, len(result.Matches))
	paths := make([]string, 0, len(result.Matches))
	for _, match := range result.Matches {
		detectors = append(detectors, match.DetectorID)
		paths = append(paths, match.Path)
	}
	return audit.Event{RequestID: requestIDFromContext(ctx), UpstreamID: upstreamID, MCPMethod: "tools/call", Outcome: outcome, Duration: duration, OccurredAt: time.Now(), DLPAction: string(result.Action), DLPDetectors: detectors, DLPPaths: paths}
}

func (proxy *Proxy) RiskMetrics() string {
	return fmt.Sprintf("mcpshield_risk_evaluations_total %d\nmcpshield_risk_high_total %d\nmcpshield_risk_critical_total %d\nmcpshield_dlp_inspections_total %d\nmcpshield_dlp_blocks_total %d\nmcpshield_dlp_redactions_total %d\n", proxy.riskEvaluations.Load(), proxy.riskHigh.Load(), proxy.riskCritical.Load(), proxy.dlpInspections.Load(), proxy.dlpBlocks.Load(), proxy.dlpRedactions.Load())
}

func riskSignals(operation policy.OperationClass) []string {
	switch operation {
	case policy.Admin:
		return []string{"privileged_operation", "write_operation"}
	case policy.Execution:
		return []string{"execution_operation"}
	case policy.Write:
		return []string{"write_operation"}
	default:
		return nil
	}
}

func riskEvent(ctx context.Context, upstreamID, outcome string, result risk.Result, duration time.Duration) audit.Event {
	signals := make([]string, 0, len(result.Signals))
	for _, signal := range result.Signals {
		signals = append(signals, signal.ID)
	}
	return audit.Event{RequestID: requestIDFromContext(ctx), UpstreamID: upstreamID, MCPMethod: "tools/call", Outcome: outcome, ProtocolVersion: ProtocolVersion, Duration: duration, OccurredAt: time.Now(), RiskScore: result.Score, RiskSeverity: string(result.Severity), RiskSignals: signals}
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
