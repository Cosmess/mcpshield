# M4 Deterministic Risk Engine Verification

## Verdict

`PASS WITH RESIDUAL RISKS`

Integration checkpoint: PR #23 `feat: integrate M4 risk with policy and proxy`,
merge commit `a1d34a6`.

M4 is implemented through policy/proxy integration and sanitized audit/metrics: immutable
application-owned signal definitions, bounded input, deterministic additive scoring,
deduplication, 0-100 cap, severity thresholds, stable signal ordering, and sanitized risk
metadata before policy enforcement.

## Acceptance criteria evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| M4-AC1 | Repeatability test for identical signal input | PASS |
| M4-AC2 | All baseline signals cap score at 100 | PASS |
| M4-AC3 | Eight baseline signal definitions are registered | PASS |
| M4-AC4 | Boundary tests cover 29/30/59/60/79/80/100 | PASS |
| M4-AC5 | Stable signal IDs, contributions, severity, and explanation are returned | PASS |
| M4-AC6 | Duplicate signals deduplicate and unknown IDs fail | PASS |
| M4-AC7 | `policy.Input.Risk` receives the result before policy enforcement; policy remains authoritative | PASS |
| M4-AC8 | Audit carries score/severity/signal IDs and `/metrics` exposes fixed risk counters | PASS |
| M4-AC9 | Signal count and ID length are bounded | PASS |

## Executed gates

```text
gofmt -w internal/risk  PASS
go test ./internal/risk  PASS
go test -race ./internal/risk  PASS
go vet ./internal/risk  PASS
```

## Residual risks and deferred work

- Production, external URL, unseen-tool, frequency, and cross-tenant signals currently
	require trusted context producers; the proxy baseline derives operation signals only.
- Risk does not authorize requests and does not yet drive approval/redaction/limits executors.
- Risk counter persistence and historical analysis are deferred to later observability work.

## Executed gates

```text
gofmt -w cmd internal       PASS
go test ./...               PASS
go test -race ./...         PASS
go vet ./...                PASS
go build ./cmd/...          PASS
docker compose config       PASS
git diff --check            PASS
```
