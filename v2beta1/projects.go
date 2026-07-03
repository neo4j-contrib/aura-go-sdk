package v2beta1

import (
	"context"
	"encoding/json"

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
}

// List returns all projects within the given organization.
func (s *projectService) List(ctx context.Context, orgID string) (*ListProjectsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing projects", slog.String("orgID", orgID))

	path := utils.ProjectsPath(orgID)

	resp, err := s.api.Get(ctx, path)
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
