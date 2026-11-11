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
- Package `v2beta1`: beta sub-package of the **same** module (there is no `v2beta1/go.mod`) targeting the Aura v2beta1 API; its design lives in the "v2beta1 Design" section below. Import it as `v2beta1 "github.com/neo4j-contrib/aura-go-sdk/v2beta1"`
- `internal/api`: HTTP request service abstraction
- `internal/httpclient`: retryable HTTP client wrapping `retryablehttp`
- `internal/utils`: validation helpers (`ValidateInstanceID`, `ValidateTenantID`, `ValidateSnapshotID`, the `RequiredID` constructors + `Validate(ctx, logger, …)`, and the `*Path` builders used by `v2beta1`)
- `internal/apisurface`: renders the exported API of a package (`aura`, `v2beta1`) to deterministic text for the golden-file contract test (see "API Surface Golden Test" below)
- Each `aura` service holds `api api.RequestService`, `timeout time.Duration`, `logger *slog.Logger`; each `v2beta1` service holds those plus a `defaultScope requestScope` field
- Services apply `context.WithTimeout` on every method call; cancellation propagates through the HTTP layer

## Test Mocks

- `mockAPIService` — ignores context entirely; use only when context behaviour is irrelevant
- `mockAPIServiceWithDelay` — respects context cancellation/deadline via `executeWithDelay`; use for context tests
- `mockAPIServiceWithCallback` — supports `OnGet`/`OnPost` hooks; use to inspect context values at the API layer
- Tests that assert a pre-cancelled or expired context is rejected MUST use `mockAPIServiceWithDelay` (not `mockAPIService`), because `mockAPIServiceWithDelay.executeWithDelay` calls `ctx.Err()` just as a real HTTP transport would

## Conventions

- Changie fragments go in `.changes/unreleased/`; kinds: `Fixed`, `Changed` (breaking), `Added`; create non-interactively with `changie new --kind Added --body "..."`
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

## v2beta1 Design

This is the design contract for the beta `v2beta1` sub-package. Preserve it when adding or changing anything in `v2beta1/`.

**Module & import.** `v2beta1/` is a *sub-package of the single module* `github.com/neo4j-contrib/aura-go-sdk` — there is no `v2beta1/go.mod`; it builds with the same `go build ./...`. Import it as `v2beta1 "github.com/neo4j-contrib/aura-go-sdk/v2beta1"`. The Architecture note that "a future v2 will be a separate module, not a path suffix" refers to a *stable major version*; `v2beta1` is the beta and currently lives as a sub-package.

**Shared spine.** Every resource is reached through a fixed hierarchy —
`organizations/{org}/projects/{project}/instances/{instance}/databases/{database}`,
with `backups`, `users` and `invites` as leaves off that spine. Build paths with the
helpers in `internal/utils/path_helpers.go` (`InstancesPath`, `SingleInstancePath`,
`BackupsPath`, `ProjectUsersPath`, …); do **not** hand-concatenate.

**Shared scoping — `v2beta1/scope.go`.** A single unexported
`requestScope{orgID, projectID, instanceID, databaseID, backupID, userID, inviteID}`
resolves the scope for every service. The public surface is:
- per-call `ScopeOption`s: `WithOrg`, `WithProject`, `WithInstance`, `WithDatabase`,
  `WithBackup`, `WithUser`, `WithInvite`
- client-level defaults: `WithDefaultOrg`, `WithDefaultProject`, `WithDefaultInstance`
  set one shared `defaultScope` that every service copies at construction (there is no
  per-service default — it is client-wide).

Each method resolves scope with `sc := resolveScope(s.defaultScope, opts)` and then
validates **only the fields its resource needs** via
`utils.Validate(ctx, s.logger, utils.OrganizationID(sc.orgID), …)` — an org-level
operation ignores the lower spine/leaf fields. A field a leaf doesn't use is simply
ignored.

**Precedence rules (the contract).** In order: start from the client default scope;
apply each option in slice order; **later options win** (`resolveScope` is
last-option-wins). Concretely: an unset field inherits the default; a per-call option
overrides its matching default; when an option is passed more than once, the last wins.
This order is the behaviour the golden and behavioural tests lock in — do not change it.

**Body-first.** Operations that carry a typed request body take the body *first*, with
scope options trailing:
`client.Instances.Create(ctx, req, v2beta1.WithOrg(org), v2beta1.WithProject(proj))`,
and likewise `.Add(ctx, req, …)`, `.Update(ctx, req, …)`, `.UpdateRole(ctx, req, …)`.
Keep body-first; scope options always trail the body.

**The eight services** (`api`, `timeout`, `logger`, `defaultScope` are every
service's fields), and the fields each operation validates in order:
| Service | Operations and required scope |
|---|---|
| `organizationService` | `List` (none); `Get` → org |
| `projectService` | `List` → org |
| `instanceService` | `List`/`Create` → org, project; `Get`/`Update`/`Delete` → + instance |
| `databaseService` | `List`/`Create` → org, project, instance; `Get`/`Delete` → + database |
| `databaseBackupService` | `List`/`Create` → org, project, instance, database; `Get` → + backup |
| `organizationUserService` | `List` → org; `Get`/`UpdateRole`/`Remove` → + user |
| `organizationInviteService` | `List`/`Create` → org; `Delete` → + invite |
| `projectUserService` | `List`/`Add` → org, project; `UpdateRole`/`Remove` → + user |

**No nested "view layer".** Deliberately *not* provided: there is no
`client.Instances(...).Databases(...)` hierarchical sub-client API. Scope is flat — one
shared `ScopeOption` set per call — and that nested-builder design was explicitly dropped.
Do not re-introduce it; add new scope via a new leaf `WithXxx` option + a matching
field on `requestScope`, mirroring `WithBackup`/`WithUser`/`WithInvite`.

**ID validation hints.** The `MissingMsg`s in `internal/utils/validate_id.go` are
reworded to point callers at the real option names (e.g. *"provide it via the WithOrg
call option or WithDefaultOrg client option"*). Keep that wording when adding a new id;
the test matrices assert these substrings.

**User-Agent telemetry.** The `v2beta1` client tags its default User-Agent with a
`/v2beta1/` segment (i.e. `aura-go-sdk/v2beta1/<version>`) so server-side telemetry can tell
`v1` from `v2` traffic when a consumer uses both. Preserve it unless the change is made in
concert with the v1 root.

**Changelog / breaking changes.** `v2beta1` is a **beta**, so interface changes — adding
or renaming/removing methods, signature changes, any exported-surface delta (the golden
test goes red) — are *expected* and recorded as changie kind **`Changed`** with the house
prefix `v2beta1: <what changed>`; do **not** hedge a beta change with "breaking" wording.
A wholly new type/service is `Added`; an interface-method addition is `Changed`.

## v2beta1 Test conventions

- Each service test file has a zero-default helper `createTest<X>Service(mock)` (default
  scope is the zero value) **and** a precedence helper
  `createTest<X>ServiceWithDefaultScope(mock *mockAPIService, def requestScope)` that sets
  `defaultScope: def` — mirror `databases_test.go`.
- In addition to the standard set (`_Success`, `_InvalidID`, `_NotFound`,
  `_AuthenticationError`, and for Lists `_EmptyResult`; context tests use
  `mockAPIServiceWithDelay` per "Test Mocks" above), every service should carry
  behavioural precedence coverage: `_DefaultScope` (fully inherited, no options),
  `_PerCallOverride` / `_DefaultScopeAndPerCall<Leaf>` (one field or the leaf supplied per
  call, rest inherited), `_LastOptionWins` (the option passed twice, last wins), and a
  body-first-with-default test where the operation carries a request.
- Body assertions: for POST/PATCH unmarshal/inspect `mock.lastBody` to confirm the body was
  serialised; these tests assert only path + method + body, never response field contents
  unless the response shape is the point.

## Adding a service / endpoint in v2beta1

Mirror `instances.go` / `instances_test.go` (v2beta1 versions). In `v2beta1`, do **not**
use the positional skeleton above; follow these instead:

### Types — `<resource>.go` (same as v1, body-first noted for the doc comment)
- Declare request/response types in the service file. For any new leaf id, reuse
  `requestScope` + add a `WithXxx` constructor to `scope.go` and a field to `requestScope`;
  do not invent a second scoping mechanism.

### Service struct & methods — `<resource>.go`

```go
type fooService struct {
    api          api.RequestService
    timeout      time.Duration
    logger       *slog.Logger
    defaultScope requestScope
}

func (s *fooService) DoThing(ctx context.Context, req *DoThingRequest, opts ...ScopeOption) (*DoThingResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()

    if req == nil { /* reject nil body first */ return nil, errors.New("request must not be nil") }

    sc := resolveScope(s.defaultScope, opts)   // default wins on unset fields
    s.logger.DebugContext(ctx, "doing thing", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

    if err := utils.Validate(ctx, s.logger,        // validate ONLY the fields this op needs
        utils.OrganizationID(sc.orgID),
        utils.ProjectID(sc.projectID),
    ); err != nil {
        return nil, err
    }

    path := utils.FooPath(sc.orgID, sc.projectID, /* … leaf ids via their own params */)  // use the helper
    body, err := json.Marshal(req)                 // marshal first; POST/Patch body before scope
    /* … api.Post / Patch / Get / Delete … */
}
```

- Body-carrying ops take the body first, options trailing the variadic — `ctx, req, opts...`.
- Validate only the field set in the table above for the operation; `utils.Validate` stops at the first missing/malformed id and returns its reworded (`With*`-aware) hint.
- Path via the `internal/utils` helper; logging levels as in v1 (`DebugContext` reads, `InfoContext` mutations).

### Interface — `interfaces.go`, client wiring — `client.go`, tests, README, golden — exactly as the v1 steps 3–7, except the signature is `…, opts ...ScopeOption)` and any new client default is a `WithDefault*` Option writing to `o.config.defaultScope` (as `WithDefaultOrg/Project/Instance`).

## Known Inconsistencies

- `AuraAPIClient.GraphAnalytics` field (public) vs `GDSSessionService` interface and `gdsSessionService` struct — the field was named before the service was formalised. Do not rename it unilaterally; it is a known breaking-change candidate for a future major version.

## Adding a New Endpoint

> **Scope note:** this section describes the **stable v1 root** (`aura`) package, which uses *positional* `id string` parameters. Do **not** apply this skeleton in the `v2beta1` sub-package — it uses the shared `ScopeOption` model instead; follow "Adding a service / endpoint in v2beta1" below.

Follow the pattern used by `instances.go` / `instances_test.go`. Every new resource or sub-action touches the same five files in the same order.

### 1. Types — `<resource>.go`

Declare all request and response types in the same file as the service. Use a `// === Types ===` comment block to separate them from the service struct.

- Response wrapper: `type List<X>Response struct { Data []<X>Data }` / `type Get<X>Response struct { Data <X>Data }`
- Request body: unexported struct for internal use (e.g. `type overwriteInstanceRequest struct { … }`)
- JSON tags use `snake_case`. Fields that may be absent from the API response use `omitempty`.

### 2. Service struct and methods — `<resource>.go`

```go
type fooService struct {
    api     api.RequestService
    timeout time.Duration
    logger  *slog.Logger
}
```

Every method follows this exact skeleton — do not deviate:

```go
func (s *fooService) DoThing(ctx context.Context, id string) (*DoThingResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()

    s.logger.DebugContext(ctx, "doing thing", slog.String("id", id))

    if err := utils.ValidateFooID(id); err != nil {          // validate before every API call
        s.logger.ErrorContext(ctx, "invalid foo ID", slog.String("error", err.Error()))
        return nil, err
    }

    resp, err := s.api.Get(ctx, "foos/"+id)                  // or Post / Patch / Delete
    if err != nil {
        s.logger.ErrorContext(ctx, "failed to do thing", slog.String("id", id), slog.String("error", err.Error()))
        return nil, err
    }

    var result DoThingResponse
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        s.logger.ErrorContext(ctx, "failed to unmarshal response", slog.String("error", err.Error()))
        return nil, err
    }

    s.logger.DebugContext(ctx, "thing done", slog.String("id", id))  // InfoContext for mutations
    return &result, nil
}
```

**Path construction:**
| Pattern | Example |
|---|---|
| Collection | `"instances"` |
| Single resource | `"instances/" + instanceID` |
| Sub-action | `fmt.Sprintf("instances/%s/pause", instanceID)` |

**Logging level on success path:**
- Read operations (List, Get): `DebugContext`
- Mutating operations (Create, Update, Delete, and action sub-paths): `InfoContext`

**ID validation helpers** (`internal/utils`):
- `utils.ValidateInstanceID(id)` — 8-char hex string
- `utils.ValidateTenantID(id)` — standard UUID
- `utils.ValidateSnapshotID(id)` — standard UUID

**Request bodies:** marshal with `json.Marshal`, pass the result as `string(body)` to `api.Post` / `api.Patch`. Use `errors.New` for static error strings; `fmt.Errorf("… %w", err)` to wrap.

### 3. Interface — `interfaces.go`

Add the new method to the relevant service interface. Add (or create) a compile-time check:

```go
var _ FooService = (*fooService)(nil)
```

Adding a method to an existing interface is a **breaking change** — it requires a `Changed` changie entry.

### 4. Client wiring — `client.go`

Add a public field to `AuraAPIClient`:

```go
Foos FooService
```

Wire it in `NewClient` alongside the existing services:

```go
service.Foos = &fooService{
    api:     apiSvc,
    timeout: o.config.apiTimeout,
    logger:  clientLogger.With(slog.String("service", "fooService")),
}
```

Update the `slog.Int("services", N)` count in the `Info` log at the bottom of `NewClient`.

### 5. Tests — `<resource>_test.go`

Add two constructor helpers at the top of the file:

```go
func createTestFooService(mock *mockAPIService) *fooService {
    return &fooService{api: mock, timeout: 30 * time.Second, logger: testLogger()}
}

func createTestFooServiceWithTimeout(mock api.RequestService, timeout time.Duration) *fooService {
    return &fooService{api: mock, timeout: timeout, logger: testLogger()}
}
```

**Required tests per method:**

| Test | Mock to use | What to assert |
|---|---|---|
| `_Success` | `mockAPIService` | correct HTTP method, exact path, response fields correctly mapped |
| `_InvalidID` (table-driven) | `mockAPIService` | empty / too-short / bad-chars all return errors without calling API |
| `_NotFound` | `mockAPIService` with `err: &api.Error{StatusCode: 404, …}` | error type is `*api.Error`, `IsNotFound()` true |
| `_AuthenticationError` | `mockAPIService` with `err: &api.Error{StatusCode: 401, …}` | `IsUnauthorized()` true |
| `_EmptyResult` (List only) | `mockAPIService` | empty slice returned, no error |
| `_ContextTimeout` | `mockAPIServiceWithDelay` with delay > service timeout | `errors.Is(err, context.DeadlineExceeded)` |
| `_QuickCancellation` | `mockAPIServiceWithDelay`, pre-expired context | `context.DeadlineExceeded` or `context.Canceled` |

For POST/PATCH methods also unmarshal `mock.lastBody` and assert request fields were serialised correctly.

Use `mockAPIServiceWithDelay` (never `mockAPIService`) whenever the test must prove a context error is returned.

### 6. README.md

Add a new H2 section `## Foo Operations` to `README.md` and a matching entry in the Table of Contents at the top of the file. Each operation gets a fenced Go code block showing a minimal working call. Follow the style and verbosity of the existing `## Instance Operations` section exactly — no prose explanation inside the code block, no extra context beyond what's needed to call the method.

- No prose between code blocks. If a method has a variant (e.g. optional fields), promote it to its own H3 sub-section rather than using a prose lead-in sentence.
- Every code block must be a complete, callable snippet — never declare only `req` without the API call that uses it.

### 7. API surface golden test

Adding `Foos` to `AuraAPIClient` and `FooService` to `interfaces.go` changes the exported surface of the package, so `TestAPISurface` will fail until the golden file is regenerated:

```bash
UPDATE_GOLDEN=1 go test . ./v2beta1 -run TestAPISurface
```

Commit the resulting diff in `testdata/api_surface.golden` (or `v2beta1/testdata/api_surface.golden`) alongside the code change. Review the diff itself — it is the actual contract change being shipped. Adding a method to an existing interface, removing/renaming an exported identifier, or changing a signature is breaking and needs a `Changed` changie entry; adding a wholly new type/service is `Added`.

## API Surface Golden Test

`api_surface_test.go` (root) and `v2beta1/api_surface_test.go` each run `TestAPISurface`, which dumps every exported const/var/func/type/method/interface method in the package (bodies, doc comments, and receiver variable names stripped) and diffs it against `testdata/api_surface.golden`. This is a contract test: it fails on *any* exported-surface change, including ones that don't break the build (e.g. a new exported method on a concrete struct that isn't part of an interface).

- To regenerate after a deliberate change: `UPDATE_GOLDEN=1 go test . ./v2beta1 -run TestAPISurface`
- A failing `TestAPISurface` with no corresponding golden-file update in the diff means an accidental surface change — investigate before regenerating
- Pure comment/doc-string edits do not affect the golden output and won't trigger this test

## PR Conventions

- One logical change per PR; breaking changes (renamed types, removed fields, interface additions) get their own PR
- A changie fragment in `.changes/unreleased/` is required for every user-visible change — run `changie new` and commit the generated YAML alongside your code
- PRs that touch only docs, CI, or tests may add the `no-changelog` label to bypass the changelog check workflow
- Required checks before review: `go test -race ./...`, `golangci-lint run`, `go build ./...` (includes `examples/`) — `go test -race ./...` includes `TestAPISurface`, so an unreviewed surface change fails CI rather than just the changelog check
