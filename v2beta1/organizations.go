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
type organizationService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all organizations accessible to the authenticated user.
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

// Get retrieves details for a specific organization by ID.
func (s *organizationService) Get(ctx context.Context, orgID string) (*GetOrganizationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting organization details", slog.String("orgID", orgID))

	path := fmt.Sprintf("organizations/%s", orgID)

	resp, err := s.api.Get(ctx, path)
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
