package v2beta1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
	"github.com/neo4j-contrib/aura-go-sdk/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// InstanceStatus represents the lifecycle state of a v2beta1 instance.
type InstanceStatus string

// Instance status constants returned by the v2beta1 API.
const (
	InstanceStatusRunning    InstanceStatus = "running"
	InstanceStatusStopped    InstanceStatus = "stopped"
	InstanceStatusPaused     InstanceStatus = "paused"
	InstanceStatusCreating   InstanceStatus = "creating"
	InstanceStatusDestroying InstanceStatus = "destroying"
	InstanceStatusUpdating   InstanceStatus = "updating"
	InstanceStatusPausing    InstanceStatus = "pausing"
	InstanceStatusResuming   InstanceStatus = "resuming"
	InstanceStatusRestoring  InstanceStatus = "restoring"
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
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Status          InstanceStatus `json:"legacy_status"`
	CloudProvider   string         `json:"cloud_provider"`
	Region          string         `json:"region"`
	Type            string         `json:"type"`
	Memory          string         `json:"memory"`
	Storage         string         `json:"storage"`
	GraphAnalytics  string         `json:"graph_analytics"`
	VectorOptimised string         `json:"vector_optimized"`
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
	Name          string `json:"name"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Memory        string `json:"memory"`
	Version       string `json:"version,omitempty"`
}

// CreateInstanceData holds the response fields for a newly provisioned instance,
// including the database password. The String() method redacts the password for
// safe use in log output.
type CreateInstanceData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CloudProvider string `json:"cloud_provider"`
	ConnectionURL string `json:"connection_url"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

// String implements fmt.Stringer and redacts the Password field.
func (c CreateInstanceData) String() string {
	return fmt.Sprintf(
		"CreateInstanceData{ID:%s Name:%s CloudProvider:%s Region:%s Type:%s Username:%s Password:[redacted]}",
		c.ID, c.Name, c.CloudProvider, c.Region, c.Type, c.Username,
	)
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
type instanceService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
	client  *Client
}

// instanceBasePath constructs the URL path for instance operations.
// instanceID is optional — pass "" when targeting the collection.
func instanceBasePath(orgID, projectID, instanceID string) string {
	base := fmt.Sprintf("organizations/%s/projects/%s/instances", orgID, projectID)
	if instanceID != "" {
		return base + "/" + instanceID
	}
	return base
}

func (s *instanceService) resolveIDs(opts []CallOption) (orgID, projectID string) {
	var orgDefault, projectDefault string
	if s.client != nil {
		s.client.mu.RLock()
		orgDefault = s.client.defaultOrgID
		projectDefault = s.client.defaultProjectID
		s.client.mu.RUnlock()
	}
	cfg := applyOptions(opts)
	orgID = cfg.orgID
	if orgID == "" {
		orgID = orgDefault
	}
	projectID = cfg.projectID
	if projectID == "" {
		projectID = projectDefault
	}
	return
}

// List returns all instances within the resolved organization and project.
func (s *instanceService) List(ctx context.Context, opts ...CallOption) (*ListInstancesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveIDs(opts)

	if orgID == "" {
		err := errors.New("organization ID is required: provide it via WithOrg call option or WithOrganization client option")
		s.logger.ErrorContext(ctx, "missing organization ID", slog.String("error", err.Error()))
		return nil, err
	}
	if projectID == "" {
		err := errors.New("project ID is required: provide it via WithProject call option or WithDefaultProject client option")
		s.logger.ErrorContext(ctx, "missing project ID", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing instances", slog.String("orgID", orgID), slog.String("projectID", projectID))

	resp, err := s.api.Get(ctx, instanceBasePath(orgID, projectID, ""))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list instances", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("error", err.Error()))
		return nil, err
	}

	var result ListInstancesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal instances response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "instances listed successfully", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for a specific instance by UUID.
func (s *instanceService) Get(ctx context.Context, instanceID string, opts ...CallOption) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveIDs(opts)

	if orgID == "" {
		err := errors.New("organization ID is required: provide it via WithOrg call option or WithOrganization client option")
		s.logger.ErrorContext(ctx, "missing organization ID", slog.String("error", err.Error()))
		return nil, err
	}
	if projectID == "" {
		err := errors.New("project ID is required: provide it via WithProject call option or WithDefaultProject client option")
		s.logger.ErrorContext(ctx, "missing project ID", slog.String("error", err.Error()))
		return nil, err
	}

	if err := utils.ValidateTenantID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}

	s.logger.DebugContext(ctx, "getting instance details", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("instanceID", instanceID))

	resp, err := s.api.Get(ctx, instanceBasePath(orgID, projectID, instanceID))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "instance retrieved successfully", slog.String("instanceID", instanceID))
	return &result, nil
}

// Create provisions a new instance within the resolved organization and project.
func (s *instanceService) Create(ctx context.Context, req *CreateInstanceRequest, opts ...CallOption) (*CreateInstanceResponse, error) {
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

	orgID, projectID := s.resolveIDs(opts)

	if orgID == "" {
		err := errors.New("organization ID is required: provide it via WithOrg call option or WithOrganization client option")
		s.logger.ErrorContext(ctx, "missing organization ID", slog.String("error", err.Error()))
		return nil, err
	}
	if projectID == "" {
		err := errors.New("project ID is required: provide it via WithProject call option or WithDefaultProject client option")
		s.logger.ErrorContext(ctx, "missing project ID", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "creating instance", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("name", req.Name))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal create instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := s.api.Post(ctx, instanceBasePath(orgID, projectID, ""), string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create instance", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("error", err.Error()))
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

// Update modifies a specific instance's configuration.
func (s *instanceService) Update(ctx context.Context, instanceID string, req *UpdateInstanceRequest, opts ...CallOption) (*GetInstanceResponse, error) {
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

	orgID, projectID := s.resolveIDs(opts)

	if orgID == "" {
		err := errors.New("organization ID is required: provide it via WithOrg call option or WithOrganization client option")
		s.logger.ErrorContext(ctx, "missing organization ID", slog.String("error", err.Error()))
		return nil, err
	}
	if projectID == "" {
		err := errors.New("project ID is required: provide it via WithProject call option or WithDefaultProject client option")
		s.logger.ErrorContext(ctx, "missing project ID", slog.String("error", err.Error()))
		return nil, err
	}

	if err := utils.ValidateTenantID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}

	s.logger.InfoContext(ctx, "updating instance", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("instanceID", instanceID))

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal update instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := s.api.Patch(ctx, instanceBasePath(orgID, projectID, instanceID), string(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal update instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "instance updated successfully", slog.String("instanceID", instanceID))
	return &result, nil
}

// Delete removes a specific instance.
func (s *instanceService) Delete(ctx context.Context, instanceID string, opts ...CallOption) (*DeleteInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveIDs(opts)

	if orgID == "" {
		err := errors.New("organization ID is required: provide it via WithOrg call option or WithOrganization client option")
		s.logger.ErrorContext(ctx, "missing organization ID", slog.String("error", err.Error()))
		return nil, err
	}
	if projectID == "" {
		err := errors.New("project ID is required: provide it via WithProject call option or WithDefaultProject client option")
		s.logger.ErrorContext(ctx, "missing project ID", slog.String("error", err.Error()))
		return nil, err
	}

	if err := utils.ValidateTenantID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}

	s.logger.InfoContext(ctx, "deleting instance", slog.String("orgID", orgID), slog.String("projectID", projectID), slog.String("instanceID", instanceID))

	resp, err := s.api.Delete(ctx, instanceBasePath(orgID, projectID, instanceID))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	if len(resp.Body) == 0 {
		s.logger.InfoContext(ctx, "instance deleted successfully", slog.String("instanceID", instanceID))
		return &DeleteInstanceResponse{}, nil
	}

	var result DeleteInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal delete instance response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "instance deleted successfully", slog.String("instanceID", instanceID))
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
