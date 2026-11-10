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
//
// It is scoped through org(+user); each call may override individual fields of the
// client's default scope via the ScopeOption values passed to it.
type organizationUserService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all users within the organization selected by the scope options.
func (s *organizationUserService) List(ctx context.Context, opts ...ScopeOption) (*ListOrganizationUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing organization users", slog.String("orgID", sc.orgID))

	path := utils.OrgUsersPath(sc.orgID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list organization users", slog.String("orgID", sc.orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListOrganizationUsersResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization users response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization users listed successfully", slog.String("orgID", sc.orgID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for the user selected by the scope options (including
// WithUser).
func (s *organizationUserService) Get(ctx context.Context, opts ...ScopeOption) (*GetOrganizationUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.UserID(sc.userID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting organization user details", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))

	path := utils.SingleOrgUserPath(sc.orgID, sc.userID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get organization user", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization user retrieved successfully", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))
	return &result, nil
}

// UpdateRole updates the organization roles of the user selected by the scope
// options (including WithUser). The request body precedes the trailing scope
// options.
func (s *organizationUserService) UpdateRole(ctx context.Context, req *UpdateOrganizationUserRequest, opts ...ScopeOption) (*GetOrganizationUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.UserID(sc.userID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating organization user role", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update organization user request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.SingleOrgUserPath(sc.orgID, sc.userID)

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update organization user role", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetOrganizationUserResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update organization user response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "organization user role updated successfully", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))
	return &result, nil
}

// Remove removes the user selected by the scope options (including WithUser) from
// the organization. Returns nil on HTTP 204.
func (s *organizationUserService) Remove(ctx context.Context, opts ...ScopeOption) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.UserID(sc.userID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "removing organization user", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))

	path := utils.SingleOrgUserPath(sc.orgID, sc.userID)

	_, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove organization user", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "organization user removed successfully", slog.String("orgID", sc.orgID), slog.String("userID", sc.userID))
	return nil
}
