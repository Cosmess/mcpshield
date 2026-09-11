# M4 Deterministic Risk Engine Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Define risk domain types and signal registry

### What

Add bounded risk input, signal definitions, score result, severity, and immutable contribution registry.

### Where

`internal/risk/`

### Depends on

M3 operation classification and M2 principal contract.

### Tests

Signal registry validation and risk result shape tests.

### Done when

- [ ] Signal IDs and contributions are application-owned.
- [ ] Risk types contain no HTTP, AI, or provider-specific dependencies.

## T2: Implement deterministic scoring and severity

### What

Evaluate known signals once, cap scores at 100, map severity thresholds, and produce stable explanations.

### Where

`internal/risk/`

### Depends on

T1.

### Tests

All baseline signals, duplicate signals, score cap, repeatability, and severity boundaries.

### Done when

- [ ] Identical input returns identical output.
- [ ] Score is always 0-100.
- [ ] Severity boundaries are tested.

## T3: Add resource and input bounds

### What

Reject unknown signals in strict mode and bound signal count, IDs, labels, and metadata sizes.

### Where

`internal/risk/`

### Depends on

T1, T2.

### Tests

Oversized input, malformed signal, unknown signal, and bounded-work tests.

### Done when

- [ ] Evaluation cannot create unbounded work.
- [ ] Raw values are not retained in results.

## T4: Integrate risk with policy and proxy

### What

Build risk context before policy evaluation and expose it to policy without allowing risk to override DENY.

### Where

`internal/mcpproxy/`, `internal/policy/`

### Depends on

T2, T3.

### Tests

Critical-risk advisory context, deny preservation, and allowed low-risk call.

### Done when

- [ ] Risk is available to policy.
- [ ] Policy remains final authorization authority.

## T5: Add risk audit and metrics

### What

Record bounded risk score, severity, and signal IDs in audit and metrics without raw arguments or secrets.

### Where

`internal/audit/`, `internal/observability/`, `internal/risk/`

### Depends on

T2, T4.

### Tests

Sanitized audit, fixed-cardinality metrics, and failure behavior.

### Done when

- [ ] Risk results are observable.
- [ ] Labels use fixed IDs and severity only.

## T6: Document and verify M4

### What

Document contributions, thresholds, trusted signal sources, policy relationship, and verification evidence.

### Where

`README.md`, `docs/security/`, `.specs/features/m4-risk-engine/verification.md`

### Depends on

T1-T5.

### Tests

Full repository gates and risk abuse-case tests.

### Done when

- [ ] AI is explicitly excluded from authorization.
- [ ] All M4 criteria map to evidence.
- [ ] Residual risks are recorded before merge.
