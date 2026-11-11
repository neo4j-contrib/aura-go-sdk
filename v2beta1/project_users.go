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

// ProjectUser represents a user that is a member of an Aura project.
type ProjectUser struct {
	UserID       string   `json:"user_id"`
	Email        string   `json:"email"`
	ProjectRoles []string `json:"project_roles"`
}

// ListProjectUsersResponse wraps the list of project users returned by the API.
type ListProjectUsersResponse struct {
	Data []ProjectUser `json:"data"`
}

// GetProjectUserResponse wraps a single project user returned by the API.
type GetProjectUserResponse struct {
	Data ProjectUser `json:"data"`
}

// AddProjectUserRequest holds the fields required to add a user to a project.
type AddProjectUserRequest struct {
	UserID       string   `json:"user_id"`
	ProjectRoles []string `json:"project_roles"`
}

// UpdateProjectUserRequest holds the fields required to update a user's role in a project.
type UpdateProjectUserRequest struct {
	ProjectRoles []string `json:"project_roles"`
}

// ============================================================================
// Service
// ============================================================================

// projectUserService handles project user operations for the v2beta1 API.
//
// It is scoped through org->project(+user); each call may override individual
// fields of the client's default scope via the ScopeOption values passed to it.
type projectUserService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all users within the organization and project selected by the scope
// options.
func (s *projectUserService) List(ctx context.Context, opts ...ScopeOption) (*ListProjectUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing project users", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

	path := utils.ProjectUsersPath(sc.orgID, sc.projectID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list project users", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListProjectUsersResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal project users response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "project users listed successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Add adds a user to the organization and project selected by the scope options.
// The request body precedes the trailing scope options. Returns nil on success
// (HTTP 201, no response body).
func (s *projectUserService) Add(ctx context.Context, req *AddProjectUserRequest, opts ...ScopeOption) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "adding user to project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal add project user request", slog.String("error", err.Error()))
		return err
	}

	path := utils.ProjectUsersPath(sc.orgID, sc.projectID)

	_, err = s.api.Post(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add user to project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "user added to project successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))
	return nil
}

// UpdateRole updates the role of the user selected by the scope options (including
// WithUser) within the organization and project. The request body precedes the
// trailing scope options.
func (s *projectUserService) UpdateRole(ctx context.Context, req *UpdateProjectUserRequest, opts ...ScopeOption) (*GetProjectUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.UserID(sc.userID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating project user role", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update project user request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.SingleProjectUserPath(sc.orgID, sc.projectID, sc.userID)

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update project user role", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetProjectUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update project user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "project user role updated successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID))
	return &result, nil
}

// Remove removes the user selected by the scope options (including WithUser) from
// the organization and project. Returns nil on success (HTTP 204, no response
// body).
func (s *projectUserService) Remove(ctx context.Context, opts ...ScopeOption) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.UserID(sc.userID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "removing user from project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID))

	path := utils.SingleProjectUserPath(sc.orgID, sc.projectID, sc.userID)

	_, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove user from project", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "user removed from project successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("userID", sc.userID))
	return nil
}
