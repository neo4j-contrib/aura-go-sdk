# PRD: v2beta1 Project Users

## Overview

Add `ProjectUserService` to the `v2beta1` package, covering the four project user management endpoints defined in the v2beta1 API spec. This mirrors the `OrganizationUserService` just shipped in v1.7.0 and follows the same service-per-resource pattern.

## Goals

- Expose all four project user operations: List, Add, UpdateRole, Remove.
- Wire the service into the v2beta1 `Client` with a public `ProjectUsers` field.
- Maintain full test coverage at parity with existing services.
- Provide a working example under `examples/v2beta1/`.
- Ship a changie changelog fragment for v1.8.0.

## Non-Goals

- Activity feed, billing, IP filters, or any other project-level endpoints.
- Changes to the existing `ProjectService` (List projects).
- Any v1 API changes.

## Requirements

### Functional Requirements

- REQ-F-001: `ProjectUserService.List(ctx, orgID, projectID)` calls `GET organizations/{org_id}/projects/{project_id}/users` and returns `*ListProjectUsersResponse`.
- REQ-F-002: `ProjectUserService.Add(ctx, orgID, projectID, req)` calls `POST organizations/{org_id}/projects/{project_id}/users` and returns `nil` on HTTP 201 (no body).
- REQ-F-003: `ProjectUserService.UpdateRole(ctx, orgID, projectID, userID, req)` calls `PATCH organizations/{org_id}/projects/{project_id}/users/{user_id}` and returns `*GetProjectUserResponse`.
- REQ-F-004: `ProjectUserService.Remove(ctx, orgID, projectID, userID)` calls `DELETE organizations/{org_id}/projects/{project_id}/users/{user_id}` and returns `nil` on HTTP 204.
- REQ-F-005: All methods validate `orgID` and `projectID` (both UUIDs) before making any API call. `UpdateRole` and `Remove` additionally validate `userID` (UUID) via the existing `utils.UserID` constructor.
- REQ-F-006: New path helpers `ProjectUsersPath(orgID, projectID)` and `SingleProjectUserPath(orgID, projectID, userID)` added to `internal/utils/path_helpers.go`.
- REQ-F-007: `client.ProjectUsers` field added to `v2beta1.Client` and wired in `NewClient`; service count incremented from 6 → 7.

### Non-Functional Requirements

- REQ-NF-001: Test coverage follows AGENTS.md matrix: `_Success`, `_InvalidID` (table-driven), `_NotFound`, `_AuthenticationError`, `_EmptyResult` (List only), `_ContextTimeout`, `_QuickCancellation`. `Add` and `UpdateRole` tests unmarshal `mock.lastBody` to assert request serialisation.
- REQ-NF-002: `go test -race ./...` and `golangci-lint run` pass with no new failures.
- REQ-NF-003: New service gets an `Added` changie entry.
- REQ-NF-004: README updated with a new `## v2beta1 Project User Operations` H2 section and a matching ToC entry.

## Technical Considerations

### Types

**File: `v2beta1/project_users.go`**

Types from spec `ProjectUser`:

```go
type ProjectUser struct {
    UserID       string   `json:"user_id"`
    Email        string   `json:"email"`
    ProjectRoles []string `json:"project_roles"`
}

type ListProjectUsersResponse struct { Data []ProjectUser `json:"data"` }
type GetProjectUserResponse   struct { Data ProjectUser  `json:"data"` }

type AddProjectUserRequest struct {
    UserID       string   `json:"user_id"`
    ProjectRoles []string `json:"project_roles"`
}

type UpdateProjectUserRequest struct {
    ProjectRoles []string `json:"project_roles"`
}
```

### Path Helpers

Add to `internal/utils/path_helpers.go`:

```go
// Returns endpoint path for users in a project
func ProjectUsersPath(orgID, projectID string) string {
    return resourcePath("organizations", orgID, "projects", projectID, "users")
}

// Returns endpoint path for a single user in a project
func SingleProjectUserPath(orgID, projectID, userID string) string {
    return resourcePath("organizations", orgID, "projects", projectID, "users", userID)
}
```

### ID Validation

`UserID` validator already exists (added in v1.7.0). No new validators needed — `OrganizationID` and `ProjectID` constructors cover the other two path parameters.

### Logging Levels

| Method     | Success path   |
|------------|----------------|
| List       | `DebugContext` |
| Add        | `InfoContext`  |
| UpdateRole | `InfoContext`  |
| Remove     | `InfoContext`  |

### Client Wiring

```go
// In Client struct
ProjectUsers ProjectUserService

// In NewClient
client.ProjectUsers = &projectUserService{...}
// services count: 6 → 7
```

### Add Returns No Body

The spec returns HTTP 201 with no response body for Add. The method signature is:

```go
Add(ctx context.Context, orgID, projectID string, req *AddProjectUserRequest) error
```

The mock in the test should return an empty body with no error; the implementation simply returns `nil` after a successful POST.

## Acceptance Criteria

- [ ] `ProjectUserService` interface defined in `interfaces.go` with List, Add, UpdateRole, Remove methods
- [ ] Compile-time check `var _ ProjectUserService = (*projectUserService)(nil)` added
- [ ] `ProjectUsersPath` and `SingleProjectUserPath` helpers added to `internal/utils/path_helpers.go`
- [ ] `client.ProjectUsers` wired in `NewClient`; services count = 7
- [ ] Full test coverage per AGENTS.md matrix for all 4 methods
- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run` passes
- [ ] Example created: `examples/v2beta1/projectUsers/`
- [ ] README updated with new section and ToC entry
- [ ] `changie new` fragment committed for v1.8.0

## Out of Scope

- Activity feed, billing, IP filters, graph analytics
- v1 API changes
- Extending existing `ProjectService`

## Open Questions

None — scope, service design, and ID formats confirmed.
