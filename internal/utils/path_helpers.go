// A single utils.go file was getting a bit large
// And certain helpers are only for v2beta1
// Split apart for the moment whilst v2beta1 endpoints are being developed
// to improve clarity
// Thse are helpers for building the longer paths used in v2beta1

package utils

import "strings"

// ===============================================================================
// Helper functions for building endpoint paths
// ===============================================================================
func resourcePath(parts ...string) string {
	return strings.Join(parts, "/")
}

// Returns endpoint path for a single OrgID
func SingleOrgPath(orgID string) string {
	return resourcePath("organizations", orgID)
}

// Returns endpoint path for org invites
func OrgInvitesPath(orgID string) string {
	return resourcePath("organizations", orgID, "invites")
}

// Returns endpoint path for a single org invite
func SingleInvitePath(orgID, inviteID string) string {
	return resourcePath("organizations", orgID, "invites", inviteID)
}

// Returns endpoint path for org users
func OrgUsersPath(orgID string) string {
	return resourcePath("organizations", orgID, "users")
}

// Returns endpoint path for a single org user
func SingleOrgUserPath(orgID, userID string) string {
	return resourcePath("organizations", orgID, "users", userID)
}

// Returns endpoint path for projects under OrgID
func ProjectsPath(orgID string) string {
	return resourcePath("organizations", orgID, "projects")
}

// Returns endpoint path for a project
func SingleProjectPath(orgID, projectID string) string {
	return resourcePath("organizations", orgID, "projects", projectID)
}

// Returns endpoint path for instances in an org/project
func InstancesPath(orgID, projectID string) string {
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

// Returns endpoint path for a single backup of a database
func SingleBackupPath(orgID, projectID, instanceID, databaseID, backupID string) string {
	return resourcePath("organizations", orgID, "projects", projectID, "instances", instanceID, "databases", databaseID, "backups", backupID)
}
