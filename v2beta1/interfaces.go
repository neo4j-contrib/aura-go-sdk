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

// Compile-time interface compliance checks
var (
	_ OrganizationService = (*organizationService)(nil)
	_ ProjectService      = (*projectService)(nil)
)
