package v2beta1

import "context"

// OrganizationService defines operations for managing Aura organizations.
type OrganizationService interface {
	// List returns all organizations accessible to the authenticated user.
	List(ctx context.Context) (*ListOrganizationsResponse, error)
	// Get retrieves details for a specific organization by ID.
	Get(ctx context.Context, orgID string) (*GetOrganizationResponse, error)
}

// ProjectService defines operations for managing Aura projects within an organization.
type ProjectService interface {
	// List returns all projects within the given organization.
	List(ctx context.Context, orgID string) (*ListProjectsResponse, error)
}

// InstanceService defines operations for managing Aura instances within a project.
type InstanceService interface {
	// List returns all instances within the given organization and project.
	List(ctx context.Context, orgID, projectID string) (*ListInstancesResponse, error)
	// Get retrieves details for a specific instance by UUID.
	Get(ctx context.Context, orgID, projectID, instanceID string) (*GetInstanceResponse, error)
	// Create provisions a new instance within the given organization and project.
	Create(ctx context.Context, orgID, projectID string, req *CreateInstanceRequest) (*CreateInstanceResponse, error)
	// Update modifies a specific instance's configuration.
	Update(ctx context.Context, orgID, projectID, instanceID string, req *UpdateInstanceRequest) (*GetInstanceResponse, error)
	// Delete removes a specific instance.
	Delete(ctx context.Context, orgID, projectID, instanceID string) (*DeleteInstanceResponse, error)
}

// DatabaseService defines operations for managing Aura databases.
type DatabaseService interface {
	// List returns all databases on the given instance.
	List(ctx context.Context, orgID, projectID, instanceID string) (*ListDatabasesResponse, error)
	// Get returns information about a single database on an instance.
	Get(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*GetDatabaseResponse, error)
	// Create provisions a new database for the given instance.
	Create(ctx context.Context, orgID, projectID, instanceID string) (*CreateDatabaseResponse, error)
	// Delete removes a database from an Aura instance.
	Delete(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*DeleteDatabaseResponse, error)
}

// DatabaseBackupService defines operations for managing Aura database backups.
type DatabaseBackupService interface {
	// List returns all backups for the specified database on an instance.
	List(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*ListBackupsResponse, error)
	// Create triggers a new backup for the specified database on an instance.
	Create(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*CreateBackupResponse, error)
	// Get returns information about a single backup of a database on an instance.
	Get(ctx context.Context, orgID, projectID, instanceID, databaseID, backupID string) (*GetBackupResponse, error)
}

// OrganizationUserService defines operations for managing users within an Aura organization.
type OrganizationUserService interface {
	// List returns all users within the given organization.
	List(ctx context.Context, orgID string) (*ListOrganizationUsersResponse, error)
	// Get retrieves details for a specific user within the given organization.
	Get(ctx context.Context, orgID, userID string) (*GetOrganizationUserResponse, error)
	// UpdateRole updates the organization roles of a specific user.
	UpdateRole(ctx context.Context, orgID, userID string, req *UpdateOrganizationUserRequest) (*GetOrganizationUserResponse, error)
	// Remove removes a user from the given organization.
	Remove(ctx context.Context, orgID, userID string) error
}

// OrganizationInviteService defines operations for managing invites within an Aura organization.
type OrganizationInviteService interface {
	// List returns all pending invites within the given organization.
	List(ctx context.Context, orgID string) (*ListOrganizationInvitesResponse, error)
	// Create sends an invite to the given email address to join the organization.
	Create(ctx context.Context, orgID string, req *CreateOrganizationInviteRequest) (*CreateOrganizationInviteResponse, error)
	// Delete revokes an existing invite from the given organization.
	Delete(ctx context.Context, orgID, inviteID string) error
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
)
