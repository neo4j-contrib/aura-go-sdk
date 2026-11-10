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

// CloudProvider respresents the allowed cloud providers
type CloudProvider string

// Cloud Provider constants returned by the v2beta1 API.
const (
	CloudProviderAWS   CloudProvider = "aws"
	CloudProviderGCP   CloudProvider = "gcp"
	CloudProviderAzure CloudProvider = "azure"
)

// InstanceType represents the type of instance
type InstanceType string

// Instance type constants returned by the v2beta1 API.
const (
	InstanceTypeEnterpriseDB     InstanceType = "enterprise-db"
	InstanceTypeEnterpriseDS     InstanceType = "enterprise-ds"
	InstanceTypeBusinessCritical InstanceType = "business-critical"
	InstanceTypeProfessionalDB   InstanceType = "professional-db"
	InstanceTypeProfessionalDS   InstanceType = "professional-ds"
	InstanceTypeFreeDB           InstanceType = "free-db"
)

// InstanceStatus represents the lifecycle state of a v2beta1 instance.
type InstanceStatus string

// Instance status constants returned by the v2beta1 API.
const (
	InstanceStatusRunning               InstanceStatus = "running"
	InstanceStatusStopped               InstanceStatus = "stopped"
	InstanceStatusPaused                InstanceStatus = "paused"
	InstanceStatusCreating              InstanceStatus = "creating"
	InstanceStatusDestroying            InstanceStatus = "destroying"
	InstanceStatusUpdating              InstanceStatus = "updating"
	InstanceStatusPausing               InstanceStatus = "pausing"
	InstanceStatusResuming              InstanceStatus = "resuming"
	InstanceStatusRestoring             InstanceStatus = "restoring"
	InstanceStatusSuspending            InstanceStatus = "suspending"
	InstanceStatusLoading               InstanceStatus = "loading"
	InstanceStatusLoadingFailed         InstanceStatus = "loading failed"
	InstanceStatusOverwriting           InstanceStatus = "overwriting"
	InstanceStatusResizing              InstanceStatus = "resizing"
	InstanceStatusSuspended             InstanceStatus = "suspended"
	InstanceStatusUpgrading             InstanceStatus = "upgrading"
	InstanceStatusEncryptionKeyDeleted  InstanceStatus = "encryption key deleted"
	InstanceStatusEncryptionKeyError    InstanceStatus = "encryption key error"
	InstanceStatusEncryptionKeyNotFound InstanceStatus = "encryption key not found"
)

// // GraphAnalyticsMode represents chosen mode for G.D.S
type GraphAnalyticsMode string

// GraphAnalyticsType holds the type of graph analytics
const (
	GraphAnalyticsTypeServerless  GraphAnalyticsMode = "serverless"
	GraphAnalyticsTypeUnavailable GraphAnalyticsMode = "unavailable"
	GraphAnalyticsTypePlugin      GraphAnalyticsMode = "plugin"
)

// InstanceSummary holds the summary fields returned for each instance in a list response.
type InstanceSummary struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Status        InstanceStatus `json:"status"`
	CloudProvider string         `json:"cloud_provider"`
	CreatedAt     time.Time      `json:"created_at"`
}

// InstanceDetails holds the full set of fields returned for a single instance lookup.
type InstanceDetails struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Status          InstanceStatus     `json:"legacy_status"`
	CloudProvider   CloudProvider      `json:"cloud_provider"`
	Region          string             `json:"region"`
	Type            InstanceType       `json:"type"`
	Memory          string             `json:"memory"`
	Storage         string             `json:"storage"`
	GraphAnalytics  GraphAnalyticsMode `json:"graph_analytics"`
	VectorOptimised bool               `json:"vector_optimized"`
	MultiDatabase   bool               `json:"multi_database"`
}

// ListInstancesResponse wraps the list of instances returned by the API.
type ListInstancesResponse struct {
	Data []InstanceSummary `json:"data"`
}

// GetInstanceResponse wraps the single instance returned by the API.
type GetInstanceResponse struct {
	Data InstanceDetails `json:"data"`
}

// CreateInstanceRequest holds the fields required to provision a new instance.
type CreateInstanceRequest struct {
	Name           string             `json:"name"`
	CloudProvider  CloudProvider      `json:"cloud_provider"`
	Region         string             `json:"region"`
	Type           InstanceType       `json:"type"`
	Memory         string             `json:"memory"`
	MultiDatabase  bool               `json:"multi_database,omitempty"`  //Only if creating multiDB
	GraphAnalytics GraphAnalyticsMode `json:"graph_analytics,omitempty"` //Only if enabling GDS
}

// CreateInstanceData holds the response fields for a newly provisioned instance,
// including the database password.
type CreateInstanceData struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	CloudProvider   CloudProvider      `json:"cloud_provider"`
	ConnectionURL   string             `json:"connection_url"`
	CreatedAt       time.Time          `json:"created_at"`
	Region          string             `json:"region"`
	Type            InstanceType       `json:"type"`
	Username        string             `json:"username"`
	Password        string             `json:"password"`
	DefaultDatabase string             `json:"default_database_id"`
	Status          InstanceStatus     `json:"legacy_status"`
	GraphAnalytics  GraphAnalyticsMode `json:"graph_analytics"`
	VectorOptimised bool               `json:"vector_optimized"`
	MultiDatabase   bool               `json:"multi_database"`
	ProjectID       string             `json:"project_id"`
	Storage         string             `json:"storage"`
}

// CreateInstanceResponse wraps the response from a successful instance creation.
type CreateInstanceResponse struct {
	Data CreateInstanceData `json:"data"`
}

// UpdateInstanceRequest holds the fields that can be modified on an existing instance.
type UpdateInstanceRequest struct {
	Name   string `json:"name,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// DeleteInstanceResponse wraps the response returned when an instance is deleted.
type DeleteInstanceResponse struct {
	Data InstanceDetails `json:"data"`
}

// ============================================================================
// Service
// ============================================================================

// instanceService handles instance operations for the v2beta1 API.
//
// It is scoped through org->project(+instance); each call may override individual
// fields of the client's default scope via the ScopeOption values passed to it.
type instanceService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all instances in the organization and project selected by the scope
// options, inheriting any unset field from the client's default scope.
func (s *instanceService) List(ctx context.Context, opts ...ScopeOption) (*ListInstancesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing instances", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID))

	path := utils.InstancesPath(sc.orgID, sc.projectID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list instances", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListInstancesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal instances response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "instances listed successfully", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for the instance selected by the scope options.
func (s *instanceService) Get(ctx context.Context, opts ...ScopeOption) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting instance details", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("instanceID", sc.instanceID))

	path := utils.SingleInstancePath(sc.orgID, sc.projectID, sc.instanceID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get instance", slog.String("instanceID", sc.instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "instance retrieved successfully", slog.String("instanceID", sc.instanceID))
	return &result, nil
}

// Create provisions a new instance in the organization and project selected by the
// scope options. The request body precedes the trailing scope options.
func (s *instanceService) Create(ctx context.Context, req *CreateInstanceRequest, opts ...ScopeOption) (*CreateInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if req == nil {
		err := errors.New("request must not be nil")
		s.logger.ErrorContext(ctx, "nil create instance request", slog.String("error", err.Error()))
		return nil, err
	}

	if err := validateCreateInstanceRequest(req); err != nil {
		s.logger.ErrorContext(ctx, "invalid create instance request", slog.String("error", err.Error()))
		return nil, err
	}

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "creating instance", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("name", req.Name))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal create instance request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.InstancesPath(sc.orgID, sc.projectID)

	resp, err := s.api.Post(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create instance", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "instance created successfully", slog.String("instanceID", result.Data.ID), slog.String("name", result.Data.Name))
	return &result, nil
}

// Update modifies the configuration of the instance selected by the scope options.
// The request body precedes the trailing scope options.
func (s *instanceService) Update(ctx context.Context, req *UpdateInstanceRequest, opts ...ScopeOption) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if req == nil {
		err := errors.New("request must not be nil")
		s.logger.ErrorContext(ctx, "nil update instance request", slog.String("error", err.Error()))
		return nil, err
	}
	if req.Name == "" && req.Memory == "" {
		err := errors.New("at least one of name or memory must be set in UpdateInstanceRequest")
		s.logger.ErrorContext(ctx, "empty update instance request", slog.String("error", err.Error()))
		return nil, err
	}

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updating instance", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("instanceID", sc.instanceID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update instance request", slog.String("error", err.Error()))
		return nil, err
	}

	path := utils.SingleInstancePath(sc.orgID, sc.projectID, sc.instanceID)

	resp, err := s.api.Patch(ctx, path, string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update instance", slog.String("instanceID", sc.instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "instance updated successfully", slog.String("instanceID", sc.instanceID))
	return &result, nil
}

// Delete removes the instance selected by the scope options.
func (s *instanceService) Delete(ctx context.Context, opts ...ScopeOption) (*DeleteInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "deleting instance", slog.String("orgID", sc.orgID), slog.String("projectID", sc.projectID), slog.String("instanceID", sc.instanceID))

	path := utils.SingleInstancePath(sc.orgID, sc.projectID, sc.instanceID)

	resp, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete instance", slog.String("instanceID", sc.instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	if len(resp.Body) == 0 {
		s.logger.InfoContext(ctx, "instance deleted successfully", slog.String("instanceID", sc.instanceID))
		return &DeleteInstanceResponse{}, nil
	}

	var result DeleteInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal delete instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "instance deleted successfully", slog.String("instanceID", sc.instanceID))
	return &result, nil
}

// validateCreateInstanceRequest checks that all required fields are present.
func validateCreateInstanceRequest(req *CreateInstanceRequest) error {
	if req.Name == "" {
		return errors.New("name must not be empty")
	}
	if req.CloudProvider == "" {
		return errors.New("cloud_provider must not be empty")
	}
	if req.Region == "" {
		return errors.New("region must not be empty")
	}
	if req.Type == "" {
		return errors.New("type must not be empty")
	}
	if req.Memory == "" {
		return errors.New("memory must not be empty")
	}
	return nil
}
