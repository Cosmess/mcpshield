package mcpproxy

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/policy"
	"github.com/Cosmess/mcpshield/internal/upstream"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type greetInput struct {
	Name string `json:"name"`
}

func TestProxyRelaysDiscoveryAndToolCalls(t *testing.T) {
	upstreamServer := mcp.NewServer(&mcp.Implementation{Name: "mock-upstream", Version: "1.0.0"}, nil)
	mcp.AddTool(upstreamServer, &mcp.Tool{Name: "greet", Description: "Greet a person"}, func(_ context.Context, _ *mcp.CallToolRequest, input greetInput) (*mcp.CallToolResult, any, error) {
		return nil, map[string]any{"message": "hello " + input.Name}, nil
	})
	upstreamHTTP := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return upstreamServer }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	defer upstreamHTTP.Close()

	definition := upstream.Definition{ID: "mock", Endpoint: upstreamHTTP.URL, Enabled: true, Timeout: time.Second, ProtocolVersion: ProtocolVersion}
	registry, err := upstream.NewRegistry([]upstream.Definition{definition})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	sink := audit.NewMemorySink(16)
	proxy, err := New(context.Background(), registry, slog.Default(), sink)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer proxy.Close()

	proxyHTTP := httptest.NewServer(proxy.Handler())
	defer proxyHTTP.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: proxyHTTP.URL + "/mcp/mock", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "greet" {
		t.Fatalf("ListTools() = %#v, want greet", tools.Tools)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "greet", Arguments: map[string]any{"name": "MCPShield"}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError || len(result.StructuredContent.(map[string]any)) == 0 {
		t.Fatalf("CallTool() result = %#v, want successful structured result", result)
	}
	if events := sink.Events(); len(events) != 1 || events[0].Outcome != "success" || events[0].MCPMethod != "tools/call" {
		t.Fatalf("audit events = %#v, want successful tools/call event", events)
	}
	if event := sink.Events()[0]; event.RiskScore != 0 || event.RiskSeverity != "LOW" || len(event.RiskSignals) != 0 {
		t.Fatalf("risk audit metadata = %#v", event)
	}
	if metrics := proxy.RiskMetrics(); !strings.Contains(metrics, "mcpshield_risk_evaluations_total 1") {
		t.Fatalf("risk metrics = %q", metrics)
	}
	denyEngine, err := policy.New([]policy.Rule{{ID: "default-deny", Decision: policy.Deny}})
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetPolicy(denyEngine)
	denied, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "greet", Arguments: map[string]any{"name": "blocked"}})
	if err == nil || denied != nil {
		t.Fatalf("denied CallTool() = %#v, %v", denied, err)
	}
	if events := sink.Events(); len(events) != 2 || events[1].Outcome != "policy_denied" {
		t.Fatalf("audit events = %#v", events)
	}
}

func TestRiskSignalsFollowOperationClass(t *testing.T) {
	if got := riskSignals(policy.Admin); len(got) != 2 || got[0] != "privileged_operation" || got[1] != "write_operation" {
		t.Fatalf("admin signals = %#v", got)
	}
	if got := riskSignals(policy.Execution); len(got) != 1 || got[0] != "execution_operation" {
		t.Fatalf("execution signals = %#v", got)
	}
	if got := riskSignals(policy.Read); len(got) != 0 {
		t.Fatalf("read signals = %#v", got)
	}
}

func TestProxyBoundsUpstreamTimeout(t *testing.T) {
	upstreamServer := mcp.NewServer(&mcp.Implementation{Name: "slow-upstream", Version: "1.0.0"}, nil)
	mcp.AddTool(upstreamServer, &mcp.Tool{Name: "slow", Description: "Slow tool"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil, map[string]string{"status": "late"}, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})
	upstreamHTTP := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return upstreamServer }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	defer upstreamHTTP.Close()
	registry, err := upstream.NewRegistry([]upstream.Definition{{ID: "slow", Endpoint: upstreamHTTP.URL, Enabled: true, Timeout: 20 * time.Millisecond, ProtocolVersion: ProtocolVersion}})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	sink := audit.NewMemorySink(8)
	proxy, err := New(context.Background(), registry, slog.Default(), sink)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer proxy.Close()
	proxyHTTP := httptest.NewServer(proxy.Handler())
	defer proxyHTTP.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: proxyHTTP.URL + "/mcp/slow", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer session.Close()
	_, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "slow", Arguments: map[string]any{}})
	if err == nil {
		t.Fatal("CallTool() error = nil, want upstream timeout")
	}
	events := sink.Events()
	if len(events) == 0 || events[len(events)-1].Outcome != "upstream_error" {
		t.Fatalf("audit events = %#v, want upstream_error", events)
	}
}

func TestProxyRejectsUnknownUpstream(t *testing.T) {
	registry, err := upstream.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	proxy, err := New(context.Background(), registry, slog.Default(), audit.NewMemorySink(4))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer proxy.Close()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/attacker", nil)
	proxy.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown upstream status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
