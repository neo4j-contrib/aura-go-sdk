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

// BackupStatus represents the status of a database backup.
type BackupStatus string

const (
	BackupStatusInProgress BackupStatus = "InProgress"
	BackupStatusFailed     BackupStatus = "Failed"
	BackupStatusCompleted  BackupStatus = "Completed"
	BackupStatusPending    BackupStatus = "Pending"
)

// DatabaseBackup represents a single backup of an Aura database.
type DatabaseBackup struct {
	ID         string       `json:"id"`
	Timestamp  string       `json:"timestamp"`
	Status     BackupStatus `json:"status"`
	Exportable bool         `json:"exportable"`
}

// ListBackupsResponse wraps the list of database backups returned by the API.
type ListBackupsResponse struct {
	Data []DatabaseBackup `json:"data"`
}

// CreateBackupResponse wraps the single database backup returned by the API
// after a backup is triggered.
type CreateBackupResponse struct {
	Data DatabaseBackup `json:"data"`
}

// GetDatabase wraps the list of databases  returned by the API.
type GetDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// ListDatabases wraps the list of databases  returned by the API.
type ListDatabasesResponse struct {
	Data []DatabaseResponse `json:"data"`
}

// DatabaseResponse represents a single Aura database.
type DatabaseResponse struct {
	ID string `json:"id"`
}

// ============================================================================
// Service
// ============================================================================

// databaseService handles database backup operations for the v2beta1 API.
type databaseService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
	client  *Client
}

func backupsPath(orgID, projectID, instanceID, databaseID string) string {
	return fmt.Sprintf(
		"organizations/%s/projects/%s/instances/%s/databases/%s/backups",
		orgID, projectID, instanceID, databaseID,
	)
}

func instancePath(orgID, projectID, instanceID string) string {
	return fmt.Sprintf(
		"organizations/%s/projects/%s/instances/%s/databases",
		orgID, projectID, instanceID,
	)
}

// resolveOrgProject reads both defaults from the client under a single lock and
// applies call options once, returning the effective org and project IDs.
func (s *databaseService) resolveOrgProject(opts []CallOption) (orgID, projectID string) {
	var defaultOrg, defaultProject string
	if s.client != nil {
		s.client.mu.RLock()
		defaultOrg = s.client.defaultOrgID
		defaultProject = s.client.defaultProjectID
		s.client.mu.RUnlock()
	}
	cfg := applyOptions(opts)
	orgID = cfg.orgID
	if orgID == "" {
		orgID = defaultOrg
	}
	projectID = cfg.projectID
	if projectID == "" {
		projectID = defaultProject
	}
	return
}

func (s *databaseService) ListDatabases(ctx context.Context, instanceID string, opts ...CallOption) (*ListDatabasesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

	if err := utils.ValidateOrgID(orgID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	if err := utils.ValidateProjectID(projectID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		s.logger.ErrorContext(ctx, "invalid instance ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	s.logger.DebugContext(ctx, "listing databases",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
	)

	path := instancePath(orgID, projectID, instanceID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list instance backups",
			slog.String("instanceID", instanceID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result ListDatabasesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal list backups response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database backups listed successfully",
		slog.String("instanceID", instanceID),
		slog.Int("count", len(result.Data)),
	)
	return &result, nil
}

func (s *databaseService) ListBackups(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*ListBackupsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

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
	if err := utils.ValidateTenantID(databaseID); err != nil {
		s.logger.ErrorContext(ctx, "invalid database ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	s.logger.DebugContext(ctx, "listing database backups",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)

	path := backupsPath(orgID, projectID, instanceID, databaseID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list database backups",
			slog.String("instanceID", instanceID),
			slog.String("databaseID", databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result ListBackupsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal list backups response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database backups listed successfully",
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
		slog.Int("count", len(result.Data)),
	)
	return &result, nil
}

func (s *databaseService) CreateBackup(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*CreateBackupResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

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
	if err := utils.ValidateTenantID(databaseID); err != nil {
		s.logger.ErrorContext(ctx, "invalid database ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	s.logger.DebugContext(ctx, "creating database backup",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)

	path := backupsPath(orgID, projectID, instanceID, databaseID)
	resp, err := s.api.Post(ctx, path, "")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create database backup",
			slog.String("instanceID", instanceID),
			slog.String("databaseID", databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result CreateBackupResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create backup response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "database backup created successfully",
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
		slog.String("backupID", result.Data.ID),
	)
	return &result, nil
}
