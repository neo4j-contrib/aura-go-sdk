package aura

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
	"github.com/LackOfMorals/aura-client/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// GetCmeksResponse contains a list of customer managed encryption keys.
type GetCmeksResponse struct {
	Data []GetCmeksData `json:"data"`
}

// GetCmeksData holds the fields for a single customer-managed encryption key entry.
type GetCmeksData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TenantID string `json:"tenant_id"`
}

// ============================================================================
// Service
// ============================================================================

// cmekService handles customer managed encryption key operations.
type cmekService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns all customer-managed encryption keys, optionally filtered by tenant.
func (c *cmekService) List(ctx context.Context, tenantID string) (*GetCmeksResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.logger.DebugContext(ctx, "listing customer managed keys")

	endpoint := "customer-managed-keys"
	if tenantID != "" {
		if err := utils.ValidateTenantID(tenantID); err != nil {
			return nil, err
		}
		endpoint += "?tenant_id=" + tenantID
	}

	resp, err := c.api.Get(ctx, endpoint)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to obtain customer managed keys", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetCmeksResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		c.logger.ErrorContext(ctx, "failed to unmarshal cmek response", slog.String("error", err.Error()))
		return nil, err
	}

	c.logger.DebugContext(ctx, "obtained customer managed keys", slog.Int("count", len(result.Data)))
	return &result, nil
}
