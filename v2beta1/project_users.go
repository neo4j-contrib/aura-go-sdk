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
type projectUserService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all users within the given organization and project.
func (s *projectUserService) List(ctx context.Context, orgID, projectID string) (*ListProjectUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing project users", slog.String("orgID", orgID), slog.String("projectID", projectID))

	path := utils.ProjectUsersPath(orgID, projectID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list project users", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListProjectUsersResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal project users response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "project users listed successfully", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Add adds a user to the given organization and project. Returns nil on success (HTTP 201, no response body).
func (s *projectUserService) Add(ctx context.Context, orgID, projectID string, req *AddProjectUserRequest) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "adding user to project", slog.String("orgID", orgID), slog.String("projectID", projectID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal add project user request", slog.String("error", err.Error()))
		return err
	}

	path := utils.ProjectUsersPath(orgID, projectID)

	_, err = s.api.Post(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add user to project", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "user added to project successfully", slog.String("orgID", orgID), slog.String("projectID", projectID))
	return nil
}

// UpdateRole updates the role of a user within the given organization and project.
func (s *projectUserService) UpdateRole(ctx context.Context, orgID, projectID, userID string, req *UpdateProjectUserRequest) (*GetProjectUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.UserID(userID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating project user role", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update project user request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.SingleProjectUserPath(orgID, projectID, userID)

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update project user role", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetProjectUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update project user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "project user role updated successfully", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID))
	return &result, nil
}

// Remove removes a user from the given organization and project. Returns nil on success (HTTP 204, no response body).
func (s *projectUserService) Remove(ctx context.Context, orgID, projectID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.UserID(userID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "removing user from project", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID))

	path := utils.SingleProjectUserPath(orgID, projectID, userID)

	_, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove user from project", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "user removed from project successfully", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("userID", userID))
	return nil
}
