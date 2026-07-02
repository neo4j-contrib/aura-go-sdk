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

// GetDatabase wraps the list of databases  returned by the API.
type GetDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// CreateDatabaseResponse wraps the response from creating a database that is returned by the API.
type CreateDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// ListDatabases wraps the list of databases  returned by the API.
type ListDatabasesResponse struct {
	Data []DatabaseResponse `json:"data"`
}

// DeleteDatabaseResponse wraps the delete database response returned by the API.
type DeleteDatabaseResponse struct {
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

/*
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
*/

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

func (s *databaseService) Create(ctx context.Context, instanceID string, opts ...CallOption) (*CreateDatabaseResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

	// Check IDs are supplied and valid
	// Using new Validate function
	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.InstanceID(instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "creating database",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
	)

	path := utils.SingleInstancePath(orgID, projectID, instanceID)
	// path := instancePath(orgID, projectID, instanceID)
	resp, err := s.api.Post(ctx, path, "")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create database",
			slog.String("instanceID", instanceID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result CreateDatabaseResponse

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database created  successfully",
		slog.String("instanceID", instanceID),
	)

	return &result, nil
}

func (s *databaseService) List(ctx context.Context, instanceID string, opts ...CallOption) (*ListDatabasesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	orgID, projectID := s.resolveOrgProject(opts)

	// Check IDs are supplied and valid
	// Using new Validate function
	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(orgID),
		utils.ProjectID(projectID),
		utils.InstanceID(instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing databases",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
	)

	path := utils.SingleInstancePath(orgID, projectID, instanceID)
	//path := instancePath(orgID, projectID, instanceID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list instance databases",
			slog.String("instanceID", instanceID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result ListDatabasesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal list databases response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "databases listed successfully",
		slog.String("instanceID", instanceID),
		slog.Int("count", len(result.Data)),
	)
	return &result, nil
}

func (s *databaseService) Get(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*GetDatabaseResponse, error) {
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

	s.logger.DebugContext(ctx, "getting databases",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)

	path := utils.BackupsPath(orgID, projectID, instanceID, databaseID)
	// path := backupsPath(orgID, projectID, instanceID, databaseID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get database",
			slog.String("instanceID", instanceID),
			slog.String("databaseID", databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result GetDatabaseResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal get database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "get database was successfull",
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)
	return &result, nil
}

func (s *databaseService) Delete(ctx context.Context, instanceID, databaseID string, opts ...CallOption) (*DeleteDatabaseResponse, error) {
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

	s.logger.DebugContext(ctx, "deleting database",
		slog.String("orgID", orgID),
		slog.String("projectID", projectID),
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)

	path := utils.BackupsPath(orgID, projectID, instanceID, databaseID)
	// path := backupsPath(orgID, projectID, instanceID, databaseID)
	resp, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete database",
			slog.String("instanceID", instanceID),
			slog.String("databaseID", databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result DeleteDatabaseResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal delete database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "database deleted  successfully",
		slog.String("instanceID", instanceID),
		slog.String("databaseID", databaseID),
	)
	return &result, nil
}
