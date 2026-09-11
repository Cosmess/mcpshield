# M6 Human Approval Verification

## Verdict

`PASS WITH GAPS`

M6 T1-T6 are implemented: approval state domain, policy-bound fingerprinting, reviewer
authorization, expiry, denial, in-memory repository, atomic one-time consumption, proxy
consumption through bound approval headers, authenticated reviewer HTTP routes, fixed
lifecycle metrics, PostgreSQL-backed persistence, migration, and an atomic conditional consume.

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
| M6-AC9 | State transition, race, and live PostgreSQL Testcontainer tests pass; consume uses conditional update | PASS |

## Executed focused gates

```text
gofmt -w internal/approval  PASS
go test ./internal/approval  PASS
go test -race ./internal/approval  PASS
go vet ./internal/approval  PASS
```

## Remaining work

- Add production secret management and connection-pool operational settings.

## Residual risks

PostgreSQL persistence is available when `MCP_SHIELD_DATABASE_URL` is configured. Production
secret management, connection pool sizing, and durable audit storage remain before production use.
