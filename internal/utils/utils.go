// Package utils provides shared internal helpers for the aura-client module.
// Nothing in this package is part of the public API.
package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// Base64Encode returns the standard base64 encoding of "s1:s2", suitable for
// use as the credential in an HTTP Basic Authorization header.
func Base64Encode(s1, s2 string) string {
	auth := s1 + ":" + s2
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Unmarshal copies a JSON payload into a value of type T.
func Unmarshal[T any](payload []byte) (T, error) {
	var result T
	err := json.Unmarshal(payload, &result)
	return result, err
}

// Marshal returns the JSON encoding of payload.
func Marshal(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

// MarshalIndent returns the indented JSON encoding of payload.
func MarshalIndent(payload any) ([]byte, error) {
	return json.MarshalIndent(payload, "", "  ")
}

// CheckDate returns an error if t is not a valid YYYY-MM-DD date string.
func CheckDate(t string) error {
	_, err := time.Parse(time.DateOnly, t)
	if err != nil {
		return fmt.Errorf("the date must in the format of YYYY-MM-DD")
	}
	return nil
}

// There are two types of UUID used in the Aura API
// One that follows 8-4-4-4-12 pattern e.g for Project ID
// the other that is a 8 pattern e.g instances ID
// so we have uuidPattern and shortIDPattern
// defined
var (
	shortIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)
	uuidPattern    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// IDFormat describes which shape an ID is expected to match.
type IDFormat int

const (
	ShortID IDFormat = iota
	UUID
)

// RequiredID describes a single ID to validate, along with the
// error message to use if it's missing or malformed.
type RequiredID struct {
	Name       string   // human-readable name, e.g. "organization ID"
	Value      string   // the actual value being checked
	Format     IDFormat // expected format
	MissingMsg string   // error text if empty
	InvalidMsg string   // error text if present but malformed
}

// constructors for the different IDs we need to validate
// Each constructor takes only the value, since the rest is fixed per ID type.

func OrganizationID(value string) RequiredID {
	return RequiredID{
		Name:       "organization ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "organization ID is required: provide it via WithOrg call option or WithOrganization client option",
		InvalidMsg: "organization ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

func ProjectID(value string) RequiredID {
	return RequiredID{
		Name:       "project ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "project ID is required: provide it via WithProject call option or WithDefaultProject client option",
		InvalidMsg: "project ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

func InstanceID(value string) RequiredID {
	return RequiredID{
		Name:       "instance ID",
		Value:      value,
		Format:     ShortID,
		MissingMsg: "instance ID is required: provide it via WithInstance call option",
		InvalidMsg: "instance ID must be an 8-character hex string formatted as xxxxxxxx",
	}
}

func DatabaseID(value string) RequiredID {
	return RequiredID{
		Name:       "database ID",
		Value:      value,
		Format:     ShortID,
		MissingMsg: "database ID is required",
		InvalidMsg: "datbase ID must be an 8-character hex string formatted as xxxxxxxx",
	}
}

// Validate checks a list of RequiredIDs in order, returning the first
// error encountered (missing or invalid), logging via the given logger.
// This allows us to call it with just a project ID to validate or several IDs as needed by
// some endpoint paths
func Validate(ctx context.Context, logger *slog.Logger, ids ...RequiredID) error {
	for _, id := range ids {
		if id.Value == "" {
			err := errors.New(id.MissingMsg)
			logger.ErrorContext(ctx, fmt.Sprintf("missing %s", id.Name), slog.String("error", err.Error()))
			return err
		}

		var pattern *regexp.Regexp
		switch id.Format {
		case UUID:
			pattern = uuidPattern
		default:
			pattern = shortIDPattern
		}

		if !pattern.MatchString(id.Value) {
			err := errors.New(id.InvalidMsg)
			logger.ErrorContext(ctx, fmt.Sprintf("invalid %s", id.Name), slog.String("error", err.Error()), slog.String("value", id.Value))
			return err
		}
	}
	return nil
}

// uuidRegex matches a standard 8-4-4-4-12 UUID. Compiled once at package init
// and shared by ValidateTenantID, ValidateSnapshotID, ValidateProjectID, and ValidateOrgID
var uuidRegex = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

// ValidateTenantID returns an error if tenantID is empty or not a valid UUID.
// Tenant is only used by v1 of Aura API
func ValidateTenantID(tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID must not be empty")
	}
	if !uuidRegex.MatchString(tenantID) {
		return fmt.Errorf("tenant ID must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}
	return nil
}

// V2 Aura API uses Project ID instead of Tenant. This is a copy of  ValidateTenantID using Projects to avoid confusion
// It returns an error if projectID is empty or not a valid UUID.
func ValidateProjectID(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project ID must not be empty: provide it via WithProject call option or WithDefaultProject client option")
	}
	if !uuidRegex.MatchString(projectID) {
		return fmt.Errorf("project ID must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}
	return nil
}

// V2 Aura API expects Org Ids in most calls. This returns an error if OrgID is empty or not a valid UUID.
func ValidateOrgID(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("organization ID must not be empty provide it via WithOrg call option or WithOrganization client option")
	}
	if !uuidRegex.MatchString(orgID) {
		return fmt.Errorf("organization ID must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}
	return nil
}

// ValidateSnapshotID returns an error if snapshotID is empty or not a valid UUID.
func ValidateSnapshotID(snapshotID string) error {
	if snapshotID == "" {
		return fmt.Errorf("snapshot ID must not be empty")
	}
	if !uuidRegex.MatchString(snapshotID) {
		return fmt.Errorf("snapshot ID must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}
	return nil
}

// uuidInstanceIDRegex matches an 8-character hex  ID.
// Used by ValidateInstanceID and ValidateDatabaseID

var uuidInstanceIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)

// ValidateInstanceID returns an error if instanceID is empty or not an 8-character hex string.
func ValidateInstanceID(instanceID string) error {
	if instanceID == "" {
		return fmt.Errorf("instance ID must not be empty")
	}
	if !uuidInstanceIDRegex.MatchString(instanceID) {
		return fmt.Errorf("instance ID must be in the format of a 8-character hex string (xxxxxxxx)")
	}
	return nil
}

// ValidateDatabaseID returns an error if databaseID is empty or not an 8-character hex string.
func ValidateDatabaseID(databaseID string) error {
	if databaseID == "" {
		return fmt.Errorf("databaseID ID must not be empty")
	}
	if !uuidInstanceIDRegex.MatchString(databaseID) {
		return fmt.Errorf("databaseID ID must be in the format of a 8-character hex string (xxxxxxxx)")
	}
	return nil
}

// TruncateString iterate only as far as needed rather than allocated full run slice
func TruncateString(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

// ===============================================================================
// Helper functions for building endpoint paths
// ===============================================================================
func resourcePath(parts ...string) string {
	return strings.Join(parts, "/")
}

// Returns endpoint path for projects under OrgID
func ProjectsPath(orgID string) string {
	return resourcePath("organizations", orgID, "projects")
}

// Returns endpoint path for a project
func IndividualProjectPath(orgID, projectID string) string {
	return resourcePath("organizations", orgID, "projects", projectID)
}

// Returns endpoint path for instances in an org/project
func InstancesPath(orgID, projectID, instanceID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances")
}

// Returns endpoint path for an instance
func SingleInstancePath(orgID, projectID, instanceID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances", instanceID)
}

// Returns endpoint path for the databases of an instance
func DatabasesPath(orgID, projectID, instanceID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances", instanceID, "databases")
}

// Returns endpoint path for a database in an instance
func SingleDatabasePath(orgID, projectID, instanceID, databaseID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances", instanceID, "databases", databaseID)
}

// Returns endpoint path for backups of a database
func BackupsPath(orgID, projectID, instanceID, databaseID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances", instanceID, "databases", databaseID, "backups")
}
