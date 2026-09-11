# M1 Remote MCP Proxy Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
docker compose config
```

## T1: Add the official MCP SDK and protocol configuration

### What

Add `github.com/modelcontextprotocol/go-sdk/mcp` v1.7.0+ and define the MCP
`2026-07-28` baseline in configuration/constants.

### Where

`go.mod`, `internal/mcp/`

### Depends on

None.

### Tests

Dependency build and protocol baseline unit test.

### Done when

- [ ] SDK version supports MCP `2026-07-28`.
- [ ] The project builds with the SDK dependency.
- [ ] No custom JSON-RPC implementation is introduced.

## T2: Implement the immutable upstream registry

### What

Load and validate configured upstream IDs, endpoints, enabled state, timeout, and
protocol version; reject duplicate IDs and unsafe empty endpoints.

### Where

`internal/upstream/`, `internal/config/`

### Depends on

T1.

### Tests

Valid lookup, unknown ID, disabled ID, duplicate ID, malformed endpoint.

### Done when

- [ ] Request input can select only a known registry ID.
- [ ] Registry data is immutable after startup.
- [ ] Endpoint and timeout validation fails safely before serving.

## T3: Add the MCP proxy use case

### What

Connect to the selected upstream through `mcp.StreamableClientTransport`, relay
MCP discovery and tool calls, and close sessions on success and failure.

### Where

`internal/mcpproxy/`

### Depends on

T1, T2.

### Tests

SDK client connection, discovery relay, tool call relay, protocol error propagation.

### Done when

- [ ] The proxy uses the official SDK transport.
- [ ] Tool names and schemas are not changed.
- [ ] Upstream sessions close on all terminal paths.

## T4: Add the `/mcp/{upstreamID}` HTTP route

### What

Route MCP Streamable HTTP traffic through the existing M0 HTTP boundary, resolve
the trusted ID, and map gateway/upstream failures to bounded responses.

### Where

`internal/httpapi/`, `cmd/gateway/`

### Depends on

T2, T3.

### Tests

Unknown ID no outbound call, discovery, tool call, timeout, cancellation, and
arbitrary URL rejection

### Done when

- [ ] `/mcp/{upstreamID}` is the only proxy target surface.
- [ ] Caller-provided URLs are never used as destinations.
- [ ] Existing M0 size and timeout controls remain active.

## T5: Add audit and proxy metrics

### What

Add an audit sink interface, bounded in-memory implementation, and metrics for
proxy requests, duration, upstream failures, and outcomes.

### Where

`internal/audit/`, `internal/observability/`, `internal/mcpproxy/`

### Depends on

T3, T4.

### Tests

Event field assertions, secret exclusion, metric output, bounded sink behavior.

### Done when

- [ ] Accepted, rejected, and failed proxy attempts produce sanitized events.
- [ ] Authorization headers and raw arguments never appear in events.
- [ ] Metric labels are bounded and do not contain payloads.

## T6: Add the M1 mock-upstream integration suite and docs

### What

Build an official SDK-backed mock upstream and document the route, protocol
baseline, local demo, known SSRF/auth gaps, and verification evidence.

### Where

`internal/testsupport/`, `internal/mcpproxy/`, `README.md`,
`.specs/features/m1-mcp-proxy/verification.md`

### Depends on

T1-T5.

### Tests

Complete client -> MCPShield -> mock upstream scenario.

### Done when

- [ ] Discovery and tool call pass through the mock upstream.
- [ ] Unknown, timeout, and upstream failure cases are covered.
- [ ] Full project gates pass and M1 verification records evidence and residual risk.