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

// ============================================================================
// Service
// ============================================================================

// databaseService handles database backup operations for the v2beta1 API.
type databaseBackupService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
	client  *Client
}

// resolveOrgProject reads both defaults from the client under a single lock and
// applies call options once, returning the effective org and project IDs.
func (s *databaseBackupService) resolveOrgProject(opts []CallOption) (orgID, projectID string) {
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

func (s *databaseBackupService) List(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*ListBackupsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

	// Check IDs are supplied and valid
	// Using new Validate function
	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.InstanceID(instanceID),
		utils.DatabaseID(databaseID),
	); err != nil {
		return nil, err
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

func (s *databaseBackupService) Create(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*CreateBackupResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

	// Check IDs are supplied and valid
	// Using new Validate function
	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.InstanceID(instanceID),
		utils.DatabaseID(databaseID),
	); err != nil {
		return nil, err
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
