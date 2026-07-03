// Package v2beta1 provides a Go client library for the Neo4j Aura v2beta1 API.
//
// The v2beta1 API exposes organization and project management operations not
// available in the stable v1 API. Construct a client with NewClient and
// authenticate using WithCredentials. Set a default organization with
// WithOrganization, or supply an org ID per-call using WithOrg.
//
// Example usage:
//
//	client, err := v2beta1.NewClient(
//	    v2beta1.WithCredentials("client-id", "client-secret"),
//	    v2beta1.WithOrganization("org-uuid"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	orgs, err := client.Organizations.List(ctx)
package v2beta1

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
)

// ============================================================================
// Constants and version
// ============================================================================

// auraAPIVersion is the API version this client targets.
const auraAPIVersion = "v2beta1"

// clientVersionFallback is embedded in the User-Agent when the real module
// version cannot be determined (local builds, go test, go run).
const clientVersionFallback = "development"

// ClientVersion is the version of this client library, embedded in every
// User-Agent header. It is resolved at runtime from the binary's build info
// so it reflects the exact tagged release the consumer imported.
var ClientVersion = clientVersionFallback

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		ClientVersion = info.Main.Version
	}
}

// ============================================================================
// Client types
// ============================================================================

// Client is the entry point for the Aura v2beta1 API.
type Client struct {
	api    api.RequestService
	logger *slog.Logger

	// Service fields — interface types for testability.
	Organizations   OrganizationService
	Projects        ProjectService
	Instances       InstanceService
	Databases       DatabaseService
	DatabaseBackups DatabaseBackupService

	// Mutex-protected defaults for org and project ID resolution.
	mu               sync.RWMutex
	defaultOrgID     string
	defaultProjectID string
}

// config holds internal configuration (unexported).
type config struct {
	baseURL          string
	apiTimeout       time.Duration
	apiRetryMax      int
	clientID         string
	clientSecret     string
	httpClient       *http.Client
	userAgent        string
	defaultHeaders   map[string]string
	maxResponseSize  int
	defaultOrgID     string
	defaultProjectID string
}

// Option is a functional option for configuring the Client.
type Option func(*options) error

// options holds the configuration applied to the client during construction.
type options struct {
	config config
	logger *slog.Logger
}

// ============================================================================
// Default options
// ============================================================================

// defaultOptions returns options with sensible defaults.
func defaultOptions() *options {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stderr, opts)

	return &options{
		config: config{
			baseURL:         "https://api.neo4j.io",
			apiTimeout:      120 * time.Second,
			apiRetryMax:     3,
			userAgent:       "aura-go-sdk/" + ClientVersion,
			maxResponseSize: 10 * 1024 * 1024, // 10 MB
		},
		logger: slog.New(handler),
	}
}

// ============================================================================
// Option constructors
// ============================================================================

// WithCredentials sets the client ID and secret used for OAuth authentication.
func WithCredentials(clientID, clientSecret string) Option {
	return func(o *options) error {
		if clientID == "" {
			return errors.New("client ID must not be empty")
		}
		if clientSecret == "" {
			return errors.New("client secret must not be empty")
		}
		o.config.clientID = clientID
		o.config.clientSecret = clientSecret
		return nil
	}
}

// WithTimeout sets a custom API timeout. Defaults to 120 seconds.
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) error {
		if timeout <= 0 {
			return errors.New("timeout must be greater than zero")
		}
		o.config.apiTimeout = timeout
		return nil
	}
}

// WithMaxRetry sets the maximum number of retries for failed requests. Defaults to 3.
func WithMaxRetry(maxRetry int) Option {
	return func(o *options) error {
		if maxRetry <= 0 {
			return errors.New("max retries must be greater than zero")
		}
		o.config.apiRetryMax = maxRetry
		return nil
	}
}

// WithLogger sets a custom slog.Logger. Defaults to warn-level logging to stderr.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) error {
		if logger == nil {
			return errors.New("logger cannot be nil")
		}
		o.logger = logger
		return nil
	}
}

// WithMaxResponseSize overrides the default maximum response body size (default: 10MB).
// Responses larger than this limit are rejected with an error.
func WithMaxResponseSize(responseSize int) Option {
	return func(o *options) error {
		if responseSize <= 0 {
			return errors.New("max response size must be greater than zero")
		}
		o.config.maxResponseSize = responseSize
		return nil
	}
}

// WithBaseURL overrides the default API base URL. Useful for staging or sandbox environments.
// The URL must use HTTPS to protect OAuth tokens and API credentials in transit.
func WithBaseURL(baseURL string) Option {
	return func(o *options) error {
		if baseURL == "" {
			return errors.New("base URL must not be empty")
		}
		if !strings.HasPrefix(baseURL, "https://") {
			return errors.New("base URL must use HTTPS to protect credentials in transit")
		}
		o.config.baseURL = baseURL
		return nil
	}
}

// WithInsecureBaseURL overrides the base URL without enforcing HTTPS.
// This is intended for local development and in-process testing only (e.g. httptest.Server).
// Never use this option against a real Aura environment — OAuth tokens and API
// credentials will be transmitted in cleartext over the network.
func WithInsecureBaseURL(baseURL string) Option {
	return func(o *options) error {
		if baseURL == "" {
			return errors.New("base URL must not be empty")
		}
		o.config.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient sets a custom *http.Client to use for all API requests. This
// lets callers inject a custom transport (e.g. for mTLS, proxies, or testing).
// Returns an error if client is nil.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		o.config.httpClient = client
		return nil
	}
}

// WithUserAgent overrides the default User-Agent header sent with every request.
// Returns an error if ua is empty.
func WithUserAgent(ua string) Option {
	return func(o *options) error {
		if ua == "" {
			return errors.New("user agent must not be empty")
		}
		o.config.userAgent = ua
		return nil
	}
}

// protectedHeaders is the set of header keys that WithDefaultHeaders silently
// drops to prevent callers from inadvertently overriding security-sensitive or
// protocol-critical headers.
var protectedHeaders = map[string]struct{}{
	"authorization": {},
	"content-type":  {},
	"user-agent":    {},
}

// WithDefaultHeaders adds the given headers to every API request. It is a no-op
// when headers is nil or empty. Keys matching Authorization, Content-Type, or
// User-Agent (case-insensitive) are silently ignored to protect credentials and
// protocol semantics.
func WithDefaultHeaders(headers map[string]string) Option {
	return func(o *options) error {
		if len(headers) == 0 {
			return nil
		}
		filtered := make(map[string]string, len(headers))
		for k, v := range headers {
			if _, protected := protectedHeaders[strings.ToLower(k)]; !protected {
				filtered[k] = v
			}
		}
		if len(filtered) > 0 {
			o.config.defaultHeaders = filtered
		}
		return nil
	}
}

// WithDefaultOrg sets the default organization ID used when no per-call WithOrg
// option is supplied. The value is stored and readable by SetOrg later.
func WithDefaultOrg(orgID string) Option {
	return func(o *options) error {
		if orgID == "" {
			return errors.New("organization ID must not be empty")
		}
		o.config.defaultOrgID = orgID
		return nil
	}
}

// WithDefaultProject sets the default project ID used when no per-call WithProject
// option is supplied. Named WithDefaultProject to avoid a name collision with the
// per-call WithProject CallOption constructor in calloptions.go.
func WithDefaultProject(projectID string) Option {
	return func(o *options) error {
		if projectID == "" {
			return errors.New("project ID must not be empty")
		}
		o.config.defaultProjectID = projectID
		return nil
	}
}

// ============================================================================
// Constructor
// ============================================================================

// NewClient creates a new Aura v2beta1 API client with functional options.
func NewClient(opts ...Option) (*Client, error) {
	o := defaultOptions()

	for _, opt := range opts {
		if err := opt(o); err != nil {
			o.logger.Error("option application failed", slog.String("error", err.Error()))
			return nil, err
		}
	}

	if o.config.clientID == "" {
		o.logger.Error("validation failed", slog.String("reason", "client ID must not be empty"))
		return nil, errors.New("client ID must not be empty")
	}
	if o.config.clientSecret == "" {
		o.logger.Error("validation failed", slog.String("reason", "client secret must not be empty"))
		return nil, errors.New("client secret must not be empty")
	}
	if o.config.baseURL == "" {
		o.logger.Error("validation failed", slog.String("reason", "base URL must not be empty"))
		return nil, errors.New("base URL must not be empty")
	}
	if o.config.apiTimeout <= 0 {
		o.logger.Error("validation failed", slog.String("reason", "API timeout must be greater than zero"), slog.Duration("timeout", o.config.apiTimeout))
		return nil, errors.New("API timeout must be greater than zero")
	}
	if o.config.userAgent == "" {
		o.logger.Error("validation failed", slog.String("reason", "user agent cannot be empty"))
		return nil, errors.New("user agent cannot be empty")
	}

	o.logger.Debug("configuration validated",
		slog.String("baseURL", o.config.baseURL),
		slog.String("apiVersion", auraAPIVersion),
		slog.Duration("apiTimeout", o.config.apiTimeout),
	)

	apiSvc := api.NewRequestService(api.Config{
		ClientID:        o.config.clientID,
		ClientSecret:    o.config.clientSecret,
		BaseURL:         o.config.baseURL,
		APIVersion:      auraAPIVersion,
		Timeout:         o.config.apiTimeout,
		MaxRetry:        o.config.apiRetryMax,
		UserAgent:       o.config.userAgent,
		HTTPClient:      o.config.httpClient,
		DefaultHeaders:  o.config.defaultHeaders,
		MaxResponseSize: o.config.maxResponseSize,
	}, o.logger)

	clientLogger := o.logger.With(slog.String("component", "v2beta1.Client"))

	client := &Client{
		api:              apiSvc,
		logger:           clientLogger,
		defaultOrgID:     o.config.defaultOrgID,
		defaultProjectID: o.config.defaultProjectID,
	}

	client.Organizations = &organizationService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "organizationService")),
		client:  client,
	}
	client.Projects = &projectService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "projectService")),
		client:  client,
	}
	client.Instances = &instanceService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "instanceService")),
		client:  client,
	}
	client.Databases = &databaseService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "databaseService")),
		client:  client,
	}

	client.DatabaseBackups = &databaseBackupService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "databaseService")),
		client:  client,
	}
	client.logger.Info("Aura v2beta1 API client initialized successfully",
		slog.Int("services", 4),
		slog.String("apiVersion", auraAPIVersion),
	)

	return client, nil
}

// ============================================================================
// Client methods
// ============================================================================

// SetOrg updates the default organization ID used by all service calls that
// do not supply a per-call WithOrg override. It is safe for concurrent use.
func (c *Client) SetOrg(orgID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultOrgID = orgID
}

// SetProject updates the default project ID used by all service calls that
// do not supply a per-call WithProject override. It is safe for concurrent use.
func (c *Client) SetProject(projectID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultProjectID = projectID
}

// Close drains idle HTTP connections held by the underlying transport. It
// should be called via defer when the client is no longer needed to avoid
// leaking file descriptors.
//
//	client, err := v2beta1.NewClient(v2beta1.WithCredentials(id, secret))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
func (c *Client) Close() {
	c.api.Close()
}
