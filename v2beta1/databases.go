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

// GetDatabaseResponse wraps the single database returned by the API.
type GetDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// CreateDatabaseResponse wraps the response from creating a database that is returned by the API.
type CreateDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// ListDatabasesResponse wraps the list of databases returned by the API.
type ListDatabasesResponse struct {
	Data []DatabaseResponse `json:"data"`
}

// DeleteDatabaseResponse wraps the delete database response returned by the API.
type DeleteDatabaseResponse struct {
	Data DatabaseResponse `json:"data"`
}

// DatabaseResponse represents a single Aura database.
type DatabaseResponse struct {
	ID string `json:"id"`
}

// ============================================================================
// Service
// ============================================================================

// databaseService handles database operations for the v2beta1 API.
//
// It is scoped through the client-level default scope; each call may override
// individual fields of it via the ScopeOption values passed to the method.
type databaseService struct {
	api          api.RequestService
	timeout      time.Duration
	logger       *slog.Logger
	defaultScope requestScope
}

// List returns all databases on the instance selected by the scope options,
// inheriting any unset field from the client's default scope.
func (s *databaseService) List(ctx context.Context, opts ...ScopeOption) (*ListDatabasesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "listing databases",
		slog.String("orgID", sc.orgID),
		slog.String("projectID", sc.projectID),
		slog.String("instanceID", sc.instanceID),
	)

	path := utils.DatabasesPath(sc.orgID, sc.projectID, sc.instanceID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list instance databases",
			slog.String("instanceID", sc.instanceID),
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
		slog.String("instanceID", sc.instanceID),
		slog.Int("count", len(result.Data)),
	)

	return &result, nil
}

// Get returns information about a single database selected by the scope options,
// inheriting any unset field from the client's default scope.
func (s *databaseService) Get(ctx context.Context, opts ...ScopeOption) (*GetDatabaseResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
		utils.DatabaseID(sc.databaseID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "getting database",
		slog.String("orgID", sc.orgID),
		slog.String("projectID", sc.projectID),
		slog.String("instanceID", sc.instanceID),
		slog.String("databaseID", sc.databaseID),
	)

	path := utils.SingleDatabasePath(sc.orgID, sc.projectID, sc.instanceID, sc.databaseID)
	resp, err := s.api.Get(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get database",
			slog.String("instanceID", sc.instanceID),
			slog.String("databaseID", sc.databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result GetDatabaseResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal get database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database retrieved successfully",
		slog.String("instanceID", sc.instanceID),
		slog.String("databaseID", sc.databaseID),
	)

	return &result, nil
}

// Create provisions a new database for the instance selected by the scope options.
func (s *databaseService) Create(ctx context.Context, opts ...ScopeOption) (*CreateDatabaseResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "creating database",
		slog.String("orgID", sc.orgID),
		slog.String("projectID", sc.projectID),
		slog.String("instanceID", sc.instanceID),
	)

	path := utils.DatabasesPath(sc.orgID, sc.projectID, sc.instanceID)
	resp, err := s.api.Post(ctx, path, "")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create database",
			slog.String("instanceID", sc.instanceID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var result CreateDatabaseResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal create database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.DebugContext(ctx, "database created successfully",
		slog.String("instanceID", sc.instanceID),
	)

	return &result, nil
}

// Delete removes the database selected by the scope options from an instance.
func (s *databaseService) Delete(ctx context.Context, opts ...ScopeOption) (*DeleteDatabaseResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sc := resolveScope(s.defaultScope, opts)

	if err := utils.Validate(ctx, s.logger,
		utils.OrganizationID(sc.orgID),
		utils.ProjectID(sc.projectID),
		utils.InstanceID(sc.instanceID),
		utils.DatabaseID(sc.databaseID),
	); err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "deleting database",
		slog.String("orgID", sc.orgID),
		slog.String("projectID", sc.projectID),
		slog.String("instanceID", sc.instanceID),
		slog.String("databaseID", sc.databaseID),
	)

	path := utils.SingleDatabasePath(sc.orgID, sc.projectID, sc.instanceID, sc.databaseID)
	resp, err := s.api.Delete(ctx, path)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete database",
			slog.String("instanceID", sc.instanceID),
			slog.String("databaseID", sc.databaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	if len(resp.Body) == 0 {
		s.logger.InfoContext(ctx, "database deleted successfully",
			slog.String("instanceID", sc.instanceID),
			slog.String("databaseID", sc.databaseID),
		)
		return &DeleteDatabaseResponse{}, nil
	}

	var result DeleteDatabaseResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal delete database response", slog.String("error", err.Error()))
		return nil, err
	}

	s.logger.InfoContext(ctx, "database deleted successfully",
		slog.String("instanceID", sc.instanceID),
		slog.String("databaseID", sc.databaseID),
	)

	return &result, nil
}
