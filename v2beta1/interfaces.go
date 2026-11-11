package v2beta1

import "context"

// OrganizationService defines operations for managing Aura organizations.
type OrganizationService interface {
	// List returns all organizations accessible to the authenticated user. It
	// requires no scope.
	List(ctx context.Context) (*ListOrganizationsResponse, error)
	// Get retrieves details for a specific organization selected by the scope
	// options.
	Get(ctx context.Context, opts ...ScopeOption) (*GetOrganizationResponse, error)
}

// ProjectService defines operations for managing Aura projects within an
// organization.
type ProjectService interface {
	// List returns all projects within the organization selected by the scope
	// options.
	List(ctx context.Context, opts ...ScopeOption) (*ListProjectsResponse, error)
}

// InstanceService defines operations for managing Aura instances within a project.
type InstanceService interface {
	// List returns all instances within the organization and project selected by
	// the scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListInstancesResponse, error)
	// Get retrieves details for a specific instance selected by the scope options.
	Get(ctx context.Context, opts ...ScopeOption) (*GetInstanceResponse, error)
	// Create provisions a new instance within the organization and project
	// selected by the scope options; the request body precedes the options.
	Create(ctx context.Context, req *CreateInstanceRequest, opts ...ScopeOption) (*CreateInstanceResponse, error)
	// Update modifies a specific instance's configuration, selected by the scope
	// options; the request body precedes the options.
	Update(ctx context.Context, req *UpdateInstanceRequest, opts ...ScopeOption) (*GetInstanceResponse, error)
	// Delete removes a specific instance selected by the scope options.
	Delete(ctx context.Context, opts ...ScopeOption) (*DeleteInstanceResponse, error)
}

// DatabaseService defines operations for managing Aura databases.
//
// Operations are scoped through org->project->instance->database. The scope is
// supplied per call via ScopeOption values (WithOrg, WithProject, WithInstance,
// WithDatabase) and inherits any unset field from the client's default scope.
type DatabaseService interface {
	// List returns all databases on the instance selected by the scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListDatabasesResponse, error)
	// Get returns information about a single database selected by the scope options.
	Get(ctx context.Context, opts ...ScopeOption) (*GetDatabaseResponse, error)
	// Create provisions a new database for the instance selected by the scope options.
	Create(ctx context.Context, opts ...ScopeOption) (*CreateDatabaseResponse, error)
	// Delete removes a database selected by the scope options from an instance.
	Delete(ctx context.Context, opts ...ScopeOption) (*DeleteDatabaseResponse, error)
}

// DatabaseBackupService defines operations for managing Aura database backups.
//
// Operations are scoped through org->project->instance->database, with WithBackup
// selecting a single backup. The scope inherits any unset field from the client's
// default scope.
type DatabaseBackupService interface {
	// List returns all backups for the database selected by the scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListBackupsResponse, error)
	// Create triggers a new backup for the database selected by the scope options.
	Create(ctx context.Context, opts ...ScopeOption) (*CreateBackupResponse, error)
	// Get returns information about the single backup selected by the scope
	// options (including WithBackup).
	Get(ctx context.Context, opts ...ScopeOption) (*GetBackupResponse, error)
}

// OrganizationUserService defines operations for managing users within an Aura
// organization.
type OrganizationUserService interface {
	// List returns all users within the organization selected by the scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListOrganizationUsersResponse, error)
	// Get retrieves details for a specific user selected by the scope options
	// (including WithUser).
	Get(ctx context.Context, opts ...ScopeOption) (*GetOrganizationUserResponse, error)
	// UpdateRole updates the organization roles of a specific user. The request
	// body precedes the scope options, which select org and user.
	UpdateRole(ctx context.Context, req *UpdateOrganizationUserRequest, opts ...ScopeOption) (*GetOrganizationUserResponse, error)
	// Remove removes a user from the organization selected by the scope options
	// (including WithUser).
	Remove(ctx context.Context, opts ...ScopeOption) error
}

// OrganizationInviteService defines operations for managing invites within an Aura
// organization.
type OrganizationInviteService interface {
	// List returns all pending invites within the organization selected by the
	// scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListOrganizationInvitesResponse, error)
	// Create sends an invite to join the organization selected by the scope
	// options; the request body precedes the options.
	Create(ctx context.Context, req *CreateOrganizationInviteRequest, opts ...ScopeOption) (*CreateOrganizationInviteResponse, error)
	// Delete revokes the invite selected by the scope options (including
	// WithInvite).
	Delete(ctx context.Context, opts ...ScopeOption) error
}

// ProjectUserService defines operations for managing users within an Aura project.
type ProjectUserService interface {
	// List returns all users within the organization and project selected by the
	// scope options.
	List(ctx context.Context, opts ...ScopeOption) (*ListProjectUsersResponse, error)
	// Add adds a user to the organization and project selected by the scope
	// options; the request body precedes the options.
	Add(ctx context.Context, req *AddProjectUserRequest, opts ...ScopeOption) error
	// UpdateRole updates the role of a specific user within the organization and
	// project selected by the scope options (including WithUser); the request
	// body precedes the options.
	UpdateRole(ctx context.Context, req *UpdateProjectUserRequest, opts ...ScopeOption) (*GetProjectUserResponse, error)
	// Remove removes a user from the organization and project selected by the
	// scope options (including WithUser).
	Remove(ctx context.Context, opts ...ScopeOption) error
}

// Compile-time interface compliance checks
var (
	_ OrganizationService       = (*organizationService)(nil)
	_ ProjectService            = (*projectService)(nil)
	_ InstanceService           = (*instanceService)(nil)
	_ DatabaseService           = (*databaseService)(nil)
	_ DatabaseBackupService     = (*databaseBackupService)(nil)
	_ OrganizationUserService   = (*organizationUserService)(nil)
	_ OrganizationInviteService = (*organizationInviteService)(nil)
	_ ProjectUserService        = (*projectUserService)(nil)
)
