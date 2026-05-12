package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
	"github.com/LackOfMorals/aura-client/internal/utils"
)

// ============================================================================
// Types
// ============================================================================

// GetSnapshotsResponse contains a list of snapshots for an instance.
type GetSnapshotsResponse struct {
	Data []GetSnapshotData `json:"data"`
}

// GetSnapshotDataResponse wraps the response for a single snapshot lookup.
type GetSnapshotDataResponse struct {
	Data GetSnapshotData `json:"data"`
}

// GetSnapshotData holds the fields returned for a single snapshot.
type GetSnapshotData struct {
	InstanceID string    `json:"instance_id"`
	SnapshotID string    `json:"snapshot_id"`
	Profile    string    `json:"profile"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	Exportable bool      `json:"exportable"`
}

// UnmarshalJSON implements json.Unmarshaler for GetSnapshotData. It parses the
// Timestamp field from the RFC3339 string format returned by the Aura API into
// a time.Time value. An empty timestamp string is silently ignored and leaves
// the field at its zero value.
func (s *GetSnapshotData) UnmarshalJSON(data []byte) error {
	var raw struct {
		InstanceID string `json:"instance_id"`
		SnapshotID string `json:"snapshot_id"`
		Profile    string `json:"profile"`
		Status     string `json:"status"`
		Timestamp  string `json:"timestamp"`
		Exportable bool   `json:"exportable"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.InstanceID = raw.InstanceID
	s.SnapshotID = raw.SnapshotID
	s.Profile = raw.Profile
	s.Status = raw.Status
	s.Exportable = raw.Exportable
	if raw.Timestamp != "" {
		t, err := time.Parse(time.RFC3339Nano, raw.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid snapshot timestamp %q: %w", raw.Timestamp, err)
		}
		s.Timestamp = t
	}
	return nil
}

// CreateSnapshotResponse contains the result of creating a snapshot.
type CreateSnapshotResponse struct {
	Data CreateSnapshotData `json:"data"`
}

// CreateSnapshotData holds the snapshot ID returned after a snapshot is created.
type CreateSnapshotData struct {
	SnapshotID string `json:"snapshot_id"`
}

// RestoreSnapshotResponse stores the response from initiating restoration of
// an instance using a snapshot. The response is the same as for getting
// instance configuration details.
type RestoreSnapshotResponse struct {
	Data InstanceData `json:"data"`
}

// SnapshotDate is used as an optional filter when listing an instance's snapshots.
type SnapshotDate struct {
	Year  int
	Month time.Month
	Day   int
}

// Today returns today's date as a *SnapshotDate for use as a snapshot list filter.
func Today() *SnapshotDate {
	y, m, d := time.Now().Date()
	return &SnapshotDate{y, m, d}
}

// ============================================================================
// Service
// ============================================================================

// snapshotService handles snapshot operations.
type snapshotService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List returns snapshots for an instance, optionally filtered by date (YYYY-MM-DD).
func (s *snapshotService) List(ctx context.Context, instanceID string, snapshotDate *SnapshotDate) (*GetSnapshotsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	s.logger.DebugContext(ctx, "listing snapshots", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	endpoint := fmt.Sprintf("instances/%s/snapshots", instanceID)
	if snapshotDate != nil {
		endpoint += fmt.Sprintf("?date=%04d-%02d-%02d", snapshotDate.Year, int(snapshotDate.Month), snapshotDate.Day)
		s.logger.DebugContext(ctx, "listing snapshots with date filter", slog.String("url", endpoint))
	}

	resp, err := s.api.Get(ctx, endpoint)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list snapshots", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetSnapshotsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal snapshots response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "snapshots listed successfully", slog.Int("count", len(result.Data)))
	return &result, nil
}

// Get returns the details for a snapshot of an instance.
func (s *snapshotService) Get(ctx context.Context, instanceID string, snapshotID string) (*GetSnapshotDataResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}
	if err := utils.ValidateSnapshotID(snapshotID); err != nil {
		s.logger.ErrorContext(ctx, "invalid snapshot ID", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "get snapshot details", slog.String("snapshotID", snapshotID), slog.String("instanceID", instanceID))

	resp, err := s.api.Get(ctx, fmt.Sprintf("instances/%s/snapshots/%s", instanceID, snapshotID))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get snapshot", slog.String("error", err.Error()))
		return nil, err
	}

	var result GetSnapshotDataResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal snapshots response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "snapshot details obtained")
	return &result, nil
}

// Create triggers an on-demand snapshot for an instance.
func (s *snapshotService) Create(ctx context.Context, instanceID string) (*CreateSnapshotResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "creating snapshot", slog.String("instanceID", instanceID))

	resp, err := s.api.Post(ctx, fmt.Sprintf("instances/%s/snapshots", instanceID), "")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create snapshot", slog.String("error", err.Error()))
		return nil, err
	}

	var result CreateSnapshotResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal snapshot response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "snapshot created", slog.String("snapshotId", result.Data.SnapshotID))
	return &result, nil
}

// Restore restores an instance from a snapshot.
func (s *snapshotService) Restore(ctx context.Context, instanceID string, snapshotID string) (*RestoreSnapshotResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, err
	}
	if err := utils.ValidateSnapshotID(snapshotID); err != nil {
		s.logger.ErrorContext(ctx, "invalid snapshot ID", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "restore instance with a snapshot", slog.String("snapshotID", snapshotID), slog.String("instanceID", instanceID))

	resp, err := s.api.Post(ctx, fmt.Sprintf("instances/%s/snapshots/%s/restore", instanceID, snapshotID), "")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to restore using snapshot", slog.String("error", err.Error()))
		return nil, err
	}

	var result RestoreSnapshotResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal snapshots restore response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "snapshot restore started")
	return &result, nil
}
