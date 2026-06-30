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

// Project represents an Aura project within an organization.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListProjectsResponse wraps the list of projects returned by the API.
type ListProjectsResponse struct {
	Data []Project `json:"data"`
}

// ============================================================================
// Service
// ============================================================================

// projectService handles project operations for the v2beta1 API.
type projectService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
	client  *Client
}

// List returns all projects within the resolved organization. The org ID is
// resolved from call options, falling back to the client default. Returns an
// error if no org ID is available from either source.
func (s *projectService) List(ctx context.Context, opts ...CallOption) (*ListProjectsResponse, error) {
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

	s.logger.DebugContext(ctx, "listing projects", slog.String("orgID", orgID))

	// Organization IDs in the v2beta1 API are standard UUIDs, so ValidateTenantID
	// (which validates the same UUID format) is intentionally reused here.
	if err := utils.ValidateTenantID(orgID); err != nil {
		s.logger.ErrorContext(ctx, "invalid organization ID", slog.String("error", err.Error()))
		return nil, errors.New("invalid organization ID: must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}

	resp, err := s.api.Get(ctx, "organizations/"+orgID+"/projects")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list projects", slog.String("orgID", orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListProjectsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal projects response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "projects listed successfully", slog.String("orgID", orgID), slog.Int("count", len(result.Data)))
	return &result, nil
}
