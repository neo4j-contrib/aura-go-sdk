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

// MFAEnrolledMethod represents a single MFA method enrolled by a user.
type MFAEnrolledMethod struct {
	ID         string `json:"id"`
	EnrolledAt string `json:"enrolled_at"`
}

// OrganizationUser represents a user within an Aura organization.
type OrganizationUser struct {
	UserID                     string              `json:"user_id"`
	Email                      string              `json:"email"`
	OrganizationRoles          []string            `json:"organization_roles"`
	ExemptFromAutomaticRemoval bool                `json:"exempt_from_automatic_removal"`
	MFAEnrollmentStatus        string              `json:"mfa_enrollment_status"`
	MFAEnrolledMethods         []MFAEnrolledMethod `json:"mfa_enrolled_methods"`
	LastActivityAt             *string             `json:"last_activity_at"`
}

// OrganizationUserProject represents a project the user belongs to within an organization.
type OrganizationUserProject struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ProjectRoles []string `json:"project_roles"`
}

// OrganizationUserDetails extends OrganizationUser with the user's project memberships.
type OrganizationUserDetails struct {
	OrganizationUser
	Projects []OrganizationUserProject `json:"projects"`
}

// ListOrganizationUsersResponse wraps the list of organization users returned by the API.
type ListOrganizationUsersResponse struct {
	Data []OrganizationUser `json:"data"`
}

// GetOrganizationUserResponse wraps the single organization user (with project details) returned by the API.
type GetOrganizationUserResponse struct {
	Data OrganizationUserDetails `json:"data"`
}

// UpdateOrganizationUserRequest is the request body for updating a user's organization roles.
type UpdateOrganizationUserRequest struct {
	OrganizationRoles []string `json:"organization_roles"`
}

// ============================================================================
// Service
// ============================================================================

// organizationUserService handles organization user operations for the v2beta1 API.
type organizationUserService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all users within the given organization.
func (s *organizationUserService) List(ctx context.Context, orgID string) (*ListOrganizationUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing organization users", slog.String("orgID", orgID))

	path := utils.OrgUsersPath(orgID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list organization users", slog.String("orgID", orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListOrganizationUsersResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization users response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization users listed successfully", slog.String("orgID", orgID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for a specific user within the given organization.
func (s *organizationUserService) Get(ctx context.Context, orgID, userID string) (*GetOrganizationUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.UserID(userID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting organization user details", slog.String("orgID", orgID), slog.String("userID", userID))

	path := utils.SingleOrgUserPath(orgID, userID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get organization user", slog.String("orgID", orgID), slog.String("userID", userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization user retrieved successfully", slog.String("orgID", orgID), slog.String("userID", userID))
	return &result, nil
}

// UpdateRole updates the organization roles of a specific user.
func (s *organizationUserService) UpdateRole(ctx context.Context, orgID, userID string, req *UpdateOrganizationUserRequest) (*GetOrganizationUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.UserID(userID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating organization user role", slog.String("orgID", orgID), slog.String("userID", userID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update organization user request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.SingleOrgUserPath(orgID, userID)

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update organization user role", slog.String("orgID", orgID), slog.String("userID", userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update organization user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "organization user role updated successfully", slog.String("orgID", orgID), slog.String("userID", userID))
	return &result, nil
}

// Remove removes a user from the given organization. Returns nil on HTTP 204.
func (s *organizationUserService) Remove(ctx context.Context, orgID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.UserID(userID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "removing organization user", slog.String("orgID", orgID), slog.String("userID", userID))

	path := utils.SingleOrgUserPath(orgID, userID)

	_, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove organization user", slog.String("orgID", orgID), slog.String("userID", userID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "organization user removed successfully", slog.String("orgID", orgID), slog.String("userID", userID))
	return nil
}
