# M2 Authentication and Identity Tasks

## Gate commands

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
docker compose config
```

## T1: Add authentication configuration and principal types

### What

Add typed authentication configuration and the immutable internal principal contract.

### Where

`internal/config/`, `internal/identity/`

### Depends on

None.

### Tests

Valid production configuration, invalid issuer/audience/JWKS, development-mode safeguards, and principal claim mapping.

### Done when

- [ ] Production authentication settings validate before serving.
- [ ] Development token mode is explicit and cannot silently enable in production.
- [ ] Principal types contain no provider-specific dependency.

## T2: Implement bounded JWKS retrieval and cache

### What

Fetch issuer keys with timeout, cache successful responses, handle rotation, and coordinate concurrent refreshes.

### Where

`internal/jwks/`

### Depends on

T1.

### Tests

Cache hit, expiry, key rotation, malformed response, timeout, outage, and concurrent refresh.

### Done when

- [ ] JWKS requests are bounded and cancellable.
- [ ] Successful keys are cached for the configured TTL.
- [ ] Refresh failures never return stale unknown keys as valid.

## T3: Implement JWT validation

### What

Validate bearer tokens, algorithm/key type, signature, issuer, audience, expiration, not-before, and required claims.

### Where

`internal/auth/`

### Depends on

T1, T2.

### Tests

Valid token, wrong issuer, wrong audience, expired, not-before, bad signature, unknown key, unsupported algorithm, malformed token, and missing required claim.

### Done when

- [ ] Invalid tokens fail closed with stable classifications.
- [ ] Valid tokens produce an internal principal.
- [ ] Raw tokens and claims are absent from errors and logs.

## T4: Integrate authentication before MCP proxying

### What

Apply authentication middleware to `/mcp/` and expose the principal to the existing M1 proxy.

### Where

`internal/httpapi/`, `cmd/gateway/`

### Depends on

T3.

### Tests

Missing token, invalid token, valid token, spoofed identity headers, and no-outbound-on-failure.

### Done when

- [ ] Unauthenticated requests receive 401 before upstream activity.
- [ ] Valid identity reaches the downstream context.
- [ ] Inbound identity headers cannot override validated claims.

## T5: Add authentication observability and audit

### What

Record sanitized authentication outcomes and bounded metrics for success, failure class, latency, and JWKS refresh behavior.

### Where

`internal/auth/`, `internal/audit/`, `internal/observability/`

### Depends on

T3, T4.

### Tests

Token/claims redaction, outcome fields, metric labels, and liveness during JWKS outage.

### Done when

- [ ] No bearer token, raw JWT, or sensitive claim is emitted.
- [ ] Authentication failures are classifiable without exposing internals.
- [ ] Liveness remains independent from identity-provider availability.

## T6: Document M2 and verify the complete slice

### What

Document production configuration, local test mode, threat boundaries, and verification evidence for authentication and identity.

### Where

`README.md`, `docs/security/`, `.specs/features/m2-authentication/verification.md`

### Depends on

T1-T5.

### Tests

Complete auth-to-M1 integration suite and all repository gates.

### Done when

- [ ] Local setup explains secure and development modes separately.
- [ ] Verification maps all M2 acceptance criteria to evidence.
- [ ] Residual risks and M3 policy handoff are explicit.
