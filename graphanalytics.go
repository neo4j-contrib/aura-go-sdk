package aura

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
)

// ============================================================================
// Types
// ============================================================================

// GetGDSSessionListResponse contains a list of GDS sessions.
type GetGDSSessionListResponse struct {
	Data []GetGDSSessionData `json:"data"`
}

// GetGDSSessionResponse contains information about a single GDS session.
type GetGDSSessionResponse struct {
	Data GetGDSSessionData `json:"data"`
}

// GetGDSSessionData holds the fields returned for a single GDS session.
type GetGDSSessionData struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Memory        string    `json:"memory"`
	InstanceID    string    `json:"instance_id"`
	DatabaseID    string    `json:"database_uuid"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	Host          string    `json:"host"`
	ExpiresAt     time.Time `json:"expiry_date"`
	TTL           string    `json:"ttl"`
	UserID        string    `json:"user_id"`
	TenantID      string    `json:"tenant_id"`
	CloudProvider string    `json:"cloud_provider"`
	Region        string    `json:"region"`
}

// UnmarshalJSON implements json.Unmarshaler for GetGDSSessionData. It parses
// the CreatedAt and ExpiresAt fields from the RFC3339 string format returned
// by the Aura API into time.Time values. Empty timestamp strings are silently
// ignored and leave the field at its zero value.
func (g *GetGDSSessionData) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Memory        string `json:"memory"`
		InstanceID    string `json:"instance_id"`
		DatabaseID    string `json:"database_uuid"`
		Status        string `json:"status"`
		CreatedAt     string `json:"created_at"`
		Host          string `json:"host"`
		ExpiresAt     string `json:"expiry_date"`
		TTL           string `json:"ttl"`
		UserID        string `json:"user_id"`
		TenantID      string `json:"tenant_id"`
		CloudProvider string `json:"cloud_provider"`
		Region        string `json:"region"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	g.ID = raw.ID
	g.Name = raw.Name
	g.Memory = raw.Memory
	g.InstanceID = raw.InstanceID
	g.DatabaseID = raw.DatabaseID
	g.Status = raw.Status
	g.Host = raw.Host
	g.TTL = raw.TTL
	g.UserID = raw.UserID
	g.TenantID = raw.TenantID
	g.CloudProvider = raw.CloudProvider
	g.Region = raw.Region
	if raw.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, raw.CreatedAt)
		if err != nil {
			return fmt.Errorf("invalid GDS session created_at %q: %w", raw.CreatedAt, err)
		}
		g.CreatedAt = t
	}
	if raw.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339Nano, raw.ExpiresAt)
		if err != nil {
			return fmt.Errorf("invalid GDS session expiry_date %q: %w", raw.ExpiresAt, err)
		}
		g.ExpiresAt = t
	}
	return nil
}

// CreateGDSSessionConfigData holds the configuration required to create a new GDS session.
type CreateGDSSessionConfigData struct {
	Name          string `json:"name"`
	TTL           string `json:"ttl"`
	TenantID      string `json:"tenant_id"`
	InstanceID    string `json:"instance_id"`
	DatabaseID    string `json:"database_uuid"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Memory        string `json:"memory"`
}

// GetGDSSessionSizeEstimation holds graph statistics used to estimate the memory
// requirements for a new GDS session.
type GetGDSSessionSizeEstimation struct {
	NodeCount                 int      `json:"node_count"`
	NodePropertyCount         int      `json:"node_property_count"`
	NodeLabelCount            int      `json:"node_label_count"`
	RelationshipCount         int      `json:"relationship_count"`
	RelationshipPropertyCount int      `json:"relationship_property_count"`
	AlgorithmCategories       []string `json:"algorithm_categories"`
}

// GDSSessionSizeEstimationResponse wraps the size estimation result.
type GDSSessionSizeEstimationResponse struct {
	Data GDSSessionSizeEstimationData `json:"data"`
}

// GDSSessionSizeEstimationData holds the estimated memory and recommended size tier.
type GDSSessionSizeEstimationData struct {
	EstimatedMemory string `json:"estimated_memory"`
	RecommendedSize string `json:"recommended_size"`
}

// DeleteGDSSessionResponse wraps the response returned when a GDS session is deleted.
type DeleteGDSSessionResponse struct {
	Data DeleteGDSSession `json:"data"`
}

// DeleteGDSSession holds the ID of the deleted session.
type DeleteGDSSession struct {
	ID string `json:"id"`
}

// ============================================================================
// Service
// ============================================================================

// gdsSessionService handles Graph Data Science session operations.
type gdsSessionService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all GDS sessions accessible to the authenticated user.
func (g *gdsSessionService) List(ctx context.Context) (*GetGDSSessionListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	g.logger.DebugContext(ctx, "listing GDS sessions")

	resp, err := g.api.Get(ctx, "graph-analytics/sessions")
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to list GDS sessions", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetGDSSessionListResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		g.logger.ErrorContext(ctx, "failed to unmarshal GDS sessions response", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.DebugContext(ctx, "GDS sessions listed successfully", slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get returns information on a single GDS session.
func (g *gdsSessionService) Get(ctx context.Context, gdsSessionID string) (*GetGDSSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	if gdsSessionID == "" {
		return nil, errors.New("GDS session ID must not be empty")
	}

	g.logger.DebugContext(ctx, "getting GDS session", slog.String("sessionID", gdsSessionID))

	resp, err := g.api.Get(ctx, fmt.Sprintf("graph-analytics/sessions/%s", gdsSessionID))
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to get GDS session", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetGDSSessionResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		g.logger.ErrorContext(ctx, "failed to unmarshal GDS session response", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.DebugContext(ctx, "GDS session obtained successfully")
	return &result, nil
}

// Create creates a new GDS session.
func (g *gdsSessionService) Create(ctx context.Context, gdsSessionConfigRequest *CreateGDSSessionConfigData) (*GetGDSSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	if gdsSessionConfigRequest == nil {
		return nil, errors.New("gdsSessionConfigRequest must not be nil")
	}

	g.logger.DebugContext(ctx, "creating GDS session")

	body, err := json.Marshal(gdsSessionConfigRequest)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to marshal create gds session request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := g.api.Post(ctx, "graph-analytics/sessions", string(body))
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to create GDS session", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetGDSSessionResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		g.logger.ErrorContext(ctx, "failed to unmarshal create GDS sessions response", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.DebugContext(ctx, "GDS session created successfully")
	return &result, nil
}

// Estimate estimates the size of a new GDS session.
func (g *gdsSessionService) Estimate(ctx context.Context, gdsSessionSizeEstimateRequest *GetGDSSessionSizeEstimation) (*GDSSessionSizeEstimationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	if gdsSessionSizeEstimateRequest == nil {
		return nil, errors.New("gdsSessionSizeEstimateRequest must not be nil")
	}

	g.logger.DebugContext(ctx, "estimating GDS session")

	body, err := json.Marshal(gdsSessionSizeEstimateRequest)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to marshal estimate gds session request", slog.String("error", err.Error()))
		return nil, err
	}

	resp, err := g.api.Post(ctx, "graph-analytics/sessions/sizing", string(body))
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to estimate GDS session", slog.String("error", err.Error()))
		return nil, err
	}

	var result GDSSessionSizeEstimationResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		g.logger.ErrorContext(ctx, "failed to unmarshal estimate GDS sessions response", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.DebugContext(ctx, "GDS session estimated successfully")
	return &result, nil
}

// Delete deletes a GDS session.
func (g *gdsSessionService) Delete(ctx context.Context, gdsSessionID string) (*DeleteGDSSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	if gdsSessionID == "" {
		return nil, errors.New("GDS session ID must not be empty")
	}

	g.logger.DebugContext(ctx, "deleting a GDS session", slog.String("sessionID", gdsSessionID))

	resp, err := g.api.Delete(ctx, fmt.Sprintf("graph-analytics/sessions/%s", gdsSessionID))
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to delete a GDS session", slog.String("error", err.Error()))
		return nil, err
	}

	var result DeleteGDSSessionResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		g.logger.ErrorContext(ctx, "failed to unmarshal GDS session delete response", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.DebugContext(ctx, "GDS session deleted successfully")
	return &result, nil
}
