# PRD: DX Improvements — Bug Fixes, Go Pattern Alignment, and Developer Gaps

## Overview

A set of targeted improvements to `aura-client` addressing bugs found in code review, alignment to idiomatic Go patterns, and gaps that reduce usability for library consumers. All changes are informed by the go-patterns skill and standard Go idioms (accept interfaces, return structs, errors are values, trust the zero value, functional options).

## Goals

- Fix silent data bugs that affect correctness (CMEK tenant filter, snapshot ID validation)
- Remove antipatterns that add noise without value (redundant `ctx.Err()` pre-checks)
- Align naming and error construction to Go conventions
- Give consumers essential missing primitives (`Close()`, `WithHTTPClient`, `WithUserAgent`, `WithDefaultHeaders`, dynamic version)
- Fix misleading public API surface (`QueriesPerSecond` field name, `CmekService` acronym casing)

## Non-Goals

- New API endpoints or resource types not already in the codebase
- Changing the module path or major version
- Altering the retry policy beyond what already exists
- Test infrastructure beyond what is needed to verify the fixes

## Requirements

### Functional Requirements

- REQ-F-001: Fix CMEK `List` query parameter from `tenantID` to `tenant_id`
- REQ-F-002: Add `utils.ValidateSnapshotID` call in `OverwriteFromSnapshot`
- REQ-F-003: Add missing error log in `tenants.GetMetrics` on API failure
- REQ-F-004: Rename `Query.QueriesPerSecond` to `Query.QueryExecutionTotal` with clear doc comment
- REQ-F-005: Downgrade "metric not found" log in `GetMetricValue` from `Error` to `Debug`
- REQ-F-006: Add `Close()` method to `AuraAPIClient` that drains idle HTTP connections
- REQ-F-007: Add `Close()` to `api.RequestService` interface and implement on `apiRequestService`
- REQ-F-008: Replace hardcoded `AuraAPIClientVersion` constant with `debug.ReadBuildInfo()` with fallback
- REQ-F-009: Add `WithHTTPClient(*http.Client) Option` for custom transport injection
- REQ-F-010: Add `WithUserAgent(string) Option` for caller-controlled User-Agent
- REQ-F-011: Add `WithDefaultHeaders(map[string]string) Option` for global preview/correlation headers
- REQ-F-012: Wire `HTTPClient` and `DefaultHeaders` through `api.Config` and into `httpclient`
- REQ-F-013: Rename `CmekService` → `CMEKService`, `Cmek` field → `CMEK`, and all `GetCmeks*` types to `GetCMEK*`
- REQ-F-014: Rename `gDSSessionService` struct to `gdsSessionService`
- REQ-F-015: Replace `fmt.Errorf("static string")` with `errors.New` in `graphanalytics.go` and `prometheus.go`
- REQ-F-016: Remove redundant import alias `utils "..."` in `instances.go` and `graphanalytics.go`
- REQ-F-017: Replace `utils.Marshal` calls with `json.Marshal` in `graphanalytics.go`
- REQ-F-018: Replace `strings.NewReader(string(data))` with `bytes.NewReader(data)` in `prometheus.go`
- REQ-F-019: Remove all redundant `ctx.Err()` pre-checks from service methods

### Non-Functional Requirements

- REQ-NF-001: All existing tests must pass with no regressions (`go test -race ./...`)
- REQ-NF-002: `go vet ./...` and `golangci-lint run` must produce no new errors
- REQ-NF-003: Breaking changes (CMEK rename, `QueriesPerSecond` rename) must be documented in a changelog fragment
- REQ-NF-004: Public API surface must not grow beyond the listed options and `Close()`

## Technical Considerations

- **CMEK rename is breaking**: `client.Cmek` → `client.CMEK`, `CmekService` → `CMEKService`, `GetCmeksResponse` → `GetCMEKsResponse`, `GetCmeksData` → `GetCMEKData`. Add a changelog entry marked `Changed`.
- **`QueriesPerSecond` rename is breaking**: callers that read this field by name will not compile after the rename. Add a changelog entry.
- **`Close()` on `api.RequestService`**: adding a method to an interface is breaking for any external implementation. Since this is an internal interface (unexported concrete type, only exported via the public service interfaces), this is safe.
- **`WithHTTPClient`**: when set, the injected client replaces the transport inside `retryablehttp`. The per-request timeout from `WithTimeout` is still applied via context, but the client's own `Timeout` field is not overridden.
- **`debug.ReadBuildInfo()`**: returns `(devel)` during `go run` / `go test`. Fallback to the literal version string in that case.
- **`WithDefaultHeaders`**: `Authorization`, `Content-Type`, and `User-Agent` must be blocked from override; silently ignore those keys.

## Acceptance Criteria

- [ ] `client.Cmek.List(ctx, tenantID)` no longer compiles; `client.CMEK.List(ctx, tenantID)` works and sends `?tenant_id=`
- [ ] `OverwriteFromSnapshot` returns a validation error for a malformed snapshot ID
- [ ] `client.Close()` exists, compiles, and calls `CloseIdleConnections` on the underlying transport
- [ ] `WithHTTPClient`, `WithUserAgent`, `WithDefaultHeaders` are accepted by `NewClient` without error
- [ ] `WithDefaultHeaders` silently ignores `Authorization`, `Content-Type`, `User-Agent` keys
- [ ] User-Agent sent in requests reflects the runtime-imported module version (via `ReadBuildInfo`)
- [ ] `QueryExecutionTotal` (renamed from `QueriesPerSecond`) is present on `QueryMetrics`
- [ ] No `fmt.Errorf` with static strings remain in `graphanalytics.go` or `prometheus.go`
- [ ] No `ctx.Err()` pre-checks remain in service method bodies
- [ ] `go test -race ./...` passes

## Out of Scope

- Adding new Aura API resource types
- Changing the retry policy or `Retry-After` header handling
- Adding `auratest` fake server (separate initiative)
- Pagination support for list endpoints
- Async polling helpers

## Open Questions

- Should `WithAPIVersion` also be added to allow staged access to a future API v2, or defer until needed?
- Should `InstanceData.Storage *string` be changed to `string` (potentially breaking for nil-checking callers)?
