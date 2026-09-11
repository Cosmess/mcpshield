# M0 Bootstrap Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
docker compose config
```

## T1: Create the Go module and gateway entry point

**What**: Add the Go module, `cmd/gateway`, and a minimal process assembly boundary.

**Where**: `go.mod`, `cmd/gateway/`, `internal/runtime/`

**Depends on**: none

**Tests**: build and startup smoke test

**Done when**:

- [ ] Go 1.27 module builds without external services.
- [ ] Gateway starts with documented defaults and exits on signal.
- [ ] `go test ./...` passes.

## T2: Add typed configuration and safe startup failures

**What**: Parse environment configuration into a typed immutable config and validate address,
timeouts, body limit, and shutdown timeout.

**Where**: `internal/config/`, `cmd/gateway/`

**Depends on**: T1

**Tests**: unit tests for valid defaults, invalid duration, invalid limit, and secret-safe errors

**Done when**:

- [ ] Invalid typed values fail before the listener starts.
- [ ] Errors identify fields without including secret values.
- [ ] Configuration tests cover boundary values.

## T3: Add structured logging and request instrumentation

**What**: Configure `log/slog` and request completion instrumentation with stable event names,
duration, status, and request ID fields.

**Where**: `internal/observability/`, `internal/httpapi/`

**Depends on**: T1, T2

**Tests**: log handler tests and request instrumentation tests

**Done when**:

- [ ] Startup, request completion, configuration failure, and shutdown events are structured.
- [ ] Secret-like configuration fields are redacted.
- [ ] Request instrumentation does not log request bodies.

## T4: Add health and metrics endpoints

**What**: Expose liveness, readiness, and Prometheus metrics with stable response contracts.

**Where**: `internal/httpapi/`, `internal/observability/`

**Depends on**: T2, T3

**Tests**: handler tests for status codes, JSON shape, readiness failure, and metrics content

**Done when**:

- [ ] `/health/live` returns 200 JSON.
- [ ] `/health/ready` distinguishes ready from not-ready without changing liveness.
- [ ] `/metrics` returns Prometheus text and includes an MCPShield metric.

## T5: Add limits, timeout, and graceful shutdown proof

**What**: Apply request body limits and server timeouts, then prove bounded shutdown with an
in-flight request.

**Where**: `internal/httpapi/`, `internal/runtime/`, integration tests

**Depends on**: T3, T4

**Tests**: integration tests for oversized request, timeout, SIGTERM/SIGINT path, and resource closure

**Done when**:

- [ ] Oversized requests are rejected safely.
- [ ] In-flight work cannot extend beyond the shutdown deadline.
- [ ] The process closes owned resources and exits with the documented status.
- [ ] Race, vet, and full tests pass.

## T6: Document local execution and CI-ready gates

**What**: Add the developer commands, Docker Compose validation guidance, and any minimal local
configuration examples required to reproduce M0.

**Where**: `README.md`, `Makefile`, `docker-compose.yml`, `.github/`

**Depends on**: T1-T5

**Tests**: command validation and `docker compose config`

**Done when**:

- [ ] A clean checkout can follow the README to build and run the gateway.
- [ ] Required Go gates are documented and executable.
- [ ] Docker Compose validation does not require credentials.