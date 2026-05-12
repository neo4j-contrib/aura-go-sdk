# AGENTS.md — aura-client

## Feedback Instructions

### TEST COMMANDS
```
go test -race ./... -timeout 120s
```

### BUILD COMMANDS
```
go build ./...
```

### LINT COMMANDS
```
golangci-lint run
```

### FORMAT COMMANDS
```
go fmt ./...
```

## Architecture

- Package `aura` (root): public API surface — service types, client, options
- `internal/api`: HTTP request service abstraction
- `internal/httpclient`: retryable HTTP client wrapping `retryablehttp`
- `internal/utils`: validation helpers (ValidateInstanceID, ValidateTenantID, ValidateSnapshotID, etc.)
- Each service (instanceService, tenantService, etc.) holds `api api.RequestService`, `timeout time.Duration`, `logger *slog.Logger`
- Services apply `context.WithTimeout` on every method call; cancellation propagates through the HTTP layer

## Test Mocks

- `mockAPIService` — ignores context entirely; use only when context behaviour is irrelevant
- `mockAPIServiceWithDelay` — respects context cancellation/deadline via `executeWithDelay`; use for context tests
- `mockAPIServiceWithCallback` — supports `OnGet`/`OnPost` hooks; use to inspect context values at the API layer
- Tests that assert a pre-cancelled or expired context is rejected MUST use `mockAPIServiceWithDelay` (not `mockAPIService`), because `mockAPIServiceWithDelay.executeWithDelay` calls `ctx.Err()` just as a real HTTP transport would

## Conventions

- Changie fragments go in `.changes/unreleased/`; kinds: `Fixed`, `Changed` (breaking), `Added`
- All service methods validate IDs before calling the API; empty check first, then format check
- Error messages follow the pattern `"invalid <thing> ID: <wrapped error>"`
- Log `ErrorContext` on API errors; `DebugContext` on success paths; `InfoContext` for mutating operations

## Aura API

- Module path: `github.com/neo4j-contrib/aura-go-sdk`
- Base URL: `https://api.neo4j.io`
- API version: `v1` (hardcoded in `client.go`; a future v2 will be a separate module, not a path suffix)
- Auth: OAuth2 client-credentials flow; token endpoint is `POST /oauth/token`
- Pagination: none — all list endpoints return the full result set in one response
- Rate limiting: 429 responses include `Retry-After`; the `retryablehttp` layer handles retries automatically
- Spec: https://neo4j.com/docs/aura/api/specification/

## Known Inconsistencies

- `AuraAPIClient.GraphAnalytics` field (public) vs `GDSSessionService` interface and `gdsSessionService` struct — the field was named before the service was formalised. Do not rename it unilaterally; it is a known breaking-change candidate for a future major version.

## PR Conventions

- One logical change per PR; breaking changes (renamed types, removed fields, interface additions) get their own PR
- A changie fragment in `.changes/unreleased/` is required for every user-visible change — run `changie new` and commit the generated YAML alongside your code
- PRs that touch only docs, CI, or tests may add the `no-changelog` label to bypass the changelog check workflow
- Required checks before review: `go test -race ./...`, `golangci-lint run`, `go build ./...` (includes `examples/`)
