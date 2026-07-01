package v2beta1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
	"github.com/neo4j-contrib/aura-go-sdk/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// Organization represents an Aura organization.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListOrganizationsResponse wraps the list of organizations returned by the API.
type ListOrganizationsResponse struct {
	Data []Organization `json:"data"`
}

// GetOrganizationResponse wraps the single organization returned by the API.
type GetOrganizationResponse struct {
	Data Organization `json:"data"`
}

// ============================================================================
// Service
// ============================================================================

// organizationService handles organization operations for the v2beta1 API.
type organizationService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
	client  *Client
}

// List returns all organizations accessible to the authenticated user.
// opts is accepted for interface uniformity but is not used by this endpoint —
// organizations are a top-level resource and require no org or project scoping.
func (s *organizationService) List(ctx context.Context, opts ...CallOption) (*ListOrganizationsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	s.logger.DebugContext(ctx, "listing organizations")

	resp, err := s.api.Get(ctx, "organizations")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list organizations", slog.String("error", err.Error()))
		return nil, err
	}

	var result ListOrganizationsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organizations response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organizations listed successfully", slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for a specific organization. The org ID is resolved from
// call options, falling back to the client default. Returns an error if no org ID
// is available from either source.
func (s *organizationService) Get(ctx context.Context, opts ...CallOption) (*GetOrganizationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var clientDefault string
	if s.client != nil {
		s.client.mu.RLock()
		clientDefault = s.client.defaultOrgID
		s.client.mu.RUnlock()
	}

	orgID := resolveOrg(clientDefault, opts)

	// Check IDs are supplied and valid
	// Using new Validate function
	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting organization details", slog.String("orgID", orgID))

	// Organization IDs in the v2beta1 API are standard UUIDs, so ValidateTenantID
	// (which validates the same UUID format) is intentionally reused here.
	if err := utils.ValidateTenantID(orgID); err != nil {
		s.logger.ErrorContext(ctx, "invalid organization ID", slog.String("error", err.Error()))
		return nil, errors.New("invalid organization ID: must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}

	resp, err := s.api.Get(ctx, "organizations/"+orgID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get organization", slog.String("orgID", orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization retrieved successfully", slog.String("orgID", orgID), slog.String("name", result.Data.Name))
	return &result, nil
}
