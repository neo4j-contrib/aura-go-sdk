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

// GetBackupResponse wraps database backup returned by the API.
type GetBackupResponse struct {
	Data DatabaseBackup `json:"data"`
}

// CreateBackupResponse wraps the single database backup returned by the API
// after a backup is triggered.
type CreateBackupResponse struct {
	Data DatabaseBackup `json:"data"`
}

// ============================================================================
// Service
// ============================================================================

// databaseBackupService handles database backup operations for the v2beta1 API.
type databaseBackupService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// List all of the backups of a database.
func (s *databaseBackupService) List(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*ListBackupsResponse, error) {
	ctx, cancelBackupList := context.WithTimeout(ctx, s.timeout)
	defer cancelBackupList()

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

	path := utils.BackupsPath(orgID, projectID, instanceID, databaseID)

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

// Create a database backup. This may take several minutes to complete.
func (s *databaseBackupService) Create(ctx context.Context, orgID, projectID, instanceID, databaseID string) (*CreateBackupResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

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

	path := utils.BackupsPath(orgID, projectID, instanceID, databaseID)

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

// Get information for a backup.
func (s *databaseBackupService) Get(ctx context.Context, orgID, projectID, instanceID, databaseID, backupID string) (*GetBackupResponse, error) {
	ctx, cancelGetBackup := context.WithTimeout(ctx, s.timeout)
	defer cancelGetBackup()

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.InstanceID(instanceID),
		utils.DatabaseID(databaseID),
		utils.DatabaseBackupID(backupID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting database backup information",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
		slog.String("backupID", backupID),
	)

	path := utils.SingleBackupPath(orgID, projectID, instanceID, databaseID, backupID)

	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get database backup",
			slog.String("instanceID", instanceID),
			slog.String("databaseID", databaseID),
			slog.String("backupID", backupID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result GetBackupResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal get backup response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database backup retrieved successfully",
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
		slog.String("backupID", result.Data.ID),
	)
	return &result, nil
}
