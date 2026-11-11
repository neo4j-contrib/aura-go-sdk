# Aura Go SDK — v2beta1 (beta)

This file documents the beta `github.com/neo4j-contrib/aura-go-sdk/v2beta1`
sub-package: how to construct the client, configure a default scope via the
client-level `WithDefaultOrg / WithDefaultProject / WithDefaultInstance` options,
and every operation against organizations, projects, instances, databases,
database backups, organization user, organization invite and project user
resources — all using the shared `ScopeOption` pattern
(`WithOrg`, `WithProject`, `WithInstance`, `WithDatabase`, `WithBackup`,
`WithUser`, `WithInvite`).

For the stable v1 API see [docs/v1.md](v1.md). For installation, a quick-start,
error handling and best practices see the main [README](../README.md).

A design write-up for the scoping model lives in `AGENTS.md`, section
"v2beta1 Design".

## Contents

- [v2beta1 API](#v2beta1-api)
- [v2beta1 Instance Operations](#v2beta1-instance-operations)
- [v2beta1 Database Operations](#v2beta1-database-operations)
- [v2beta1 Database Backup Operations](#v2beta1-database-backup-operations)
- [v2beta1 Organization User Operations](#v2beta1-organization-user-operations)
- [v2beta1 Organization Invite Operations](#v2beta1-organization-invite-operations)
- [v2beta1 Project User Operations](#v2beta1-project-user-operations)

---

## v2beta1 API

The `v2beta1` sub-package is part of the same Go module as the stable v1 client and targets the Neo4j Aura v2beta1 API. It exposes organization and project management operations not available in the stable v1 API.

### Installation

The `v2beta1` package is part of the same module as the stable v1 client:

```bash
go get github.com/neo4j-contrib/aura-go-sdk
```

### Import

```go
import v2beta1 "github.com/neo4j-contrib/aura-go-sdk/v2beta1"
```

### Client construction

```go
client, err := v2beta1.NewClient(
    v2beta1.WithCredentials("your-client-id", "your-client-secret"),
)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close()
```

Scope (organization → project → instance → database, plus the `backup`, `user`
and `invite` leaves) is supplied per call as variadic `ScopeOption` values —
`WithOrg`, `WithProject`, `WithInstance`, `WithDatabase`, `WithBackup`, `WithUser`
and `WithInvite`. Each service validates only the portion of the scope its resource
needs, so an org-level call ignores the lower fields.

A scope can instead be set once on the client with `WithDefaultOrg`,
`WithDefaultProject` or `WithDefaultInstance`, and any field left unset in a call
inherits that default — which keeps repeat operations to a one-line call. A
per-call option overrides the matching default, and when more than one option sets
the same field the last one wins:

```go
// Inherited default scope:
client, _ := v2beta1.NewClient(
    v2beta1.WithCredentials("your-client-id", "your-client-secret"),
    v2beta1.WithDefaultOrg(orgID),
    v2beta1.WithDefaultProject(projectID),
)

databases, err := client.Databases.List(ctx) // org + project inherited from the default
if err != nil {
    log.Fatalf("Error: %v", err)
}
```

The examples in the sections below pass scope explicitly on every call for
clarity; swap a constant org/project/instance for a `WithDefault*` client option
to drop the repeated argument. Operations that carry a request body take that body
first, then the scope options — for example
`client.Instances.Create(ctx, req, v2beta1.WithOrg(orgID), v2beta1.WithProject(projectID))`.

### List Organizations

```go
ctx := context.Background()

orgs, err := client.Organizations.List(ctx)
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, org := range orgs.Data {
    fmt.Printf("Organization: %s (ID: %s)\n", org.Name, org.ID)
}
```

### Get Organization Details

```go
ctx := context.Background()

org, err := client.Organizations.Get(ctx, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Organization: %s\n", org.Data.Name)
```

### List Projects

```go
ctx := context.Background()

projects, err := client.Projects.List(ctx, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, project := range projects.Data {
    fmt.Printf("Project: %s (ID: %s)\n", project.Name, project.ID)
}
```

## v2beta1 Instance Operations

### List Instances

```go
ctx := context.Background()

instances, err := client.Instances.List(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, instance := range instances.Data {
    fmt.Printf("Instance: %s (ID: %s) status: %s\n", instance.Name, instance.ID, instance.Status)
}
```

### Get Instance Details

```go
ctx := context.Background()

instance, err := client.Instances.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Instance: %s status: %s connection: %s\n",
    instance.Data.Name, instance.Data.Status, instance.Data.ConnectionURL)
```

### Create an Instance

```go
ctx := context.Background()

req := &v2beta1.CreateInstanceRequest{
    Name:          "my-instance",
    CloudProvider: "gcp",
    Region:        "europe-west1",
    Type:          "enterprise-db",
    Memory:        "8GB",
}

instance, err := client.Instances.Create(ctx, req, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Created: %s (ID: %s)\n", instance.Data.Name, instance.Data.ID)
// Save instance.Data.Password securely — it is only returned once.
```

### Update an Instance

```go
ctx := context.Background()

req := &v2beta1.UpdateInstanceRequest{
    Name:   "renamed-instance",
    Memory: "16GB",
}

instance, err := client.Instances.Update(ctx, req, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Updated: %s memory: %s\n", instance.Data.Name, instance.Data.Memory)
```

### Delete an Instance

```go
ctx := context.Background()

result, err := client.Instances.Delete(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Deleted instance: %s\n", result.Data.ID)
```

## v2beta1 Database Operations

`client.Databases` manages the databases within an Aura instance. Instance IDs and database IDs are both 8-character hex strings (e.g. `"abcdef01"`).

### List Databases

```go
ctx := context.Background()

databases, err := client.Databases.List(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, db := range databases.Data {
    fmt.Printf("Database ID: %s\n", db.ID)
}
```

### Get a Database

```go
ctx := context.Background()

db, err := client.Databases.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"), v2beta1.WithDatabase("12345678"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Database ID: %s\n", db.Data.ID)
```

### Create a Database

```go
ctx := context.Background()

db, err := client.Databases.Create(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Database created: %s\n", db.Data.ID)
```

### Delete a Database

```go
ctx := context.Background()

// ⚠️ WARNING: This is irreversible!
result, err := client.Databases.Delete(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"), v2beta1.WithDatabase("12345678"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Deleted database: %s\n", result.Data.ID)
```

## v2beta1 Database Backup Operations
> **Note:** This may get truncated to just client.Backups.  Not entirely sold on client.DatabaseBackups which is a bit jarring.  And we've already got client.snaphots. 
> Just store away that the naming here could alter in the future

`client.DatabaseBackups` manages backups for a specific database on an instance.

### List Backups

```go
ctx := context.Background()

backups, err := client.DatabaseBackups.List(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"), v2beta1.WithDatabase("12345678"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, backup := range backups.Data {
    fmt.Printf("Backup: %s status: %s exportable: %v\n", backup.ID, backup.Status, backup.Exportable)
}
```

### Create a Backup

```go
ctx := context.Background()

backup, err := client.DatabaseBackups.Create(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"), v2beta1.WithDatabase("12345678"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Backup created: %s status: %s\n", backup.Data.ID, backup.Data.Status)
```

### Get a Backup

```go
ctx := context.Background()

backup, err := client.DatabaseBackups.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("abcdef01"), v2beta1.WithDatabase("12345678"), v2beta1.WithBackup("backup-id"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Backup: %s status: %s exportable: %v\n",
    backup.Data.ID, backup.Data.Status, backup.Data.Exportable)
```

## v2beta1 Organization User Operations

`client.OrganizationUsers` manages users within an Aura organization.

### List Organization Users

```go
ctx := context.Background()

users, err := client.OrganizationUsers.List(ctx, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, user := range users.Data {
    fmt.Printf("User: %s (ID: %s) roles: %v\n", user.Email, user.UserID, user.OrganizationRoles)
}
```

### Get Organization User Details

```go
ctx := context.Background()

user, err := client.OrganizationUsers.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithUser("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("User: %s roles: %v\n", user.Data.Email, user.Data.OrganizationRoles)
for _, project := range user.Data.Projects {
    fmt.Printf("  Project: %s (ID: %s) roles: %v\n", project.Name, project.ID, project.ProjectRoles)
}
```

### Update Organization User Role

```go
ctx := context.Background()

req := &v2beta1.UpdateOrganizationUserRequest{
    OrganizationRoles: []string{"organization-member"},
}

user, err := client.OrganizationUsers.UpdateRole(ctx, req, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithUser("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Updated roles: %v\n", user.Data.OrganizationRoles)
```

### Remove a User from an Organization

```go
ctx := context.Background()

err := client.OrganizationUsers.Remove(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithUser("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Println("User removed from organization")
```

## v2beta1 Organization Invite Operations

`client.OrganizationInvites` manages pending invites for an Aura organization.

### List Organization Invites

```go
ctx := context.Background()

invites, err := client.OrganizationInvites.List(ctx, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, invite := range invites.Data {
    fmt.Printf("Invite: %s (ID: %s) status: %s\n", invite.Email, invite.ID, invite.Status)
}
```

### Create an Organization Invite

```go
ctx := context.Background()

req := &v2beta1.CreateOrganizationInviteRequest{
    Email: "new-member@example.com",
    Roles: []string{"organization-member"},
}

invite, err := client.OrganizationInvites.Create(ctx, req, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Invite sent to %s (ID: %s)\n", invite.Data.Email, invite.Data.ID)
```

### Create an Organization Invite with Project Roles

```go
ctx := context.Background()

req := &v2beta1.CreateOrganizationInviteRequest{
    Email: "new-member@example.com",
    Roles: []string{"organization-member"},
    ProjectInvites: []v2beta1.ProjectInviteEntry{
        {
            ProjectID:    "your-project-uuid",
            ProjectRoles: []string{"project-member"},
        },
    },
}

invite, err := client.OrganizationInvites.Create(ctx, req, v2beta1.WithOrg("your-org-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Invite sent to %s (ID: %s)\n", invite.Data.Email, invite.Data.ID)
```

### Delete an Organization Invite

```go
ctx := context.Background()

err := client.OrganizationInvites.Delete(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithInvite("a1b2c3d4-e5f6-7890-abcd-ef1234567890"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Println("Invite deleted")
```

## v2beta1 Project User Operations

`client.ProjectUsers` manages the users within an Aura project.

### List Project Users

```go
ctx := context.Background()

users, err := client.ProjectUsers.List(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

for _, user := range users.Data {
    fmt.Printf("User: %s (%s) roles: %v\n", user.Email, user.UserID, user.ProjectRoles)
}
```

### Add a Project User

```go
ctx := context.Background()

err := client.ProjectUsers.Add(ctx, &v2beta1.AddProjectUserRequest{
    UserID:       "user-uuid",
    ProjectRoles: []string{"viewer"},
}, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}
```

### Update a Project User's Role

```go
ctx := context.Background()

user, err := client.ProjectUsers.UpdateRole(ctx, &v2beta1.UpdateProjectUserRequest{
    ProjectRoles: []string{"editor"},
}, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithUser("user-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Updated roles: %v\n", user.Data.ProjectRoles)
```

### Remove a Project User

```go
ctx := context.Background()

err := client.ProjectUsers.Remove(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithUser("user-uuid"))
if err != nil {
    log.Fatalf("Error: %v", err)
}
```
