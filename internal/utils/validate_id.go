// A single utils.go file was getting a bit large
// And certain helpers are only for v2beta1
// Split apart for the moment whilst v2beta1 endpoints are being developed
// to improve clarity
// Thse are helpers for validating the various UUIDs

package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
)

// ===============================================================================
// Helper functions for validating the various UUIDs used in Aura API
// ===============================================================================

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

// Checks Organization ID is a long form UUID  xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func OrganizationID(value string) RequiredID {
	return RequiredID{
		Name:       "organization ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "organization ID is required: provide it via WithOrg call option or WithOrganization client option",
		InvalidMsg: "organization ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

// Checks project ID is a long form UUID  xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func ProjectID(value string) RequiredID {
	return RequiredID{
		Name:       "project ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "project ID is required: provide it via WithProject call option or WithDefaultProject client option",
		InvalidMsg: "project ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

// Checks instance ID is a short form UUID  xxxxxxxx
func InstanceID(value string) RequiredID {
	return RequiredID{
		Name:       "instance ID",
		Value:      value,
		Format:     ShortID,
		MissingMsg: "instance ID is required: provide it via WithInstance call option",
		InvalidMsg: "instance ID must be an 8-character hex string formatted as xxxxxxxx",
	}
}

// Checks database ID is a short form UUID  xxxxxxxx
func DatabaseID(value string) RequiredID {
	return RequiredID{
		Name:       "database ID",
		Value:      value,
		Format:     ShortID,
		MissingMsg: "database ID is required",
		InvalidMsg: "datbase ID must be an 8-character hex string formatted as xxxxxxxx",
	}
}

// Checks database backup ID is a long form UUID  xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func DatabaseBackupID(value string) RequiredID {
	return RequiredID{
		Name:       "backup ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "database backup ID is required",
		InvalidMsg: "database backup ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

// UserID checks that the supplied value is a long-form UUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func UserID(value string) RequiredID {
	return RequiredID{
		Name:       "user ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "user ID is required",
		InvalidMsg: "user ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
}

// InviteID checks that the supplied value is a long-form UUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func InviteID(value string) RequiredID {
	return RequiredID{
		Name:       "invite ID",
		Value:      value,
		Format:     UUID,
		MissingMsg: "invite ID is required",
		InvalidMsg: "invite ID must be an hex string formatted as xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
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
