package aura

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/v2/internal/api"
	"github.com/neo4j-contrib/aura-go-sdk/v2/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// InstanceStatus represents the lifecycle state of an Aura database instance.
type InstanceStatus string

// Instance status constants returned by the Aura API.
const (
	StatusRunning       InstanceStatus = "running"
	StatusStopped       InstanceStatus = "stopped"
	StatusPaused        InstanceStatus = "paused"
	StatusAvailable     InstanceStatus = "available"
	StatusCreating      InstanceStatus = "creating"
	StatusDestroying    InstanceStatus = "destroying"
	StatusPausing       InstanceStatus = "pausing"
	StatusSuspending    InstanceStatus = "suspending"
	StatusSuspended     InstanceStatus = "suspended"
	StatusResuming      InstanceStatus = "resuming"
	StatusLoading       InstanceStatus = "loading"
	StatusLoadingFailed InstanceStatus = "loading failed"
	StatusRestoring     InstanceStatus = "restoring"
	StatusUpdating      InstanceStatus = "updating"
	StatusOverwriting   InstanceStatus = "overwriting"

	// Deprecated: StatusRestroying was a misspelling. Use StatusRestoring.
	StatusRestroying = StatusRestoring
)

// ListInstancesResponse contains a list of instances in a tenant.
type ListInstancesResponse struct {
	Data []ListInstanceData `json:"data"`
}

// ListInstanceData holds the summary fields returned for each instance in a list response.
type ListInstanceData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Created       string `json:"created_at"`
	TenantID      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
}

// CreateInstanceConfigData holds the configuration required to provision a new instance.
type CreateInstanceConfigData struct {
	Name                 string `json:"name"`
	TenantID             string `json:"tenant_id"`
	CloudProvider        string `json:"cloud_provider"`
	Region               string `json:"region"`
	Type                 string `json:"type"`
	Version              string `json:"version"`
	Memory               string `json:"memory"`
	VectorOptimized      bool   `json:"vector_optimized,omitempty"`
	GraphAnalyticsPlugin bool   `json:"graph_analytics_plugin,omitempty"`
	CustomerManagedKeyID string `json:"customer_managed_key_id,omitempty"`
}

// createInstanceFromSourceRequest is the internal POST body used when cloning from a source.
type createInstanceFromSourceRequest struct {
	CreateInstanceConfigData
	SourceInstanceID string `json:"source_instance_id"`
	SourceSnapshotID string `json:"source_snapshot_id,omitempty"`
}

// CreateInstanceResponse wraps the response from a successful instance creation.
type CreateInstanceResponse struct {
	Data CreateInstanceData `json:"data"`
}

// CreateInstanceData holds the response fields for a newly provisioned instance.
// It contains the database password returned by the API — treat this value as a
// secret and avoid logging or serialising the struct directly. The String()
// method redacts the password for safe use in log output.
type CreateInstanceData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	TenantID      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
	ConnectionURL string `json:"connection_url"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

// String implements fmt.Stringer and redacts the Password field so that
// accidentally logging or printing this struct never exposes credentials.
func (c CreateInstanceData) String() string {
	return fmt.Sprintf(
		"CreateInstanceData{ID:%s Name:%s TenantID:%s CloudProvider:%s Region:%s Type:%s Username:%s Password:[redacted]}",
		c.ID, c.Name, c.TenantID, c.CloudProvider, c.Region, c.Type, c.Username,
	)
}

// UpdateInstanceData holds the fields that can be modified on an existing instance.
type UpdateInstanceData struct {
	Name              string `json:"name,omitempty"`
	Memory            string `json:"memory,omitempty"`
	CDCEnrichmentMode string `json:"cdc_enrichment_mode,omitempty"`
	SecondariesCount  int    `json:"secondaries_count,omitempty"`
}

// GetInstanceResponse wraps the response for a single instance lookup.
type GetInstanceResponse struct {
	Data InstanceData `json:"data"`
}

// DeleteInstanceResponse wraps the response returned when an instance is deleted.
type DeleteInstanceResponse struct {
	Data InstanceData `json:"data"`
}

// InstanceData holds the full set of fields returned for a single instance.
type InstanceData struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Status          InstanceStatus `json:"status"`
	TenantID        string         `json:"tenant_id"`
	CloudProvider   string         `json:"cloud_provider"`
	ConnectionURL   string         `json:"connection_url"`
	Region          string         `json:"region"`
	Type            string         `json:"type"`
	Memory          string         `json:"memory"`
	Storage         *string        `json:"storage"`
	CDCEnrichment   string         `json:"cdc_enrichment_mode"`
	GDSPlugin       bool           `json:"graph_analytics_plugin"`
	MetricsURL      string         `json:"metrics_integration_url"`
	Secondaries     int            `json:"secondaries_count"`
	VectorOptimized bool           `json:"vector_optimized"`
	CreatedAt       *time.Time     `json:"created_at,omitempty"`
}

type overwriteInstanceRequest struct {
	SourceInstanceID string `json:"source_instance_id,omitempty"`
	SourceSnapshotID string `json:"source_snapshot_id,omitempty"`
}

// OverwriteInstanceResponse wraps the job ID returned when an overwrite operation is started.
type OverwriteInstanceResponse struct {
	Data string `json:"data"`
}

// ============================================================================
// Service
// ============================================================================

// instanceService handles instance operations.
type instanceService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all instances accessible to the authenticated user.
func (i *instanceService) List(ctx context.Context) (*ListInstancesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "listing instances")

	resp, err := i.api.Get(ctx, "instances")
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to list instances", slog.String("error", err.Error()))
		return nil, err
	}

	var result ListInstancesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal instances response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.DebugContext(ctx, "instances listed successfully", slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for a specific instance by ID.
func (i *instanceService) Get(ctx context.Context, instanceID string) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "getting instance details", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Get(ctx, "instances/"+instanceID)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to get instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.DebugContext(ctx, "instance retrieved successfully", slog.String("instanceID", instanceID), slog.String("name", result.Data.Name), slog.Any("status", result.Data.Status))
	return &result, nil
}

// Create provisions a new database instance.
func (i *instanceService) Create(ctx context.Context, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	if instanceRequest == nil {
		err := errors.New("instanceRequest must not be nil")
		i.logger.ErrorContext(ctx, "instanceRequest must not be nil", slog.String("error", err.Error()))
		return nil, err
	}

	if err := validateCreateInstanceConfig(instanceRequest); err != nil {
		i.logger.ErrorContext(ctx, "failed to validate instance configuration", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.DebugContext(ctx, "creating instance", slog.String("name", instanceRequest.Name), slog.String("tenantID", instanceRequest.TenantID))

	body, err := json.Marshal(instanceRequest)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to marshal instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Post(ctx, "instances", string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to create instance", slog.String("name", instanceRequest.Name), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal create instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance created successfully", slog.String("instanceID", result.Data.ID), slog.String("name", result.Data.Name))
	return &result, nil
}

// Delete removes an instance by ID.
func (i *instanceService) Delete(ctx context.Context, instanceID string) (*DeleteInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "deleting instance", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Delete(ctx, "instances/"+instanceID)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to delete instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result DeleteInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal delete instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance deleted successfully", slog.String("instanceID", instanceID))
	return &result, nil
}

// Pause suspends an instance by ID.
func (i *instanceService) Pause(ctx context.Context, instanceID string) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "pausing instance", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Post(ctx, fmt.Sprintf("instances/%s/pause", instanceID), "")
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to pause instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal pause instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance paused successfully", slog.String("instanceID", instanceID))
	return &result, nil
}

// Resume restarts a paused instance by ID.
func (i *instanceService) Resume(ctx context.Context, instanceID string) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "resuming instance", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Post(ctx, fmt.Sprintf("instances/%s/resume", instanceID), "")
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to resume instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal resume instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance resumed successfully", slog.String("instanceID", instanceID))
	return &result, nil
}

// Update modifies an instance's configuration.
func (i *instanceService) Update(ctx context.Context, instanceID string, instanceRequest *UpdateInstanceData) (*GetInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "updating instance", slog.String("instanceID", instanceID))

	if instanceRequest == nil {
		err := errors.New("instanceRequest must not be nil")
		i.logger.ErrorContext(ctx, "instanceRequest must not be nil", slog.String("error", err.Error()))
		return nil, err
	}

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	body, err := json.Marshal(instanceRequest)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to marshal instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Patch(ctx, fmt.Sprintf("instances/%s", instanceID), string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to update instance", slog.String("instanceID", instanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal update instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance updated successfully", slog.String("instanceID", instanceID), slog.String("name", result.Data.Name))
	return &result, nil
}

// OverwriteFromInstance replaces instance data from another instance.
func (i *instanceService) OverwriteFromInstance(ctx context.Context, instanceID string, sourceInstanceID string) (*OverwriteInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "overwriting instance", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	if sourceInstanceID == "" {
		return nil, fmt.Errorf("must provide sourceInstanceID")
	}

	if err := utils.ValidateInstanceID(sourceInstanceID); err != nil {
		return nil, fmt.Errorf("invalid source instance ID: %w", err)
	}

	requestBody := overwriteInstanceRequest{
		SourceInstanceID: sourceInstanceID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to marshal instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Post(ctx, fmt.Sprintf("instances/%s/overwrite", instanceID), string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to overwrite instance from another instance", slog.String("instanceID", instanceID), slog.String("sourceInstanceID", sourceInstanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result OverwriteInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal overwrite instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance overwrite started", slog.String("instanceID", instanceID))
	return &result, nil
}

// OverwriteFromSnapshot replaces instance data from a snapshot.
func (i *instanceService) OverwriteFromSnapshot(ctx context.Context, instanceID string, sourceSnapshotID string) (*OverwriteInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	i.logger.DebugContext(ctx, "overwriting instance", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		i.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	if sourceSnapshotID == "" {
		return nil, fmt.Errorf("must provide sourceSnapshotID")
	}

	if err := utils.ValidateSnapshotID(sourceSnapshotID); err != nil {
		return nil, fmt.Errorf("invalid source snapshot ID: %w", err)
	}

	requestBody := overwriteInstanceRequest{
		SourceSnapshotID: sourceSnapshotID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to marshal instance request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := i.api.Post(ctx, fmt.Sprintf("instances/%s/overwrite", instanceID), string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to overwrite instance with a snapshot", slog.String("instanceID", instanceID), slog.String("snapshotID", sourceSnapshotID), slog.String("error", err.Error()))
		return nil, err
	}

	var result OverwriteInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		i.logger.ErrorContext(ctx, "failed to unmarshal overwrite instance response", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance overwrite started", slog.String("instanceID", instanceID))
	return &result, nil
}

// CreateFromInstance provisions a new instance cloned from an existing source instance.
func (i *instanceService) CreateFromInstance(ctx context.Context, sourceInstanceID string, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	if instanceRequest == nil {
		return nil, errors.New("instanceRequest must not be nil")
	}
	if sourceInstanceID == "" {
		return nil, fmt.Errorf("must provide sourceInstanceID")
	}
	if err := utils.ValidateInstanceID(sourceInstanceID); err != nil {
		return nil, fmt.Errorf("invalid source instance ID: %w", err)
	}
	if err := validateCreateInstanceConfig(instanceRequest); err != nil {
		i.logger.ErrorContext(ctx, "failed to validate instance configuration", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.DebugContext(ctx, "creating instance from source instance", slog.String("name", instanceRequest.Name), slog.String("sourceInstanceID", sourceInstanceID))

	body, err := json.Marshal(createInstanceFromSourceRequest{
		CreateInstanceConfigData: *instanceRequest,
		SourceInstanceID:         sourceInstanceID,
	})
	if err != nil {
		return nil, err
	}

	resp, err := i.api.Post(ctx, "instances", string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to create instance from source instance", slog.String("sourceInstanceID", sourceInstanceID), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance created from source instance", slog.String("instanceID", result.Data.ID), slog.String("sourceInstanceID", sourceInstanceID))
	return &result, nil
}

// CreateFromSnapshot provisions a new instance cloned from a specific snapshot.
// Both sourceInstanceID and sourceSnapshotID are required by the API; the snapshot
// must belong to the source instance and must be exportable.
func (i *instanceService) CreateFromSnapshot(ctx context.Context, sourceInstanceID string, sourceSnapshotID string, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	if instanceRequest == nil {
		return nil, errors.New("instanceRequest must not be nil")
	}
	if sourceInstanceID == "" {
		return nil, fmt.Errorf("must provide sourceInstanceID")
	}
	if err := utils.ValidateInstanceID(sourceInstanceID); err != nil {
		return nil, fmt.Errorf("invalid source instance ID: %w", err)
	}
	if sourceSnapshotID == "" {
		return nil, fmt.Errorf("must provide sourceSnapshotID")
	}
	if err := utils.ValidateSnapshotID(sourceSnapshotID); err != nil {
		return nil, fmt.Errorf("invalid source snapshot ID: %w", err)
	}
	if err := validateCreateInstanceConfig(instanceRequest); err != nil {
		i.logger.ErrorContext(ctx, "failed to validate instance configuration", slog.String("error", err.Error()))
		return nil, err
	}

	i.logger.DebugContext(ctx, "creating instance from snapshot", slog.String("name", instanceRequest.Name), slog.String("sourceInstanceID", sourceInstanceID), slog.String("sourceSnapshotID", sourceSnapshotID))

	body, err := json.Marshal(createInstanceFromSourceRequest{
		CreateInstanceConfigData: *instanceRequest,
		SourceInstanceID:         sourceInstanceID,
		SourceSnapshotID:         sourceSnapshotID,
	})
	if err != nil {
		return nil, err
	}

	resp, err := i.api.Post(ctx, "instances", string(body))
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to create instance from snapshot", slog.String("sourceInstanceID", sourceInstanceID), slog.String("sourceSnapshotID", sourceSnapshotID), slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateInstanceResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}

	i.logger.InfoContext(ctx, "instance created from snapshot", slog.String("instanceID", result.Data.ID), slog.String("sourceInstanceID", sourceInstanceID), slog.String("sourceSnapshotID", sourceSnapshotID))
	return &result, nil
}

// validateCreateInstanceConfig performs basic checks that the minimum number
// of configuration options have been supplied when creating an instance.
func validateCreateInstanceConfig(instanceConfig *CreateInstanceConfigData) error {
	if instanceConfig.Name == "" {
		return fmt.Errorf("instance name must not be empty")
	}
	if len(instanceConfig.Name) > 30 {
		return fmt.Errorf("instance name must be less than 30 characters long")
	}
	if instanceConfig.TenantID == "" {
		return fmt.Errorf("tenant ID must not be empty")
	}
	if err := utils.ValidateTenantID(instanceConfig.TenantID); err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}
	if instanceConfig.CloudProvider == "" {
		return fmt.Errorf("cloud provider must not be empty")
	}
	if instanceConfig.Region == "" {
		return fmt.Errorf("region must not be empty")
	}
	if instanceConfig.Type == "" {
		return fmt.Errorf("instance type must not be empty")
	}
	if instanceConfig.Version == "" {
		return fmt.Errorf("version must not be empty")
	}
	if instanceConfig.Memory == "" {
		return fmt.Errorf("memory must not be empty")
	}
	return nil
}
