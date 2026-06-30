// Package utils provides shared internal helpers for the aura-client module.
// Nothing in this package is part of the public API.
package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
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
