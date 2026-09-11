# M6 Human Approval Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Define approval domain and state machine

### What

Add statuses, sanitized request summary, reviewer model, transition errors, and approval record types.

### Where

`internal/approval/`

### Depends on

M3 decisions, M4 risk result, M5 DLP metadata.

### Tests

State transition table and invalid transition tests.

### Done when

- [ ] Only documented transitions are possible.
- [ ] Domain types contain no HTTP, AI, or database-specific authority.

## T2: Implement fingerprinting and sanitization

### What

Canonicalize bound fields, hash the request fingerprint, and sanitize arguments/secrets for approval records.

### Where

`internal/approval/`

### Depends on

T1.

### Tests

Stable fingerprint, argument mutation, policy/risk mutation, raw-secret absence, and DLP metadata tests.

### Done when

- [ ] Changing any bound field invalidates consumption.
- [ ] Raw tokens and matched secret values never persist.

## T3: Implement repository and service transitions

### What

Add repository interface, in-memory implementation, create/review/expire/atomic-consume use cases, and reviewer guards.

### Where

`internal/approval/`

### Depends on

T1, T2.

### Tests

Reviewer scope, self-approval, expiry, denial, one-time consume, and concurrent consume tests.

### Done when

- [ ] Exactly one concurrent consumer succeeds.
- [ ] Expired/denied records cannot be consumed.

## T4: Integrate REQUIRE_APPROVAL with proxy

### What

Pause policy approval decisions, expose approval status, and require successful bound consumption before upstream invocation.

### Where

`internal/mcpproxy/`, `internal/policy/`, `internal/approval/`

### Depends on

T3.

### Tests

Approval-required no-outbound, approved consume outbound, mutation replay rejection, and expiry.

### Done when

- [ ] REQUIRE_APPROVAL never auto-executes.
- [ ] Only consumed matching approvals reach upstream.

## T5: Add approval audit and metrics

### What

Record sanitized lifecycle transitions and bounded pending/approved/expired/consumed metrics.

### Where

`internal/audit/`, `internal/httpapi/`, `internal/approval/`

### Depends on

T3, T4.

### Tests

Audit redaction, transition counters, backlog gauge, and failure behavior.

### Done when

- [ ] Every transition is auditable.
- [ ] Metrics use fixed statuses and no payload labels.

## T6: Document and verify M6

### What

Document operator workflow, reviewer permissions, expiry, replay protection, and explicit AI boundaries.

### Where

`README.md`, `docs/security/`, `.specs/features/m6-approval/verification.md`

### Depends on

T1-T5.

### Tests

Full repository gates and approval abuse-case suite.

### Done when

- [ ] AI cannot approve or consume.
- [ ] All M6 criteria map to evidence.
- [ ] Residual operational risks are documented.
