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

// ProjectInviteEntry represents a project-level role assignment within an invite.
type ProjectInviteEntry struct {
	ProjectID    string   `json:"project_id"`
	ProjectRoles []string `json:"project_roles"`
}

// OrganizationInvite represents a pending invite to an Aura organization.
type OrganizationInvite struct {
	ID                string               `json:"id"`
	Email             string               `json:"email"`
	InvitedBy         string               `json:"invited_by"`
	OrganizationID    string               `json:"organization_id"`
	OrganizationRoles []string             `json:"organization_roles"`
	ProjectInvites    []ProjectInviteEntry `json:"project_invites"`
	Status            string               `json:"status"`
	ExpiresAt         string               `json:"expires_at"`
}

// ListOrganizationInvitesResponse wraps the list of organization invites returned by the API.
type ListOrganizationInvitesResponse struct {
	Data []OrganizationInvite `json:"data"`
}

// CreateOrganizationInviteResponse wraps the single invite returned after creation.
type CreateOrganizationInviteResponse struct {
	Data OrganizationInvite `json:"data"`
}

// CreateOrganizationInviteRequest is the request body for creating an organization invite.
type CreateOrganizationInviteRequest struct {
	Email          string               `json:"email"`
	Roles          []string             `json:"roles"`
	ProjectInvites []ProjectInviteEntry `json:"project_invites,omitempty"`
}

// ============================================================================
// Service
// ============================================================================

// organizationInviteService handles organization invite operations for the v2beta1 API.
type organizationInviteService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all pending invites within the given organization.
func (s *organizationInviteService) List(ctx context.Context, orgID string) (*ListOrganizationInvitesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing organization invites", slog.String("orgID", orgID))

	path := utils.OrgInvitesPath(orgID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list organization invites", slog.String("orgID", orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListOrganizationInvitesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal organization invites response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "organization invites listed successfully", slog.String("orgID", orgID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Create sends an invite to the given email address to join the organization.
func (s *organizationInviteService) Create(ctx context.Context, orgID string, req *CreateOrganizationInviteRequest) (*CreateOrganizationInviteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "creating organization invite", slog.String("orgID", orgID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal create organization invite request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.OrgInvitesPath(orgID)

	resp, err := s.api.Post(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create organization invite", slog.String("orgID", orgID), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateOrganizationInviteResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create organization invite response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "organization invite created successfully", slog.String("orgID", orgID))
	return &result, nil
}

// Delete revokes an existing invite from the given organization. Returns nil on HTTP 204.
func (s *organizationInviteService) Delete(ctx context.Context, orgID, inviteID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.InviteID(inviteID),
	); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "deleting organization invite", slog.String("orgID", orgID), slog.String("inviteID", inviteID))

	path := utils.SingleInvitePath(orgID, inviteID)

	_, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete organization invite", slog.String("orgID", orgID), slog.String("inviteID", inviteID), slog.String("error", err.Error()))
		return err
	}

	s.logger.InfoContext(ctx, "organization invite deleted successfully", slog.String("orgID", orgID), slog.String("inviteID", inviteID))
	return nil
}
