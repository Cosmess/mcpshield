# M1 Remote MCP Proxy Verification

## Verdict

`PASS WITH RESIDUAL RISKS`

The M1 vertical slice accepts a trusted upstream ID, connects to the upstream through the
official Go SDK Streamable HTTP transport, mirrors discovered tools through an SDK-backed
stateless handler, forwards tool calls, records sanitized outcomes, and rejects unknown
upstreams. Authentication, upstream credentials, and production SSRF controls remain
deliberately outside this slice.

## Acceptance criteria evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| M1-AC1 | `internal/upstream/registry_test.go`; unknown-ID proxy test | PASS |
| M1-AC2 | Registry exposes only configured IDs; arbitrary URL lookup test | PASS |
| M1-AC3 | `internal/mcpproxy/proxy.go` uses `mcp.StreamableClientTransport`; SDK integration test | PASS |
| M1-AC4 | `TestProxyRelaysDiscoveryAndToolCalls` asserts the `greet` schema/name | PASS |
| M1-AC5 | Same test calls `greet` through MCPShield and asserts structured output | PASS |
| M1-AC6 | SDK owns MCP `2026-07-28` negotiation and metadata; no custom JSON-RPC path | PASS |
| M1-AC7 | `TestProxyBoundsUpstreamTimeout` proves the configured upstream timeout | PASS |
| M1-AC8 | Memory audit sink records success, rejection, and timeout outcomes without payload fields | PASS |
| M1-AC9 | SDK-backed `httptest` upstream covers discovery, tool call, unknown ID, and timeout | PASS |

## Executed gates

```text
gofmt -w cmd internal       PASS
go mod tidy                 PASS
go test ./...               PASS
go test -race ./...         PASS
go vet ./...                PASS
go build ./cmd/...          PASS
docker compose config       PASS
git diff --check            PASS
```

## Residual risks and deferred work

- M1 does not authenticate callers or upstreams; M2 owns identity and credential strategy.
- Configured endpoints are trusted inputs but do not yet have complete SSRF enforcement.
- The audit sink is in-memory; PostgreSQL authority and outbox durability belong to a later slice.
- Tool schemas are mirrored at startup; schema drift detection is deferred.