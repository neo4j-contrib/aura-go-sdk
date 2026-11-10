package v2beta1

import (
	"context"
	"encoding/json"
	"fmt"
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
//
// It is scoped through org; each call may override the org field of the client's
// default scope via the ScopeOption values passed to it.
type organizationService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all organizations accessible to the authenticated user. It requires
// no scope.
func (s *organizationService) List(ctx context.Context) (*ListOrganizationsResponse, error) {
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

// Get retrieves details for the organization selected by the scope options (with a
// default org inherited from the client when unset).
func (s *organizationService) Get(ctx context.Context, opts ...ScopeOption) (*GetOrganizationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting organization details", slog.String("orgID", sc.orgID))

	path := fmt.Sprintf("organizations/%s", sc.orgID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get organization", slog.String("orgID", sc.orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization retrieved successfully", slog.String("orgID", sc.orgID), slog.String("name", result.Data.Name))
	return &result, nil
}
