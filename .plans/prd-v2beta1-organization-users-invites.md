# PRD: v2beta1 Organization Users & Invites

## Overview

Add `OrganizationUserService` and `OrganizationInviteService` to the `v2beta1` package, covering the organization user management and invite management endpoints defined in the v2beta1 API spec. This follows the same service-per-resource pattern already established by `organizationService`, `projectService`, `instanceService`, etc.

## Goals

- Expose all four organization user operations: List, Get, Update role, Remove.
- Expose all three organization invite operations: List, Create, Delete.
- Wire both services into the v2beta1 `Client` with public fields.
- Maintain full test coverage at parity with existing services.
- Provide working examples under `examples/v2beta1/`.
- Ship a changie changelog fragment for v1.7.0.

## Non-Goals

- Project-level user management (`/projects/{project_id}/users`).
- Billing, activity-feed, IP filters, or graph-analytics endpoints.
- Any v1 API changes.

## Requirements

### Functional Requirements

- REQ-F-001: `OrganizationUserService.List(ctx, orgID)` calls `GET organizations/{org_id}/users` and returns `*ListOrganizationUsersResponse`.
- REQ-F-002: `OrganizationUserService.Get(ctx, orgID, userID)` calls `GET organizations/{org_id}/users/{user_id}` and returns `*GetOrganizationUserResponse` (includes `projects` field — `OrganizationUserDetails` schema).
- REQ-F-003: `OrganizationUserService.UpdateRole(ctx, orgID, userID, req)` calls `PATCH organizations/{org_id}/users/{user_id}` and returns `*GetOrganizationUserResponse`.
- REQ-F-004: `OrganizationUserService.Remove(ctx, orgID, userID)` calls `DELETE organizations/{org_id}/users/{user_id}` and returns `nil` on HTTP 204.
- REQ-F-005: `OrganizationInviteService.List(ctx, orgID)` calls `GET organizations/{org_id}/invites` and returns `*ListOrganizationInvitesResponse`.
- REQ-F-006: `OrganizationInviteService.Create(ctx, orgID, req)` calls `POST organizations/{org_id}/invites` and returns `*CreateOrganizationInviteResponse`.
- REQ-F-007: `OrganizationInviteService.Delete(ctx, orgID, inviteID)` calls `DELETE organizations/{org_id}/invites/{invite_id}` and returns `nil` on HTTP 204.
- REQ-F-008: All methods validate `orgID` (UUID) and additional IDs (userID, inviteID — both UUIDs) before making any API call.
- REQ-F-009: New `UserID` and `InviteID` validator constructors added to `internal/utils/validate_id.go`.
- REQ-F-010: Path helpers `OrgUsersPath`, `SingleOrgUserPath`, `OrgInvitesPath`, and `SingleInvitePath` already exist in `internal/utils/path_helpers.go` — use them directly.
- REQ-F-011: `client.OrganizationUsers` and `client.OrganizationInvites` fields added to `v2beta1.Client` and wired in `NewClient`.

### Non-Functional Requirements

- REQ-NF-001: Test coverage follows AGENTS.md matrix: `_Success`, `_InvalidID` (table-driven), `_NotFound`, `_AuthenticationError`, `_EmptyResult` (List only), `_ContextTimeout`, `_QuickCancellation`. POST/PATCH tests also unmarshal `mock.lastBody` to assert request serialization.
- REQ-NF-002: `go test -race ./...` and `golangci-lint run` pass with no new failures.
- REQ-NF-003: Breaking-change interface additions get a `Changed` changie entry; new services get `Added` entries.
- REQ-NF-004: README updated with two new H2 sections (`## v2beta1 Organization User Operations`, `## v2beta1 Organization Invite Operations`) and matching ToC entries.

## Technical Considerations

### Types

**File: `v2beta1/organization_users.go`**

Key types to define (from spec `OrganizationUser`, `OrganizationUserDetails`, `OrganizationUserProject`):

```go
type OrganizationUser struct {
    UserID                      string                `json:"user_id"`
    Email                       string                `json:"email"`
    OrganizationRoles           []string              `json:"organization_roles"`
    ExemptFromAutomaticRemoval  bool                  `json:"exempt_from_automatic_removal"`
    MFAEnrollmentStatus         string                `json:"mfa_enrollment_status"`
    MFAEnrolledMethods          []MFAEnrolledMethod   `json:"mfa_enrolled_methods"`
    LastActivityAt              *string               `json:"last_activity_at"`
}

type MFAEnrolledMethod struct {
    ID         string `json:"id"`
    EnrolledAt string `json:"enrolled_at"`
}

type OrganizationUserProject struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    ProjectRoles []string `json:"project_roles"`
}

type OrganizationUserDetails struct {
    OrganizationUser
    Projects []OrganizationUserProject `json:"projects"`
}

type ListOrganizationUsersResponse struct { Data []OrganizationUser `json:"data"` }
type GetOrganizationUserResponse   struct { Data OrganizationUserDetails `json:"data"` }
type UpdateOrganizationUserRequest struct { OrganizationRoles []string `json:"organization_roles"` }
```

**File: `v2beta1/organization_invites.go`**

Key types (from spec `OrganizationInvite`):

```go
type OrganizationInvite struct {
    ID                string                `json:"id"`
    Email             string                `json:"email"`
    InvitedBy         string                `json:"invited_by"`
    OrganizationID    string                `json:"organization_id"`
    OrganizationRoles []string              `json:"organization_roles"`
    ProjectInvites    []ProjectInviteEntry  `json:"project_invites"`
    Status            string                `json:"status"`
    ExpiresAt         string                `json:"expires_at"`
}

type ProjectInviteEntry struct {
    ProjectID    string   `json:"project_id"`
    ProjectRoles []string `json:"project_roles"`
}

type ListOrganizationInvitesResponse   struct { Data []OrganizationInvite `json:"data"` }
type CreateOrganizationInviteResponse  struct { Data OrganizationInvite   `json:"data"` }
type CreateOrganizationInviteRequest struct {
    Email          string               `json:"email"`
    Roles          []string             `json:"roles"`
    ProjectInvites []ProjectInviteEntry `json:"project_invites,omitempty"`
}
```

### ID Validation

- `userID` and `inviteID` are both standard UUIDs — add `UserID(value string) RequiredID` and `InviteID(value string) RequiredID` constructors to `internal/utils/validate_id.go`.

### Logging Levels

| Method         | Success path        |
|----------------|---------------------|
| List           | `DebugContext`      |
| Get            | `DebugContext`      |
| UpdateRole     | `InfoContext`       |
| Remove         | `InfoContext`       |
| Invite.List    | `DebugContext`      |
| Invite.Create  | `InfoContext`       |
| Invite.Delete  | `InfoContext`       |

### Client Wiring

```go
// In Client struct
OrganizationUsers   OrganizationUserService
OrganizationInvites OrganizationInviteService

// In NewClient
client.OrganizationUsers = &organizationUserService{...}
client.OrganizationInvites = &organizationInviteService{...}
```

Update `slog.Int("services", N)` count from 4 → 6.

## Acceptance Criteria

- [ ] `OrganizationUserService` interface defined in `interfaces.go` with List, Get, UpdateRole, Remove methods
- [ ] `OrganizationInviteService` interface defined in `interfaces.go` with List, Create, Delete methods
- [ ] Compile-time checks `var _ OrganizationUserService = (*organizationUserService)(nil)` etc. added
- [ ] `UserID` and `InviteID` validators added to `internal/utils/validate_id.go`
- [ ] `client.OrganizationUsers` and `client.OrganizationInvites` wired in `NewClient`
- [ ] Full test coverage per AGENTS.md matrix for all 7 methods
- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run` passes
- [ ] Two examples created: `examples/v2beta1/orgUsers/` and `examples/v2beta1/orgInvites/`
- [ ] README updated with new sections and ToC entries
- [ ] `changie new` fragment committed for v1.7.0

## Out of Scope

- Project-level user management
- Billing, IP filters, activity feed, graph analytics
- v1 API changes

## Open Questions

None — scope, service design, and ID formats confirmed.
