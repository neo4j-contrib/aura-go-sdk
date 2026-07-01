package v2beta1

import "context"

// OrganizationService defines operations for managing Aura organizations.
type OrganizationService interface {
	// List returns all organizations accessible to the authenticated user.
	List(ctx context.Context, opts ...CallOption) (*ListOrganizationsResponse, error)
	// Get retrieves details for a specific organization. The org ID is resolved from
	// call options (via WithOrg) or from the client-level default (WithOrganization).
	Get(ctx context.Context, opts ...CallOption) (*GetOrganizationResponse, error)
}

// ProjectService defines operations for managing Aura projects within an organization.
type ProjectService interface {
	// List returns all projects within the resolved organization. The org ID is
	// resolved from call options (via WithOrg) or from the client-level default
	// (WithOrganization).
	List(ctx context.Context, opts ...CallOption) (*ListProjectsResponse, error)
}

// InstanceService defines operations for managing Aura instances within a project.
type InstanceService interface {
	// List returns all instances within the resolved organization and project.
	List(ctx context.Context, opts ...CallOption) (*ListInstancesResponse, error)
	// Get retrieves details for a specific instance by UUID.
	Get(ctx context.Context, instanceID string, opts ...CallOption) (*GetInstanceResponse, error)
	// Create provisions a new instance within the resolved organization and project.
	Create(ctx context.Context, req *CreateInstanceRequest, opts ...CallOption) (*CreateInstanceResponse, error)
	// Update modifies a specific instance's configuration.
	Update(ctx context.Context, instanceID string, req *UpdateInstanceRequest, opts ...CallOption) (*GetInstanceResponse, error)
	// Delete removes a specific instance.
	Delete(ctx context.Context, instanceID string, opts ...CallOption) (*DeleteInstanceResponse, error)
}

// DatabaseService defines operations for managing Aura databases.
type DatabaseService interface {
	// List returns all databases on an instance
	List(ctx context.Context, instanceID string, opts ...CallOption) (*ListDatabasesResponse, error)
	// Get returns information about a single database on an instance
	Get(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*GetDatabaseResponse, error)
	// Create provisions a new database for an instance
	Create(ctx context.Context, instanceID string, opts ...CallOption) (*CreateDatabaseResponse, error)
	// Delete removes a database from an Aura instance
	Delete(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*DeleteDatabaseResponse, error)
}

// DatabaseBackup Service defines operations for managing Aura database backups.
type DatabaseBackupService interface {
	// List returns all backups for the specified database within an instance.
	List(ctx context.Context, instanceID string, databaseID string, opts ...CallOption) (*ListBackupsResponse, error)
	// Create triggers a new backup for the specified database within an instance.
	Create(ctx context.Context, instanceID string, databaseID string, opts ...CallOption) (*CreateBackupResponse, error)
}

// Compile-time interface compliance checks
var (
	_ OrganizationService   = (*organizationService)(nil)
	_ ProjectService        = (*projectService)(nil)
	_ InstanceService       = (*instanceService)(nil)
	_ DatabaseService       = (*databaseService)(nil)
	_ DatabaseBackupService = (*databaseBackupService)(nil)
)
