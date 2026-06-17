package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
	"github.com/neo4j-contrib/aura-go-sdk/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// ListTenantsResponse contains a list of tenants in your organisation.
type ListTenantsResponse struct {
	Data []TenantsResponseData `json:"data"`
}

// TenantsResponseData holds the summary fields for a single tenant in a list response.
type TenantsResponseData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetTenantResponse contains details of a tenant.
type GetTenantResponse struct {
	Data TenantResponseData `json:"data"`
}

// TenantResponseData holds the full details returned for a single tenant.
type TenantResponseData struct {
	ID                     string                        `json:"id"`
	Name                   string                        `json:"name"`
	InstanceConfigurations []TenantInstanceConfiguration `json:"instance_configurations"`
}

// TenantInstanceConfiguration describes one available instance configuration for a tenant.
type TenantInstanceConfiguration struct {
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	RegionName    string `json:"region_name"`
	Type          string `json:"type"`
	Memory        string `json:"memory"`
	Storage       string `json:"storage"`
	Version       string `json:"version"`
}

// GetTenantMetricsURLResponse wraps the Prometheus metrics endpoint URL for a tenant.
type GetTenantMetricsURLResponse struct {
	Data GetTenantMetricsURLData `json:"data"`
}

// GetTenantMetricsURLData holds the Prometheus endpoint URL.
type GetTenantMetricsURLData struct {
	Endpoint string `json:"endpoint"`
}

// ============================================================================
// Service
// ============================================================================

// tenantService handles tenant operations.
type tenantService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all tenants accessible to the authenticated user.
func (t *tenantService) List(ctx context.Context) (*ListTenantsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	t.logger.DebugContext(ctx, "listing tenants")

	resp, err := t.api.Get(ctx, "tenants")
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to list tenants", slog.String("error", err.Error()))
		return nil, err
	}

	var result ListTenantsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.logger.ErrorContext(ctx, "failed to unmarshal tenants response", slog.String("error", err.Error()))
		return nil, err
	}

	t.logger.DebugContext(ctx, "tenants listed successfully", slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get retrieves details for a specific tenant by ID.
func (t *tenantService) Get(ctx context.Context, tenantID string) (*GetTenantResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	if err := utils.ValidateTenantID(tenantID); err != nil {
		t.logger.ErrorContext(ctx, "invalid tenant ID", slog.String("error", err.Error()))
		return nil, err
	}

	t.logger.DebugContext(ctx, "getting tenant details", slog.String("tenantID", tenantID))

	resp, err := t.api.Get(ctx, fmt.Sprintf("tenants/%s", tenantID))
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get tenant details", slog.String("tenantID", tenantID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetTenantResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.logger.ErrorContext(ctx, "failed to unmarshal tenant response", slog.String("error", err.Error()))
		return nil, err
	}

	t.logger.DebugContext(ctx, "tenant obtained successfully", slog.String("name", result.Data.Name))
	return &result, nil
}

// GetMetrics retrieves the Prometheus metrics URL for a specific tenant.
func (t *tenantService) GetMetrics(ctx context.Context, tenantID string) (*GetTenantMetricsURLResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	if err := utils.ValidateTenantID(tenantID); err != nil {
		t.logger.ErrorContext(ctx, "invalid tenant ID", slog.String("error", err.Error()))
		return nil, err
	}

	t.logger.DebugContext(ctx, "getting tenant prometheus metrics url", slog.String("tenantID", tenantID))

	resp, err := t.api.Get(ctx, fmt.Sprintf("tenants/%s/metrics-integration", tenantID))
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get tenant metrics url", slog.String("tenantID", tenantID), slog.String("error", err.Error()))
		return nil, err
	}

	var result GetTenantMetricsURLResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.logger.ErrorContext(ctx, "failed to unmarshal tenant metrics url response", slog.String("error", err.Error()))
		return nil, err
	}

	t.logger.DebugContext(ctx, "tenant metrics url obtained successfully", slog.String("endpoint", result.Data.Endpoint))
	return &result, nil
}
