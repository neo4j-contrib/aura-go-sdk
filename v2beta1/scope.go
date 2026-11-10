package v2beta1

// scope.go holds the shared per-call scoping machinery used by every v2beta1
// service. In the v2 API every resource is reached through a fixed spine:
//
//	organizations/{org}/projects/{project}/instances/{instance}/databases/{database}/...
//
// with backup, user and invite identifiers as leaves off that spine. The scope
// that requests need is therefore supplied two ways:
//
//   - a stable default, set once on the client with WithDefaultOrg,
//     WithDefaultProject or WithDefaultInstance, and
//
//   - per-call overrides, passed as ScopeOption values (WithOrg, WithProject,
//     WithInstance, WithDatabase, WithBackup, WithUser, WithInvite).
//
// A per-call option overrides the matching client default; an option that is left
// unset inherits the default. This keeps the common case to a one-line call when
// a default scope is configured (client.Databases.List(ctx)) while still
// supporting any combination of scope on demand. When an operation also carries a
// typed request body, that body comes first and the ScopeOption values trail it
// (for example client.Instances.Create(ctx, req, WithOrg(orgID))).

// requestScope is the resolved org->project->instance->database (+leaf) scope for
// a single operation. It is unexported; callers customise it only through the
// ScopeOption constructors.
type requestScope struct {
	orgID      string
	projectID  string
	instanceID string
	databaseID string
	backupID   string
	userID     string
	inviteID   string
}

// ScopeOption customises the scope of a single operation.
//
// The same option type applies to every service, because every resource is
// reached through the same spine; each service validates only the subset of the
// scope it needs (an org-level operation ignores project/instance/database and so
// on). A leaf field that an operation does not use (for example backupID on a
// database List) is simply ignored.
//
// It is exported so callers may store and reuse a set of options, for example:
//
//	var scope = []v2beta1.ScopeOption{
//	    v2beta1.WithOrg(orgID),
//	    v2beta1.WithProject(projID),
//	}
//	...
//	db, err := client.Databases.Get(ctx, append(scope, v2beta1.WithDatabase(dbID))...)
type ScopeOption func(*requestScope)

// resolveScope returns a copy of def with each option in opts applied in order,
// the last option for any given field winning.
func resolveScope(def requestScope, opts []ScopeOption) requestScope {
	s := def
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// WithOrg sets the organization ID for a scope.
func WithOrg(orgID string) ScopeOption {
	return func(s *requestScope) { s.orgID = orgID }
}

// WithProject sets the project ID for a scope.
func WithProject(projectID string) ScopeOption {
	return func(s *requestScope) { s.projectID = projectID }
}

// WithInstance sets the instance ID for a scope.
func WithInstance(instanceID string) ScopeOption {
	return func(s *requestScope) { s.instanceID = instanceID }
}

// WithDatabase sets the database ID for a scope.
func WithDatabase(databaseID string) ScopeOption {
	return func(s *requestScope) { s.databaseID = databaseID }
}

// WithBackup sets the database backup ID for a scope. Used by
// DatabaseBackupService operations that address a single backup.
func WithBackup(backupID string) ScopeOption {
	return func(s *requestScope) { s.backupID = backupID }
}

// WithUser sets the user ID for a scope. Shared by the organization-user and
// project-user operations that address a specific user.
func WithUser(userID string) ScopeOption {
	return func(s *requestScope) { s.userID = userID }
}

// WithInvite sets the invite ID for a scope. Used by
// OrganizationInviteService.Delete.
func WithInvite(inviteID string) ScopeOption {
	return func(s *requestScope) { s.inviteID = inviteID }
}
