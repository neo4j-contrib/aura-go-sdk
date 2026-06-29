package v2beta1

// CallOption is a functional option applied per-call to override client defaults.
type CallOption func(*callConfig)

// callConfig holds per-call overrides resolved before each API request.
type callConfig struct {
	orgID     string
	projectID string
}

// WithOrg returns a CallOption that sets the organization ID for a single call,
// overriding the client-level default.
func WithOrg(orgID string) CallOption {
	return func(c *callConfig) {
		c.orgID = orgID
	}
}

// WithProject returns a CallOption that sets the project ID for a single call,
// overriding the client-level default.
func WithProject(projectID string) CallOption {
	return func(c *callConfig) {
		c.projectID = projectID
	}
}

// applyOptions builds a callConfig by applying all provided CallOptions.
// Call sites that need both org and project resolution should call this once
// and inspect the returned struct directly, rather than calling resolveOrg
// and resolveProject separately.
func applyOptions(opts []CallOption) callConfig {
	cfg := callConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// resolveOrg returns the per-call org ID from opts if non-empty, otherwise
// returns the supplied clientDefault.
func resolveOrg(clientDefault string, opts []CallOption) string {
	cfg := applyOptions(opts)
	if cfg.orgID != "" {
		return cfg.orgID
	}
	return clientDefault
}

// resolveProject returns the per-call project ID from opts if non-empty,
// otherwise returns the supplied clientDefault.
func resolveProject(clientDefault string, opts []CallOption) string {
	cfg := applyOptions(opts)
	if cfg.projectID != "" {
		return cfg.projectID
	}
	return clientDefault
}
