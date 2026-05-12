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
