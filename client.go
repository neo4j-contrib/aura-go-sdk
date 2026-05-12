// Package aura provides a Go client library for the Neo4j Aura API.
//
// The client supports all major Aura API operations including instance management,
// snapshots, tenant operations, and customer-managed encryption keys (CMEK).
//
// Example usage:
//
//	client, err := aura.NewClient(
//	    aura.WithCredentials("client-id", "client-secret"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	instances, err := client.Instances.List(ctx)
package aura

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
)

// ============================================================================
// Constants and version
// ============================================================================

// auraAPIVersion is the version of the Aura API this client targets.
// It is intentionally not user-configurable — a new major API version
// will be delivered as a separate module (e.g. aura-api-client/v2).
const auraAPIVersion = "v1"

// auraAPIClientVersionFallback is the hardcoded fallback used when
// debug.ReadBuildInfo() is unavailable (devel builds, go test, go run).
const auraAPIClientVersionFallback = "v1.10.0"

// AuraAPIClientVersion is the version of this client library. It is populated
// at init time from the module's build info, falling back to the literal version
// in devel/test builds where build info is unavailable.
var AuraAPIClientVersion = auraAPIClientVersionFallback

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		AuraAPIClientVersion = info.Main.Version
	}
}

// -ldflags "-X 'main.Version=9999'"
// This gets set as part of the Github workflow
var ClientVersion = "development"

// ============================================================================
// Client types
// ============================================================================

// AuraAPIClient is the main client for interacting with the Neo4j Aura API.
//
//nolint:revive // AuraAPIClient is intentional: the package is named aura and the type name is established in v1.
type AuraAPIClient struct {
	api    api.RequestService // Handles authenticated API requests
	logger *slog.Logger       // Structured logger

	// Grouped services — using interface types for testability.
	Tenants        TenantService
	Instances      InstanceService
	Snapshots      SnapshotService
	Cmek           CmekService
	GraphAnalytics GDSSessionService
	Prometheus     PrometheusService
}

// config holds internal configuration (unexported).
type config struct {
	baseURL        string            // the base URL of the Aura API
	apiTimeout     time.Duration     // how long to wait for a response from an Aura API endpoint
	apiRetryMax    int               // the number of retries to attempt
	clientID       string            // client ID used to obtain an OAuth token
	clientSecret   string            // client secret used to obtain an OAuth token
	httpClient     *http.Client      // optional custom HTTP client (injected transport)
	userAgent      string            // optional User-Agent override
	defaultHeaders map[string]string // optional headers added to every API request
}

// Option is a functional option for configuring the AuraAPIClient.
type Option func(*options) error

// options holds the configuration that will be applied to the client.
type options struct {
	config config
	logger *slog.Logger
}

// ============================================================================
// Constructor and options
// ============================================================================

// defaultOptions returns options with sensible defaults.
func defaultOptions() *options {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stderr, opts)

	return &options{
		config: config{
			baseURL:     "https://api.neo4j.io",
			apiTimeout:  120 * time.Second,
			apiRetryMax: 3,
		},
		logger: slog.New(handler),
	}
}

// WithCredentials sets the client ID and secret used for OAuth authentication.
func WithCredentials(clientID, clientSecret string) Option {
	return func(o *options) error {
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

// Close drains idle HTTP connections held by the underlying transport. It
// should be called via defer when the client is no longer needed to avoid
// leaking file descriptors.
//
//	client, err := aura.NewClient(aura.WithCredentials(id, secret))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
func (c *AuraAPIClient) Close() {
	c.api.Close()
}

// NewClient creates a new Aura API client with functional options.
func NewClient(opts ...Option) (*AuraAPIClient, error) {
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

	o.logger.Debug("configuration validated",
		slog.String("baseURL", o.config.baseURL),
		slog.String("apiVersion", auraAPIVersion),
		slog.Duration("apiTimeout", o.config.apiTimeout),
	)

	userAgent := "aura-go-client/" + ClientVersion
	if o.config.userAgent != "" {
		userAgent = o.config.userAgent
	}

	apiSvc := api.NewRequestService(api.Config{
		ClientID:       o.config.clientID,
		ClientSecret:   o.config.clientSecret,
		BaseURL:        o.config.baseURL,
		APIVersion:     auraAPIVersion,
		Timeout:        o.config.apiTimeout,
		MaxRetry:       o.config.apiRetryMax,
		UserAgent:      userAgent,
		HTTPClient:     o.config.httpClient,
		DefaultHeaders: o.config.defaultHeaders,
	}, o.logger)

	clientLogger := o.logger.With(slog.String("component", "AuraAPIClient"))

	service := &AuraAPIClient{
		api:    apiSvc,
		logger: clientLogger,
	}

	service.Tenants = &tenantService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "tenantService")),
	}
	service.Instances = &instanceService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "instanceService")),
	}
	service.Snapshots = &snapshotService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "snapshotService")),
	}
	service.Cmek = &cmekService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "cmekService")),
	}
	service.GraphAnalytics = &gdsSessionService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "gdsSessionService")),
	}
	service.Prometheus = &prometheusService{
		api:     apiSvc,
		timeout: o.config.apiTimeout,
		logger:  clientLogger.With(slog.String("service", "prometheusService")),
	}

	service.logger.Info("Aura API client initialized successfully",
		slog.Int("services", 6),
		slog.String("version", ClientVersion),
		slog.String("apiVersion", auraAPIVersion),
	)

	return service, nil
}
