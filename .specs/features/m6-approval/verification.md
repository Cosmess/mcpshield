# M6 Human Approval Verification

## Verdict

`PASS WITH GAPS`

M6 T1-T6 are implemented: approval state domain, policy-bound fingerprinting, reviewer
authorization, expiry, denial, in-memory repository, atomic one-time consumption, proxy
consumption through bound approval headers, authenticated reviewer HTTP routes, fixed
lifecycle metrics, optional PostgreSQL-backed persistence, and an atomic conditional consume.

## Acceptance criteria evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| M6-AC1 | `Service.CreatePending` requires `REQUIRE_APPROVAL` and creates PENDING records | PASS |
| M6-AC2 | Record stores hashes, policy/risk metadata, and no raw bearer/DLP values | PASS |
| M6-AC3 | Reviewer scopes/roles and self-review rejection tests | PASS |
| M6-AC4 | Expiry checks on create/review/consume | PASS |
| M6-AC5 | Canonical fingerprint includes principal, request, policy decision, and risk context | PASS |
| M6-AC6 | Atomic in-memory consume with concurrent one-success test | PASS |
| M6-AC7 | Proxy creates PENDING without headers and consumes only a matching approved fingerprint | PASS |
| M6-AC8 | No AI approval transition exists; reviewer routes require authenticated reviewer context | PASS |
| M6-AC9 | State transition and race tests pass; PostgreSQL consume uses conditional update | PASS WITH GAP |

## Executed focused gates

```text
gofmt -w internal/approval  PASS
go test ./internal/approval  PASS
go test -race ./internal/approval  PASS
go vet ./internal/approval  PASS
```

## Remaining work

- Add PostgreSQL Testcontainer integration coverage for migration and repository transitions.
- Run the database-backed full repository gates before the implementation PR.

## Residual risks

PostgreSQL persistence is available when `MCP_SHIELD_DATABASE_URL` is configured, but
Testcontainer coverage, production secret management, connection pool sizing, and durable
audit storage remain before production use.
