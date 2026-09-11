# M7 OPA/Rego Adapter Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Define internal evaluator contract

### What

Introduce an internal evaluator boundary and adapt the native engine to it without exposing OPA types.

### Where

`internal/policy/`

### Depends on

M3 policy engine.

### Tests

Contract tests for native evaluator results and decision parity.

### Done when

- [ ] Proxy depends on internal contract only.
- [ ] Native behavior remains unchanged.

## T2: Build sanitized OPA input and strict output types

### What

Create bounded input/output adapters, decision allowlist, reason/policy-version limits, and raw-secret rejection.

### Where

`internal/opa/`, `internal/policy/`

### Depends on

T1, M4 risk, M5 DLP.

### Tests

Sanitization, output mapping, malformed output, and version metadata.

### Done when

- [ ] OPA input contains no bearer tokens or raw secrets.
- [ ] Unknown decisions fail closed.

## T3: Implement OPA evaluator client

### What

Call OPA with context timeout, bounded response, cancellation, and explicit native fallback mode.

### Where

`internal/opa/`

### Depends on

T2.

### Tests

Allow, deny, approval, timeout, unavailable, malformed response, and fallback.

### Done when

- [ ] OPA failure behavior is configured explicitly.
- [ ] Every request is bounded and cancellable.

## T4: Integrate evaluator selection with policy/proxy

### What

Select native or OPA evaluator at startup and preserve deny precedence, approval, DLP, and risk boundaries.

### Where

`cmd/gateway/`, `internal/mcpproxy/`, `internal/policy/`

### Depends on

T3.

### Tests

Native-only, OPA mode, fallback, deny preservation, and no-upstream simulation.

### Done when

- [ ] OPA can be enabled explicitly.
- [ ] Native default-deny remains safe.

## T5: Add OPA observability

### What

Add fixed OPA metrics and sanitized evaluator/version audit metadata.

### Where

`internal/audit/`, `internal/httpapi/`, `internal/opa/`

### Depends on

T3, T4.

### Tests

Metric counters, timeout/error/fallback events, and no-input-leak assertions.

### Done when

- [ ] OPA metrics have bounded cardinality.
- [ ] Audit never contains raw policy input.

## T6: Document and verify M7

### What

Document enabling OPA, failure modes, fallback semantics, policy versioning, and native-vs-OPA boundaries.

### Where

`README.md`, `docs/security/`, `.specs/features/m7-opa-adapter/verification.md`

### Depends on

T1-T5.

### Tests

Full repository gates and OPA failure matrix.

### Done when

- [ ] OPA is optional and explicit.
- [ ] AI remains non-authoritative.
- [ ] All M7 criteria map to evidence.
