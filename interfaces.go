// Package aura provides a Go client library for the Neo4j Aura API.
package aura

import "context"

// TenantService defines operations for managing tenants
type TenantService interface {
	// List returns all tenants accessible to the authenticated user
	List(ctx context.Context) (*ListTenantsResponse, error)
	// Get retrieves details for a specific tenant by ID
	Get(ctx context.Context, tenantID string) (*GetTenantResponse, error)
	// GetMetrics gets URL for project level Prometheus metrics
	GetMetrics(ctx context.Context, tenantID string) (*GetTenantMetricsURLResponse, error)
}

// InstanceService defines operations for managing database instances
type InstanceService interface {
	// List returns all instances accessible to the authenticated user
	List(ctx context.Context) (*ListInstancesResponse, error)
	// Get retrieves details for a specific instance by ID
	Get(ctx context.Context, instanceID string) (*GetInstanceResponse, error)
	// Create provisions a new database instance
	Create(ctx context.Context, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error)
	// CreateFromInstance provisions a new instance cloned from an existing source instance
	CreateFromInstance(ctx context.Context, sourceInstanceID string, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error)
	// CreateFromSnapshot provisions a new instance cloned from a specific snapshot.
	// Both sourceInstanceID and sourceSnapshotID are required; the snapshot must belong
	// to the source instance and must be exportable.
	CreateFromSnapshot(ctx context.Context, sourceInstanceID string, sourceSnapshotID string, instanceRequest *CreateInstanceConfigData) (*CreateInstanceResponse, error)
	// Delete removes an instance by ID
	Delete(ctx context.Context, instanceID string) (*DeleteInstanceResponse, error)
	// Pause suspends an instance by ID
	Pause(ctx context.Context, instanceID string) (*GetInstanceResponse, error)
	// Resume restarts a paused instance by ID
	Resume(ctx context.Context, instanceID string) (*GetInstanceResponse, error)
	// Update modifies an instance's configuration
	Update(ctx context.Context, instanceID string, instanceRequest *UpdateInstanceData) (*GetInstanceResponse, error)
	// Overwrite replaces instance data from another instance or snapshot
	OverwriteFromInstance(ctx context.Context, instanceID string, sourceInstanceID string) (*OverwriteInstanceResponse, error)
	// Overwrite replaces instance data from another instance or snapshot
	OverwriteFromSnapshot(ctx context.Context, instanceID string, sourceSnapshotID string) (*OverwriteInstanceResponse, error)
}

// SnapshotService defines operations for managing instance snapshots
type SnapshotService interface {
	// List returns snapshots for an instance, optionally filtered by date (YYYY-MM-DD)
	List(ctx context.Context, instanceID string, snapshotDate *SnapshotDate) (*GetSnapshotsResponse, error)
	// Create triggers an on-demand snapshot for an instance
	Create(ctx context.Context, instanceID string) (*CreateSnapshotResponse, error)
	// Get returns details for a snapshot of an instance
	Get(ctx context.Context, instanceID string, snapshotID string) (*GetSnapshotDataResponse, error)
	// Restore instance from a snapshot
	Restore(ctx context.Context, instanceID string, snapshotID string) (*RestoreSnapshotResponse, error)
}

// CMEKService defines operations for customer-managed encryption keys
type CMEKService interface {
	// List returns all customer-managed encryption keys, optionally filtered by tenant
	List(ctx context.Context, tenantID string) (*GetCMEKsResponse, error)
}

// GDSSessionService defines operations for Graph Data Science sessions
type GDSSessionService interface {
	// List returns all GDS sessions accessible to the authenticated user
	List(ctx context.Context) (*GetGDSSessionListResponse, error)
	// Estimate the size of a GDS session
	Estimate(ctx context.Context, GDSSessionSizeEstimateRequest *GetGDSSessionSizeEstimation) (*GDSSessionSizeEstimationResponse, error)
	// Create a new GDS session
	Create(ctx context.Context, GDSSessionConfigRequest *CreateGDSSessionConfigData) (*GetGDSSessionResponse, error)
	// Get the details for a single GDS Session
	Get(ctx context.Context, GDSSessionID string) (*GetGDSSessionResponse, error)
	// Delete a single GDS Session
	Delete(ctx context.Context, GDSSessionID string) (*DeleteGDSSessionResponse, error)
}

// PrometheusService defines operations for querying Prometheus metrics
type PrometheusService interface {
	// FetchRawMetrics fetches and parses raw Prometheus metrics from an Aura metrics endpoint
	FetchRawMetrics(ctx context.Context, prometheusURL string) (*PrometheusMetricsResponse, error)
	// GetMetricValue retrieves a specific metric value by name and optional label filters
	GetMetricValue(ctx context.Context, metrics *PrometheusMetricsResponse, name string, labelFilters map[string]string) (float64, error)
	// GetInstanceHealth retrieves comprehensive health metrics for an instance
	GetInstanceHealth(ctx context.Context, instanceID string, prometheusURL string) (*PrometheusHealthMetrics, error)
}

// Compile-time interface compliance checks
var (
	_ TenantService     = (*tenantService)(nil)
	_ InstanceService   = (*instanceService)(nil)
	_ SnapshotService   = (*snapshotService)(nil)
	_ CMEKService       = (*cmekService)(nil)
	_ GDSSessionService = (*gdsSessionService)(nil)
	_ PrometheusService = (*prometheusService)(nil)
)
