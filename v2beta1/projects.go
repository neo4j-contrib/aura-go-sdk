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

// CreateProjectRequest holds the fields required to create a new project.
type CreateProjectRequest struct {
	Name             string   `json:"name"`
	BillingAccountID string   `json:"billing_account_id,omitempty"`
	SSOConfigIDs     []string `json:"sso_config_ids,omitempty"`
}

// CreateProjectResponse wraps the project returned by the API after creation.
type CreateProjectResponse struct {
	Data Project `json:"data"`
}

// UpdateProjectRequest holds the fields to update a project.
type UpdateProjectRequest struct {
	Name string `json:"name"`
}

// GetProjectResponse wraps the project returned by the API.
type GetProjectResponse struct {
	Data Project `json:"data"`
}

// DeleteProjectResponse wraps the response returned when a project is deleted.
type DeleteProjectResponse struct{}

// ============================================================================
// Service
// ============================================================================

// projectService handles project operations for the v2beta1 API.
//
// It is scoped through org; each call may override the org field of the client's
// default scope via the ScopeOption values passed to it.
type projectService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all projects in the organization selected by the scope options
// (with a default org inherited from the client when unset).
func (s *projectService) List(ctx context.Context, opts ...ScopeOption) (*ListProjectsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing projects", slog.String("orgID", sc.orgID))

	path := utils.ProjectsPath(sc.orgID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list projects", slog.String("orgID", sc.orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListProjectsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal projects response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "projects listed successfully", slog.String("orgID", sc.orgID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Create creates a new project in the organization selected by the scope options.
// The request body precedes the options.
func (s *projectService) Create(ctx context.Context, req *CreateProjectRequest, opts ...ScopeOption) (*CreateProjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if req == nil {
		return nil, errors.New("request must not be nil")
	}

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "creating project", slog.String("orgID", sc.orgID), slog.String("name", req.Name))

	path := utils.ProjectsPath(sc.orgID)

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal create project request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := s.api.Post(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create project", slog.String("orgID", sc.orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateProjectResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create project response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "project created successfully", slog.String("orgID", sc.orgID), slog.String("projectID", result.Data.ID))
	return &result, nil
}

// Get returns details for a specific project selected by the scope options.
func (s *projectService) Get(ctx context.Context, opts ...ScopeOption) (*GetProjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

	path := utils.SingleProjectPath(sc.orgID, sc.projectID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetProjectResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal project response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "project retrieved successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))
	return &result, nil
}

// Update updates a specific project selected by the scope options.
func (s *projectService) Update(ctx context.Context, req *UpdateProjectRequest, opts ...ScopeOption) (*GetProjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if req == nil {
		return nil, errors.New("request must not be nil")
	}

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("name", req.Name))

	path := utils.SingleProjectPath(sc.orgID, sc.projectID)

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update project request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetProjectResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal project response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "project updated successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))
	return &result, nil
}

// Delete removes a specific project selected by the scope options.
func (s *projectService) Delete(ctx context.Context, opts ...ScopeOption) (*DeleteProjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "deleting project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

	path := utils.SingleProjectPath(sc.orgID, sc.projectID)

	resp, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	if len(resp.Body) == 0 {
		s.logger.InfoContext(ctx, "project deleted successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))
		return &DeleteProjectResponse{}, nil
	}

	var result DeleteProjectResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal delete project response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "project deleted successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))
	return &result, nil
}
