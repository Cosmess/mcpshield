# M6 Human Approval Verification

## Verdict

`PASS WITH GAPS`

M6 T1-T3 are implemented: approval state domain, policy-bound fingerprinting, reviewer
authorization, expiry, denial, in-memory repository, and atomic one-time consumption.
Proxy integration, lifecycle audit/metrics, and durable PostgreSQL storage remain.

## Acceptance criteria evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| M6-AC1 | `Service.CreatePending` requires `REQUIRE_APPROVAL` and creates PENDING records | PASS |
| M6-AC2 | Record stores hashes, policy/risk metadata, and no raw bearer/DLP values | PASS |
| M6-AC3 | Reviewer scopes/roles and self-review rejection tests | PASS |
| M6-AC4 | Expiry checks on create/review/consume | PASS |
| M6-AC5 | Canonical fingerprint includes principal, request, policy decision, and risk context | PASS |
| M6-AC6 | Atomic in-memory consume with concurrent one-success test | PASS |
| M6-AC7 | Proxy integration is not implemented yet | GAP |
| M6-AC8 | No AI approval transition exists in the domain API | PASS WITH GAP |
| M6-AC9 | State transition and race tests pass for the in-memory repository | PASS WITH GAP |

## Executed focused gates

```text
gofmt -w internal/approval  PASS
go test ./internal/approval  PASS
go test -race ./internal/approval  PASS
go vet ./internal/approval  PASS
```

## Remaining work

- Integrate `REQUIRE_APPROVAL` into the MCP proxy before upstream invocation.
- Add approval lifecycle audit events and fixed-status metrics.
- Replace in-memory repository with PostgreSQL-backed authoritative storage.
- Run full repository gates before the implementation PR.

## Residual risks

The current repository is process-local and loses approval state on restart. PostgreSQL,
transactional consumption, operational reviewer provisioning, and durable audit are required
before production use.
