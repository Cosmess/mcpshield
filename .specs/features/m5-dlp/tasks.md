# M5 DLP and Secret Detection Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Define DLP domain and detector registry

### What

Add bounded inspection input, actions, matches, results, and immutable application-owned detector definitions.

### Where

`internal/dlp/`

### Depends on

M4 audit and proxy boundaries.

### Tests:

Domain validation, action matrix, detector registry, and match redaction invariants.

### Done when:

- [ ] Matches contain no raw values.
- [ ] Detector IDs and actions are validated.

## T2: Implement baseline secret detectors

### What

Implement AWS key, JWT-like, GitHub token, bearer, private-key header, API-key field, and password-field detectors.

### Where

`internal/dlp/`

### Depends on

T1.

### Tests:

Synthetic positive cases, safe negative cases, nested paths, and arrays.

### Done when:

- [ ] Every baseline detector has deterministic tests.
- [ ] Raw matches never appear in errors or results.

## T3: Implement bounded inspection and redaction

### What

Inspect bounded JSON payloads, enforce depth/match/path limits, and produce valid structurally redacted JSON.

### Where

`internal/dlp/`

### Depends on

T1, T2.

### Tests:

Malformed JSON, oversized payload, depth, match count, redaction round trip, and no-secret-output tests.

### Done when:

- [ ] Invalid/oversized input fails safely.
- [ ] Redaction preserves valid JSON structure.

## T4: Integrate request and response DLP with proxy

### What

Inspect tool arguments before upstream dispatch and inspect tool results before client delivery; block or redact according to action.

### Where

`internal/mcpproxy/`, `internal/dlp/`

### Depends on

T2, T3.

### Tests:

BLOCK no-outbound, REDACT request, REDACT response, and AUDIT-only flow.

### Done when:

- [ ] BLOCK never invokes upstream.
- [ ] Redacted payload reaches the correct boundary.
- [ ] Original payload is not logged.

## T5: Add DLP audit and metrics

### What

Record fixed detector/action/path counters and sanitized audit events without payload values.

### Where

`internal/audit/`, `internal/httpapi/`, `internal/dlp/`

### Depends on

T3, T4.

### Tests:

Event redaction, metric cardinality, match count, and failure behavior.

### Done when:

- [ ] Audit contains detector/path/action metadata only.
- [ ] Metrics use fixed detector/action labels.

## T6: Document and verify M5

### What

Document detector limitations, configuration, privacy guarantees, false-positive/false-negative risk, and evidence.

### Where

`README.md`, `docs/security/`, `.specs/features/m5-dlp/verification.md`

### Depends on

T1-T5.

### Tests

Full repository gates and DLP abuse-case suite.

### Done when

- [ ] README states deterministic matching is not complete enterprise DLP.
- [ ] All M5 criteria map to evidence.
- [ ] Residual privacy risks are recorded before merge.
