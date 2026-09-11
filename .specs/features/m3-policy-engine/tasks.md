# M3 Native Policy Engine Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Define policy domain types and decisions

### What

Add typed decisions, operation classes, policy input, evaluation result, and principal-aware context.

### Where

`internal/policy/`

### Depends on

M2 principal contract.

### Tests

Decision model and operation classification tests.

### Done when

- [ ] No policy type imports HTTP, OPA, or AI packages.
- [ ] All five decisions and six operation classes are represented.

## T2: Implement policy loading and validation

### What

Load immutable startup rules and validate IDs, priorities, patterns, decisions, and conflicts.

### Where

`internal/policy/`, `internal/config/`

### Depends on

T1.

### Tests

Valid rules, duplicate IDs, invalid patterns, missing decisions, and contradictory precedence.

### Done when

- [ ] Invalid policy fails before serving.
- [ ] Compiled rule ordering is deterministic.

## T3: Implement deterministic evaluator

### What

Match principal, tenant, upstream, MCP method, tool pattern, operation class, and argument constraints; resolve precedence and deny-wins ties.

### Where

`internal/policy/`

### Depends on

T1, T2.

### Tests

Table-driven match, no-match deny, priority, tie, and argument predicate tests.

### Done when

- [ ] Same input always returns the same result.
- [ ] No-match returns DENY.
- [ ] Matched policy IDs and reason are returned.

## T4: Enforce decisions in MCP proxy

### What

Evaluate tool calls before upstream invocation and map DENY to a protocol-safe error while preserving future decision outcomes.

### Where

`internal/mcpproxy/`, `internal/httpapi/`

### Depends on

T3.

### Tests

Allowed call, denied call with outbound spy, approval decision, and audit results.

### Done when

- [ ] No denied tool reaches the upstream.
- [ ] Allowed tool behavior remains compatible with M1.
- [ ] Every decision is auditable.

## T5: Add policy simulation

### What

Expose an internal simulation use case for candidate policies without mutating active rules or invoking an upstream.

### Where

`internal/policy/`, `internal/httpapi/`

### Depends on

T3.

### Tests

Simulation result parity and no-side-effect tests.

### Done when

- [ ] Simulation and live evaluation share the evaluator.
- [ ] Simulation cannot call upstream sessions.

## T6: Document and verify M3

### What

Document default deny, precedence, policy examples, simulation limits, and evidence.

### Where

`README.md`, `docs/security/`, `.specs/features/m3-policy-engine/verification.md`

### Depends on

T1-T5.

### Tests

Full repository gates and policy abuse-case suite.

### Done when

- [ ] Default deny and AI non-authority are explicit.
- [ ] All M3 criteria map to evidence.
- [ ] Residual risks are recorded before merge.
