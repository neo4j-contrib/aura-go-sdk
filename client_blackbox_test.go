// client_blackbox_test.go – black-box tests for the aura package.
//
// By using package aura_test (note: NOT package aura) this file can only access
// exported symbols.  That discipline ensures the tests exercise the public API
// exactly as a downstream consumer would, without relying on any unexported
// helpers, types, or fields.
//
// Patterns demonstrated here are therefore directly copy-paste-useful for
// callers who want to write their own unit tests against code that depends on
// the aura client.
package aura_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	aura "github.com/LackOfMorals/aura-client"
)

// =============================================================================
// Mock service implementations
//
// Because this is a black-box test file we cannot reuse the internal mocks
// from test_helpers.go.  We implement the exported interfaces directly here.
// These mocks are intentionally simple: configure the return values you want,
// then call the method, then inspect the call-tracking fields.
// =============================================================================

// --- Instances ---------------------------------------------------------------

type mockInstanceService struct {
	ListResp      *aura.ListInstancesResponse
	ListErr       error
	GetResp       *aura.GetInstanceResponse
	GetErr        error
	CreateResp    *aura.CreateInstanceResponse
	CreateErr     error
	DeleteResp    *aura.DeleteInstanceResponse
	DeleteErr     error
	PauseResp     *aura.GetInstanceResponse
	PauseErr      error
	ResumeResp    *aura.GetInstanceResponse
	ResumeErr     error
	UpdateResp    *aura.GetInstanceResponse
	UpdateErr     error
	OverwriteResp *aura.OverwriteInstanceResponse
	OverwriteErr  error

	LastMethod           string
	LastInstanceID       string
	LastSourceInstanceID string
	LastSourceSnapshotID string
	LastCreateReq        *aura.CreateInstanceConfigData
	LastUpdateReq        *aura.UpdateInstanceData
	CallCount            int
}

func (m *mockInstanceService) List(_ context.Context) (*aura.ListInstancesResponse, error) {
	m.LastMethod = "List"
	m.CallCount++
	return m.ListResp, m.ListErr
}
func (m *mockInstanceService) Get(_ context.Context, id string) (*aura.GetInstanceResponse, error) {
	m.LastMethod = "Get"
	m.LastInstanceID = id
	m.CallCount++
	return m.GetResp, m.GetErr
}
func (m *mockInstanceService) Create(_ context.Context, req *aura.CreateInstanceConfigData) (*aura.CreateInstanceResponse, error) {
	m.LastMethod = "Create"
	m.LastCreateReq = req
	m.CallCount++
	return m.CreateResp, m.CreateErr
}
func (m *mockInstanceService) Delete(_ context.Context, id string) (*aura.DeleteInstanceResponse, error) {
	m.LastMethod = "Delete"
	m.LastInstanceID = id
	m.CallCount++
	return m.DeleteResp, m.DeleteErr
}
func (m *mockInstanceService) Pause(_ context.Context, id string) (*aura.GetInstanceResponse, error) {
	m.LastMethod = "Pause"
	m.LastInstanceID = id
	m.CallCount++
	return m.PauseResp, m.PauseErr
}
func (m *mockInstanceService) Resume(_ context.Context, id string) (*aura.GetInstanceResponse, error) {
	m.LastMethod = "Resume"
	m.LastInstanceID = id
	m.CallCount++
	return m.ResumeResp, m.ResumeErr
}
func (m *mockInstanceService) Update(_ context.Context, id string, req *aura.UpdateInstanceData) (*aura.GetInstanceResponse, error) {
	m.LastMethod = "Update"
	m.LastInstanceID = id
	m.LastUpdateReq = req
	m.CallCount++
	return m.UpdateResp, m.UpdateErr
}
func (m *mockInstanceService) OverwriteFromInstance(_ context.Context, id, srcInst string) (*aura.OverwriteInstanceResponse, error) {
	m.LastMethod = "Overwrite"
	m.LastInstanceID = id
	m.LastSourceInstanceID = srcInst
	m.CallCount++
	return m.OverwriteResp, m.OverwriteErr
}

func (m *mockInstanceService) OverwriteFromSnapshot(_ context.Context, id, srcSnap string) (*aura.OverwriteInstanceResponse, error) {
	m.LastMethod = "Overwrite"
	m.LastInstanceID = id
	m.LastSourceSnapshotID = srcSnap
	m.CallCount++
	return m.OverwriteResp, m.OverwriteErr
}

// --- Tenants -----------------------------------------------------------------

type mockTenantService struct {
	ListResp       *aura.ListTenantsResponse
	ListErr        error
	GetResp        *aura.GetTenantResponse
	GetErr         error
	GetMetricsResp *aura.GetTenantMetricsURLResponse
	GetMetricsErr  error

	LastMethod   string
	LastTenantID string
	CallCount    int
}

func (m *mockTenantService) List(_ context.Context) (*aura.ListTenantsResponse, error) {
	m.LastMethod = "List"
	m.CallCount++
	return m.ListResp, m.ListErr
}
func (m *mockTenantService) Get(_ context.Context, id string) (*aura.GetTenantResponse, error) {
	m.LastMethod = "Get"
	m.LastTenantID = id
	m.CallCount++
	return m.GetResp, m.GetErr
}
func (m *mockTenantService) GetMetrics(_ context.Context, id string) (*aura.GetTenantMetricsURLResponse, error) {
	m.LastMethod = "GetMetrics"
	m.LastTenantID = id
	m.CallCount++
	return m.GetMetricsResp, m.GetMetricsErr
}

// --- Snapshots ---------------------------------------------------------------

type mockSnapshotService struct {
	ListResp    *aura.GetSnapshotsResponse
	ListErr     error
	CreateResp  *aura.CreateSnapshotResponse
	CreateErr   error
	GetResp     *aura.GetSnapshotDataResponse
	GetErr      error
	RestoreResp *aura.RestoreSnapshotResponse
	RestoreErr  error

	LastMethod     string
	LastInstanceID string
	LastSnapshotID string
	LastDate       *aura.SnapshotDate
	CallCount      int
}

func (m *mockSnapshotService) List(_ context.Context, instanceID string, date *aura.SnapshotDate) (*aura.GetSnapshotsResponse, error) {
	m.LastMethod = "List"
	m.LastInstanceID = instanceID
	m.LastDate = date
	m.CallCount++
	return m.ListResp, m.ListErr
}
func (m *mockSnapshotService) Create(_ context.Context, instanceID string) (*aura.CreateSnapshotResponse, error) {
	m.LastMethod = "Create"
	m.LastInstanceID = instanceID
	m.CallCount++
	return m.CreateResp, m.CreateErr
}
func (m *mockSnapshotService) Get(_ context.Context, instanceID, snapshotID string) (*aura.GetSnapshotDataResponse, error) {
	m.LastMethod = "Get"
	m.LastInstanceID = instanceID
	m.LastSnapshotID = snapshotID
	m.CallCount++
	return m.GetResp, m.GetErr
}
func (m *mockSnapshotService) Restore(_ context.Context, instanceID, snapshotID string) (*aura.RestoreSnapshotResponse, error) {
	m.LastMethod = "Restore"
	m.LastInstanceID = instanceID
	m.LastSnapshotID = snapshotID
	m.CallCount++
	return m.RestoreResp, m.RestoreErr
}

// --- CMEK --------------------------------------------------------------------

type mockCmekService struct {
	ListResp *aura.GetCMEKsResponse
	ListErr  error

	LastTenantID string
	CallCount    int
}

func (m *mockCmekService) List(_ context.Context, tenantID string) (*aura.GetCMEKsResponse, error) {
	m.LastTenantID = tenantID
	m.CallCount++
	return m.ListResp, m.ListErr
}

// --- Graph Analytics (GDS Sessions) -----------------------------------------

type mockGDSSessionService struct {
	ListResp     *aura.GetGDSSessionListResponse
	ListErr      error
	EstimateResp *aura.GDSSessionSizeEstimationResponse
	EstimateErr  error
	CreateResp   *aura.GetGDSSessionResponse
	CreateErr    error
	GetResp      *aura.GetGDSSessionResponse
	GetErr       error
	DeleteResp   *aura.DeleteGDSSessionResponse
	DeleteErr    error

	LastMethod    string
	LastSessionID string
	CallCount     int
}

func (m *mockGDSSessionService) List(_ context.Context) (*aura.GetGDSSessionListResponse, error) {
	m.LastMethod = "List"
	m.CallCount++
	return m.ListResp, m.ListErr
}
func (m *mockGDSSessionService) Estimate(_ context.Context, _ *aura.GetGDSSessionSizeEstimation) (*aura.GDSSessionSizeEstimationResponse, error) {
	m.LastMethod = "Estimate"
	m.CallCount++
	return m.EstimateResp, m.EstimateErr
}
func (m *mockGDSSessionService) Create(_ context.Context, _ *aura.CreateGDSSessionConfigData) (*aura.GetGDSSessionResponse, error) {
	m.LastMethod = "Create"
	m.CallCount++
	return m.CreateResp, m.CreateErr
}
func (m *mockGDSSessionService) Get(_ context.Context, id string) (*aura.GetGDSSessionResponse, error) {
	m.LastMethod = "Get"
	m.LastSessionID = id
	m.CallCount++
	return m.GetResp, m.GetErr
}
func (m *mockGDSSessionService) Delete(_ context.Context, id string) (*aura.DeleteGDSSessionResponse, error) {
	m.LastMethod = "Delete"
	m.LastSessionID = id
	m.CallCount++
	return m.DeleteResp, m.DeleteErr
}

// --- Prometheus --------------------------------------------------------------

type mockPrometheusService struct {
	FetchResp  *aura.PrometheusMetricsResponse
	FetchErr   error
	GetValResp float64
	GetValErr  error
	HealthResp *aura.PrometheusHealthMetrics
	HealthErr  error

	LastMethod     string
	LastInstanceID string
	LastMetricName string
	CallCount      int
}

func (m *mockPrometheusService) FetchRawMetrics(_ context.Context, _ string) (*aura.PrometheusMetricsResponse, error) {
	m.LastMethod = "FetchRawMetrics"
	m.CallCount++
	return m.FetchResp, m.FetchErr
}
func (m *mockPrometheusService) GetMetricValue(_ context.Context, _ *aura.PrometheusMetricsResponse, name string, _ map[string]string) (float64, error) {
	m.LastMethod = "GetMetricValue"
	m.LastMetricName = name
	m.CallCount++
	return m.GetValResp, m.GetValErr
}
func (m *mockPrometheusService) GetInstanceHealth(_ context.Context, instanceID, _ string) (*aura.PrometheusHealthMetrics, error) {
	m.LastMethod = "GetInstanceHealth"
	m.LastInstanceID = instanceID
	m.CallCount++
	return m.HealthResp, m.HealthErr
}

// =============================================================================
// Helpers
// =============================================================================

// newTestClient creates a real AuraAPIClient using only exported options.
// Because it uses dummy credentials that never hit the network during
// construction, it is safe to call from any test.
func newTestClient(t *testing.T, extraOpts ...aura.Option) *aura.AuraAPIClient {
	t.Helper()
	opts := append([]aura.Option{aura.WithCredentials("bb-test-id", "bb-test-secret")}, extraOpts...)
	client, err := aura.NewClient(opts...)
	if err != nil {
		t.Fatalf("newTestClient: unexpected error: %v", err)
	}
	return client
}

// discardLogger returns a slog.Logger that silently drops all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.NewFile(0, os.DevNull), &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// Client construction
// =============================================================================

func TestBlackBox_NewClient_ValidCredentials(t *testing.T) {
	client, err := aura.NewClient(aura.WithCredentials("my-id", "my-secret"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_NewClient_MissingClientID(t *testing.T) {
	_, err := aura.NewClient(aura.WithCredentials("", "some-secret"))
	if err == nil {
		t.Fatal("expected error for empty client ID")
	}
	if err.Error() != "client ID must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_NewClient_MissingClientSecret(t *testing.T) {
	_, err := aura.NewClient(aura.WithCredentials("some-id", ""))
	if err == nil {
		t.Fatal("expected error for empty client secret")
	}
	if err.Error() != "client secret must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_NewClient_NoOptions(t *testing.T) {
	_, err := aura.NewClient()
	if err == nil {
		t.Fatal("expected error when no credentials provided")
	}
}

func TestBlackBox_NewClient_ServicesNonNil(t *testing.T) {
	client := newTestClient(t)

	if client.Instances == nil {
		t.Error("Instances service is nil")
	}
	if client.Tenants == nil {
		t.Error("Tenants service is nil")
	}
	if client.Snapshots == nil {
		t.Error("Snapshots service is nil")
	}
	if client.CMEK == nil {
		t.Error("CMEK service is nil")
	}
	if client.GraphAnalytics == nil {
		t.Error("GraphAnalytics service is nil")
	}
	if client.Prometheus == nil {
		t.Error("Prometheus service is nil")
	}
}

func TestBlackBox_NewClient_WithTimeout_Valid(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithTimeout(45*time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_NewClient_WithTimeout_Zero(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithTimeout(0),
	)
	if err == nil {
		t.Fatal("expected error for zero timeout")
	}
	if err.Error() != "timeout must be greater than zero" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_NewClient_WithTimeout_Negative(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithTimeout(-1*time.Second),
	)
	if err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestBlackBox_NewClient_WithBaseURL_Valid(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithBaseURL("https://api.staging.neo4j.io"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_NewClient_WithBaseURL_Empty(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithBaseURL(""),
	)
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
	if err.Error() != "base URL must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_NewClient_WithLogger_Valid(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_NewClient_WithLogger_Nil(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithLogger(nil),
	)
	if err == nil {
		t.Fatal("expected error for nil logger")
	}
	if err.Error() != "logger cannot be nil" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_NewClient_WithMaxRetry_Valid(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithMaxRetry(10),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_NewClient_WithMaxRetry_Zero(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithMaxRetry(0),
	)
	if err == nil {
		t.Fatal("expected error for zero max retry")
	}
}

func TestBlackBox_NewClient_WithMaxRetry_Negative(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithMaxRetry(-5),
	)
	if err == nil {
		t.Fatal("expected error for negative max retry")
	}
}

func TestBlackBox_NewClient_AllValidOptions(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithBaseURL("https://api.sandbox.neo4j.io"),
		aura.WithTimeout(30*time.Second),
		aura.WithMaxRetry(5),
		aura.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// Options are applied in order; a later valid option should not undo an earlier
// option that already failed — construction must stop at the first error.
func TestBlackBox_NewClient_OptionAppliedInOrder(t *testing.T) {
	// WithTimeout(0) is invalid; WithTimeout(30s) afterwards should never run.
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithTimeout(0),
		aura.WithTimeout(30*time.Second), // never reached
	)
	if err == nil {
		t.Fatal("expected error from first invalid option")
	}
	if err.Error() != "timeout must be greater than zero" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

// =============================================================================
// WithHTTPClient option
// =============================================================================

func TestBlackBox_WithHTTPClient_Nil_ReturnsError(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithHTTPClient(nil),
	)
	if err == nil {
		t.Fatal("expected error for nil HTTP client")
	}
	if err.Error() != "HTTP client cannot be nil" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_WithHTTPClient_NonNil_Accepted(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithHTTPClient(&http.Client{}),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// =============================================================================
// WithUserAgent option
// =============================================================================

func TestBlackBox_WithUserAgent_Empty_ReturnsError(t *testing.T) {
	_, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithUserAgent(""),
	)
	if err == nil {
		t.Fatal("expected error for empty user agent")
	}
	if err.Error() != "user agent must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestBlackBox_WithUserAgent_NonEmpty_Accepted(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithUserAgent("my-app/1.0"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// =============================================================================
// WithDefaultHeaders option
// =============================================================================

func TestBlackBox_WithDefaultHeaders_Nil_IsNoOp(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithDefaultHeaders(nil),
	)
	if err != nil {
		t.Fatalf("expected no error for nil headers, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_WithDefaultHeaders_Empty_IsNoOp(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithDefaultHeaders(map[string]string{}),
	)
	if err != nil {
		t.Fatalf("expected no error for empty headers, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_WithDefaultHeaders_ValidHeaders_Accepted(t *testing.T) {
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithDefaultHeaders(map[string]string{
			"X-Request-ID": "abc-123",
			"X-Tenant":     "my-tenant",
		}),
	)
	if err != nil {
		t.Fatalf("expected no error for valid headers, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_WithDefaultHeaders_ProtectedHeadersDropped(t *testing.T) {
	// Protected headers should be silently dropped; construction must succeed.
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithDefaultHeaders(map[string]string{
			"Authorization": "Bearer sneaky-token",
			"Content-Type":  "text/plain",
			"User-Agent":    "evil-agent/1.0",
		}),
	)
	if err != nil {
		t.Fatalf("expected no error (protected headers silently dropped), got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestBlackBox_WithDefaultHeaders_ProtectedHeadersCaseInsensitive(t *testing.T) {
	// Mixed-case variants of protected keys must also be dropped without error.
	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithDefaultHeaders(map[string]string{
			"authorization": "Bearer lower",
			"CONTENT-TYPE":  "text/html",
			"User-agent":    "mixed-case/1.0",
			"X-Custom":      "kept",
		}),
	)
	if err != nil {
		t.Fatalf("expected no error for mixed-case protected headers, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// =============================================================================
// WithDefaultHeaders — end-to-end: headers reach the mock HTTP server
// =============================================================================

// TestBlackBox_WithDefaultHeaders_ReachServer verifies that headers supplied via
// WithDefaultHeaders are present on every API request sent by the client.
// It uses an httptest.Server to act as a stand-in Aura API and inspects the
// incoming request headers directly.
func TestBlackBox_WithDefaultHeaders_ReachServer(t *testing.T) {
	const customHeaderKey = "X-Request-ID"
	const customHeaderVal = "test-req-42"

	// tokenResp is the minimal OAuth token response the client needs before it
	// will attempt any authenticated API call.
	tokenResp, _ := json.Marshal(map[string]any{
		"token_type":   "Bearer",
		"access_token": "test-token",
		"expires_in":   int64(3600),
	})

	var capturedCustomHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(tokenResp)
		default:
			// Capture the custom header from the authenticated API request.
			capturedCustomHeader = r.Header.Get(customHeaderKey)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	defer srv.Close()

	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithInsecureBaseURL(srv.URL),
		aura.WithDefaultHeaders(map[string]string{
			customHeaderKey: customHeaderVal,
		}),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Trigger a real API call so headers are sent to the server.
	_, _ = client.Instances.List(context.Background())

	if capturedCustomHeader != customHeaderVal {
		t.Errorf("expected %s header %q, got %q", customHeaderKey, customHeaderVal, capturedCustomHeader)
	}
}

// TestBlackBox_WithUserAgent_ReachServer verifies that the User-Agent override
// supplied via WithUserAgent reaches the server on every API request.
func TestBlackBox_WithUserAgent_ReachServer(t *testing.T) {
	const customUA = "my-app/2.0"

	tokenResp, _ := json.Marshal(map[string]any{
		"token_type":   "Bearer",
		"access_token": "test-token",
		"expires_in":   int64(3600),
	})

	var capturedUA string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(tokenResp)
		default:
			capturedUA = r.Header.Get("User-Agent")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	defer srv.Close()

	client, err := aura.NewClient(
		aura.WithCredentials("id", "secret"),
		aura.WithInsecureBaseURL(srv.URL),
		aura.WithUserAgent(customUA),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	_, _ = client.Instances.List(context.Background())

	if capturedUA != customUA {
		t.Errorf("expected User-Agent %q, got %q", customUA, capturedUA)
	}
}

// =============================================================================
// Close
// =============================================================================

func TestBlackBox_Close_DoesNotPanic(t *testing.T) {
	client := newTestClient(t)

	// Calling Close() once must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close() panicked: %v", r)
		}
	}()
	client.Close()
}

func TestBlackBox_Close_CalledTwiceDoesNotPanic(t *testing.T) {
	client := newTestClient(t)

	// Calling Close() a second time must not panic even though idle
	// connections are already drained.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("second Close() panicked: %v", r)
		}
	}()
	client.Close()
	client.Close()
}

// =============================================================================
// Service interface injection
//
// These tests confirm that exported service fields accept any implementation of
// the corresponding interface – the primary extensibility contract for consumers.
// =============================================================================

func TestBlackBox_ServiceInjection_Instances(t *testing.T) {
	client := newTestClient(t)
	var _ aura.InstanceService = (*mockInstanceService)(nil) // compile-time check

	mock := &mockInstanceService{
		ListResp: &aura.ListInstancesResponse{
			Data: []aura.ListInstanceData{{ID: "inst-1", Name: "test"}},
		},
	}
	client.Instances = mock

	result, err := client.Instances.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "inst-1" {
		t.Errorf("unexpected result: %+v", result)
	}
	if mock.CallCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.CallCount)
	}
}

func TestBlackBox_ServiceInjection_Tenants(t *testing.T) {
	client := newTestClient(t)
	var _ aura.TenantService = (*mockTenantService)(nil) // compile-time check

	mock := &mockTenantService{
		ListResp: &aura.ListTenantsResponse{
			Data: []aura.TenantsResponseData{{ID: "tenant-1", Name: "my-tenant"}},
		},
	}
	client.Tenants = mock

	result, err := client.Tenants.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "tenant-1" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestBlackBox_ServiceInjection_Snapshots(t *testing.T) {
	client := newTestClient(t)
	var _ aura.SnapshotService = (*mockSnapshotService)(nil) // compile-time check

	mock := &mockSnapshotService{
		CreateResp: &aura.CreateSnapshotResponse{Data: aura.CreateSnapshotData{SnapshotID: "snap-1"}},
	}
	client.Snapshots = mock

	result, err := client.Snapshots.Create(context.Background(), "instance-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.SnapshotID != "snap-1" {
		t.Errorf("expected snapshot ID snap-1, got %s", result.Data.SnapshotID)
	}
}

func TestBlackBox_ServiceInjection_Cmek(t *testing.T) {
	client := newTestClient(t)
	var _ aura.CMEKService = (*mockCmekService)(nil) // compile-time check

	mock := &mockCmekService{
		ListResp: &aura.GetCMEKsResponse{
			Data: []aura.GetCMEKsData{{ID: "cmek-1", Name: "my-key", TenantID: "tenant-1"}},
		},
	}
	client.CMEK = mock

	result, err := client.CMEK.List(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "cmek-1" {
		t.Errorf("unexpected result: %+v", result)
	}
	if mock.LastTenantID != "tenant-1" {
		t.Errorf("expected tenant ID tenant-1, got %s", mock.LastTenantID)
	}
}

func TestBlackBox_ServiceInjection_GraphAnalytics(t *testing.T) {
	client := newTestClient(t)
	var _ aura.GDSSessionService = (*mockGDSSessionService)(nil) // compile-time check

	mock := &mockGDSSessionService{
		ListResp: &aura.GetGDSSessionListResponse{
			Data: []aura.GetGDSSessionData{{ID: "session-1", Name: "my-session"}},
		},
	}
	client.GraphAnalytics = mock

	result, err := client.GraphAnalytics.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "session-1" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestBlackBox_ServiceInjection_Prometheus(t *testing.T) {
	client := newTestClient(t)
	var _ aura.PrometheusService = (*mockPrometheusService)(nil) // compile-time check

	mock := &mockPrometheusService{
		HealthResp: &aura.PrometheusHealthMetrics{
			InstanceID:    "inst-1",
			OverallStatus: "healthy",
		},
	}
	client.Prometheus = mock

	result, err := client.Prometheus.GetInstanceHealth(context.Background(), "inst-1", "https://metrics.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OverallStatus != "healthy" {
		t.Errorf("expected healthy status, got %s", result.OverallStatus)
	}
	if mock.LastInstanceID != "inst-1" {
		t.Errorf("expected instance ID inst-1, got %s", mock.LastInstanceID)
	}
}

// =============================================================================
// Instance service operations
// =============================================================================

func TestBlackBox_Instances_List_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockInstanceService{
		ListResp: &aura.ListInstancesResponse{
			Data: []aura.ListInstanceData{
				{ID: "a1b2c3d4", Name: "prod-db", TenantID: "t1", CloudProvider: "gcp"},
				{ID: "e5f6a7b8", Name: "dev-db", TenantID: "t1", CloudProvider: "aws"},
			},
		},
	}
	client.Instances = mock

	result, err := client.Instances.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(result.Data))
	}
	if result.Data[0].Name != "prod-db" {
		t.Errorf("unexpected first instance: %+v", result.Data[0])
	}
	if mock.LastMethod != "List" {
		t.Errorf("expected method List, got %s", mock.LastMethod)
	}
}

func TestBlackBox_Instances_List_Empty(t *testing.T) {
	client := newTestClient(t)
	client.Instances = &mockInstanceService{
		ListResp: &aura.ListInstancesResponse{Data: []aura.ListInstanceData{}},
	}

	result, err := client.Instances.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result.Data))
	}
}

func TestBlackBox_Instances_List_Error(t *testing.T) {
	client := newTestClient(t)
	apiErr := &aura.Error{StatusCode: 401, Message: "Unauthorized"}
	client.Instances = &mockInstanceService{ListErr: apiErr}

	_, err := client.Instances.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var gotErr *aura.Error
	if !errors.As(err, &gotErr) {
		t.Fatalf("expected *aura.Error, got %T", err)
	}
	if !gotErr.IsUnauthorized() {
		t.Errorf("expected IsUnauthorized() = true, status = %d", gotErr.StatusCode)
	}
}

func TestBlackBox_Instances_Get_Success(t *testing.T) {
	client := newTestClient(t)
	const instanceID = "aabbccdd"
	client.Instances = &mockInstanceService{
		GetResp: &aura.GetInstanceResponse{
			Data: aura.InstanceData{
				ID:     instanceID,
				Name:   "my-db",
				Status: "running",
			},
		},
	}

	result, err := client.Instances.Get(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != instanceID {
		t.Errorf("expected ID %s, got %s", instanceID, result.Data.ID)
	}
	if result.Data.Status != "running" {
		t.Errorf("expected status %s, got %s", aura.StatusRunning, result.Data.Status)
	}
}

func TestBlackBox_Instances_Get_NotFound(t *testing.T) {
	client := newTestClient(t)
	client.Instances = &mockInstanceService{
		GetErr: &aura.Error{StatusCode: 404, Message: "Instance not found"},
	}

	_, err := client.Instances.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *aura.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *aura.Error, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("expected IsNotFound() = true, status = %d", apiErr.StatusCode)
	}
}

func TestBlackBox_Instances_Create_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockInstanceService{
		CreateResp: &aura.CreateInstanceResponse{
			Data: aura.CreateInstanceData{
				ID:       "new-inst",
				Name:     "test-create",
				Password: "s3cr3t",
			},
		},
	}
	client.Instances = mock

	req := &aura.CreateInstanceConfigData{
		Name:          "test-create",
		TenantID:      "tenant-1",
		CloudProvider: "gcp",
		Region:        "europe-west1",
		Type:          "enterprise-db",
		Version:       "5",
		Memory:        "4GB",
	}
	result, err := client.Instances.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "new-inst" {
		t.Errorf("expected ID new-inst, got %s", result.Data.ID)
	}
	if result.Data.Password == "" {
		t.Error("expected password to be populated")
	}
	if mock.LastCreateReq == nil || mock.LastCreateReq.Name != "test-create" {
		t.Errorf("create request not forwarded correctly")
	}
}

func TestBlackBox_Instances_Delete_Success(t *testing.T) {
	client := newTestClient(t)
	const instanceID = "ddeeff00"
	mock := &mockInstanceService{
		DeleteResp: &aura.DeleteInstanceResponse{
			Data: aura.InstanceData{ID: instanceID, Status: "destroying"},
		},
	}
	client.Instances = mock

	result, err := client.Instances.Delete(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != "destroying" {
		t.Errorf("expected status destroying, got %s", result.Data.Status)
	}
	if mock.LastInstanceID != instanceID {
		t.Errorf("expected instanceID %s, got %s", instanceID, mock.LastInstanceID)
	}
}

func TestBlackBox_Instances_Pause_Success(t *testing.T) {
	client := newTestClient(t)
	const instanceID = "11223344"
	client.Instances = &mockInstanceService{
		PauseResp: &aura.GetInstanceResponse{
			Data: aura.InstanceData{ID: instanceID, Status: aura.StatusPaused},
		},
	}

	result, err := client.Instances.Pause(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != aura.StatusPaused {
		t.Errorf("expected status paused, got %s", result.Data.Status)
	}
}

func TestBlackBox_Instances_Resume_Success(t *testing.T) {
	client := newTestClient(t)
	const instanceID = "55667788"
	client.Instances = &mockInstanceService{
		ResumeResp: &aura.GetInstanceResponse{
			Data: aura.InstanceData{ID: instanceID, Status: aura.StatusRunning},
		},
	}

	result, err := client.Instances.Resume(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != aura.StatusRunning {
		t.Errorf("expected status running, got %s", result.Data.Status)
	}
}

func TestBlackBox_Instances_Update_Success(t *testing.T) {
	client := newTestClient(t)
	const instanceID = "99aabbcc"
	mock := &mockInstanceService{
		UpdateResp: &aura.GetInstanceResponse{
			Data: aura.InstanceData{ID: instanceID, Name: "renamed", Memory: "16GB"},
		},
	}
	client.Instances = mock

	result, err := client.Instances.Update(
		context.Background(), instanceID,
		&aura.UpdateInstanceData{Name: "renamed", Memory: "16GB"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Name != "renamed" {
		t.Errorf("expected name renamed, got %s", result.Data.Name)
	}
	if result.Data.Memory != "16GB" {
		t.Errorf("expected memory 16GB, got %s", result.Data.Memory)
	}
	if mock.LastInstanceID != instanceID {
		t.Errorf("instance ID not forwarded correctly")
	}
}

func TestBlackBox_Instances_Overwrite_WithSourceInstance(t *testing.T) {
	client := newTestClient(t)
	mock := &mockInstanceService{
		OverwriteResp: &aura.OverwriteInstanceResponse{Data: "overwrite-job-1"},
	}
	client.Instances = mock

	result, err := client.Instances.OverwriteFromInstance(context.Background(), "ddeeff00", "aabbccdd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data == "" {
		t.Error("expected a job ID in response")
	}
	if mock.LastSourceInstanceID != "aabbccdd" {
		t.Errorf("source instance ID not forwarded correctly")
	}
	if mock.LastSourceSnapshotID != "" {
		t.Errorf("snapshot ID should be empty, got %s", mock.LastSourceSnapshotID)
	}
}

func TestBlackBox_Instances_Overwrite_WithSnapshot(t *testing.T) {
	client := newTestClient(t)
	mock := &mockInstanceService{
		OverwriteResp: &aura.OverwriteInstanceResponse{Data: "overwrite-job-2"},
	}
	client.Instances = mock

	_, err := client.Instances.OverwriteFromSnapshot(context.Background(), "ddeeff00", "snap-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.LastSourceSnapshotID != "snap-abc" {
		t.Errorf("snapshot ID not forwarded correctly")
	}
}

// =============================================================================
// Tenant service operations
// =============================================================================

func TestBlackBox_Tenants_List_Success(t *testing.T) {
	client := newTestClient(t)
	client.Tenants = &mockTenantService{
		ListResp: &aura.ListTenantsResponse{
			Data: []aura.TenantsResponseData{
				{ID: "t1", Name: "Production"},
				{ID: "t2", Name: "Staging"},
			},
		},
	}

	result, err := client.Tenants.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(result.Data))
	}
}

func TestBlackBox_Tenants_Get_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockTenantService{
		GetResp: &aura.GetTenantResponse{
			Data: aura.TenantResponseData{ID: "t1", Name: "Production"},
		},
	}
	client.Tenants = mock

	result, err := client.Tenants.Get(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "t1" {
		t.Errorf("expected ID t1, got %s", result.Data.ID)
	}
	if mock.LastTenantID != "t1" {
		t.Errorf("tenant ID not forwarded correctly")
	}
}

func TestBlackBox_Tenants_GetMetrics_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockTenantService{
		GetMetricsResp: &aura.GetTenantMetricsURLResponse{
			Data: aura.GetTenantMetricsURLData{Endpoint: "https://metrics.neo4j.io/t1"},
		},
	}
	client.Tenants = mock

	result, err := client.Tenants.GetMetrics(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Endpoint == "" {
		t.Error("expected a non-empty endpoint URL")
	}
}

// =============================================================================
// Snapshot service operations
// =============================================================================

func TestBlackBox_Snapshots_List_Success(t *testing.T) {
	client := newTestClient(t)
	client.Snapshots = &mockSnapshotService{
		ListResp: &aura.GetSnapshotsResponse{
			Data: []aura.GetSnapshotData{
				{SnapshotID: "snap-1", InstanceID: "inst-1", Status: "Completed"},
				{SnapshotID: "snap-2", InstanceID: "inst-1", Status: "Completed"},
			},
		},
	}

	result, err := client.Snapshots.List(context.Background(), "inst-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(result.Data))
	}
}

func TestBlackBox_Snapshots_List_WithDateFilter(t *testing.T) {
	client := newTestClient(t)
	mock := &mockSnapshotService{
		ListResp: &aura.GetSnapshotsResponse{Data: []aura.GetSnapshotData{}},
	}
	client.Snapshots = mock

	filter := aura.SnapshotDate{Year: 2024, Month: time.January, Day: 01}
	_, err := client.Snapshots.List(context.Background(), "inst-1", &filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.LastDate != &filter {
		t.Errorf("date filter not forwarded: %d-%s-%d", mock.LastDate.Year, mock.LastDate.Month.String(), mock.LastDate.Day)
	}
}

func TestBlackBox_Snapshots_Create_Success(t *testing.T) {
	client := newTestClient(t)
	client.Snapshots = &mockSnapshotService{
		CreateResp: &aura.CreateSnapshotResponse{Data: aura.CreateSnapshotData{SnapshotID: "new-snap"}},
	}

	result, err := client.Snapshots.Create(context.Background(), "inst-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.SnapshotID != "new-snap" {
		t.Errorf("expected snapshot ID new-snap, got %s", result.Data.SnapshotID)
	}
}

func TestBlackBox_Snapshots_Get_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockSnapshotService{
		GetResp: &aura.GetSnapshotDataResponse{
			Data: aura.GetSnapshotData{SnapshotID: "snap-1", Status: "Completed", Exportable: true},
		},
	}
	client.Snapshots = mock

	result, err := client.Snapshots.Get(context.Background(), "inst-1", "snap-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Data.Exportable {
		t.Error("expected exportable = true")
	}
	if mock.LastSnapshotID != "snap-1" {
		t.Errorf("snapshot ID not forwarded correctly")
	}
}

func TestBlackBox_Snapshots_Restore_Success(t *testing.T) {
	client := newTestClient(t)
	client.Snapshots = &mockSnapshotService{
		RestoreResp: &aura.RestoreSnapshotResponse{
			Data: aura.InstanceData{ID: "inst-1", Status: "restoring"},
		},
	}

	result, err := client.Snapshots.Restore(context.Background(), "inst-1", "snap-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != "restoring" {
		t.Errorf("expected status restoring, got %s", result.Data.Status)
	}
}

// =============================================================================
// CMEK service operations
// =============================================================================

func TestBlackBox_Cmek_List_Success(t *testing.T) {
	client := newTestClient(t)
	client.CMEK = &mockCmekService{
		ListResp: &aura.GetCMEKsResponse{
			Data: []aura.GetCMEKsData{
				{ID: "k1", Name: "key-one", TenantID: "t1"},
			},
		},
	}

	result, err := client.CMEK.List(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "k1" {
		t.Errorf("unexpected CMEK result: %+v", result)
	}
}

func TestBlackBox_Cmek_List_Error(t *testing.T) {
	client := newTestClient(t)
	client.CMEK = &mockCmekService{
		ListErr: &aura.Error{StatusCode: 403, Message: "Forbidden"},
	}

	_, err := client.CMEK.List(context.Background(), "t1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// =============================================================================
// Graph Analytics (GDS session) service operations
// =============================================================================

func TestBlackBox_GraphAnalytics_List_Success(t *testing.T) {
	client := newTestClient(t)
	client.GraphAnalytics = &mockGDSSessionService{
		ListResp: &aura.GetGDSSessionListResponse{
			Data: []aura.GetGDSSessionData{
				{ID: "sess-1", Name: "analysis-run", Status: "Ready"},
			},
		},
	}

	result, err := client.GraphAnalytics.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 session, got %d", len(result.Data))
	}
}

func TestBlackBox_GraphAnalytics_Estimate_Success(t *testing.T) {
	client := newTestClient(t)
	client.GraphAnalytics = &mockGDSSessionService{
		EstimateResp: &aura.GDSSessionSizeEstimationResponse{
			Data: aura.GDSSessionSizeEstimationData{
				EstimatedMemory: "8GB",
				RecommendedSize: "enterprise-db",
			},
		},
	}

	result, err := client.GraphAnalytics.Estimate(context.Background(), &aura.GetGDSSessionSizeEstimation{
		NodeCount:         100000,
		RelationshipCount: 500000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.EstimatedMemory != "8GB" {
		t.Errorf("unexpected estimated memory: %s", result.Data.EstimatedMemory)
	}
}

func TestBlackBox_GraphAnalytics_Create_Success(t *testing.T) {
	client := newTestClient(t)
	client.GraphAnalytics = &mockGDSSessionService{
		CreateResp: &aura.GetGDSSessionResponse{
			Data: aura.GetGDSSessionData{ID: "sess-new", Status: "Creating"},
		},
	}

	result, err := client.GraphAnalytics.Create(context.Background(), &aura.CreateGDSSessionConfigData{
		Name:       "new-session",
		InstanceID: "inst-1",
		Memory:     "8GB",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "sess-new" {
		t.Errorf("unexpected create response: %+v", result)
	}
}

func TestBlackBox_GraphAnalytics_Get_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockGDSSessionService{
		GetResp: &aura.GetGDSSessionResponse{
			Data: aura.GetGDSSessionData{ID: "sess-1", Status: "Ready"},
		},
	}
	client.GraphAnalytics = mock

	result, err := client.GraphAnalytics.Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != "Ready" {
		t.Errorf("unexpected session status: %+v", result)
	}
	if mock.LastSessionID != "sess-1" {
		t.Errorf("session ID not forwarded correctly")
	}
}

func TestBlackBox_GraphAnalytics_Delete_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockGDSSessionService{
		DeleteResp: &aura.DeleteGDSSessionResponse{Data: aura.DeleteGDSSession{ID: "sess-1"}},
	}
	client.GraphAnalytics = mock

	result, err := client.GraphAnalytics.Delete(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "sess-1" {
		t.Errorf("expected deleted session ID sess-1, got %s", result.Data.ID)
	}
}

// =============================================================================
// Prometheus service operations
// =============================================================================

func TestBlackBox_Prometheus_FetchRawMetrics_Success(t *testing.T) {
	client := newTestClient(t)
	client.Prometheus = &mockPrometheusService{
		FetchResp: &aura.PrometheusMetricsResponse{
			Metrics: map[string][]aura.PrometheusMetric{
				"neo4j_bolt_connections_running": {
					{Name: "neo4j_bolt_connections_running", Value: 5},
				},
			},
		},
	}

	result, err := client.Prometheus.FetchRawMetrics(context.Background(), "https://metrics.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Metrics == nil {
		t.Fatal("expected non-nil metrics response")
	}
	if _, ok := result.Metrics["neo4j_bolt_connections_running"]; !ok {
		t.Error("expected bolt connections metric")
	}
}

func TestBlackBox_Prometheus_GetMetricValue_Success(t *testing.T) {
	client := newTestClient(t)
	mock := &mockPrometheusService{GetValResp: 42.5}
	client.Prometheus = mock

	metricsResp := &aura.PrometheusMetricsResponse{Metrics: map[string][]aura.PrometheusMetric{}}
	val, err := client.Prometheus.GetMetricValue(context.Background(), metricsResp, "some_metric", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42.5 {
		t.Errorf("expected 42.5, got %f", val)
	}
	if mock.LastMetricName != "some_metric" {
		t.Errorf("metric name not forwarded correctly")
	}
}

func TestBlackBox_Prometheus_GetInstanceHealth_Success(t *testing.T) {
	client := newTestClient(t)
	client.Prometheus = &mockPrometheusService{
		HealthResp: &aura.PrometheusHealthMetrics{
			InstanceID:    "inst-1",
			OverallStatus: "healthy",
			Resources: aura.ResourceMetrics{
				CPUUsagePercent:    12.5,
				MemoryUsagePercent: 45.0,
			},
		},
	}

	result, err := client.Prometheus.GetInstanceHealth(context.Background(), "inst-1", "https://metrics.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OverallStatus != "healthy" {
		t.Errorf("expected healthy, got %s", result.OverallStatus)
	}
	if result.Resources.CPUUsagePercent != 12.5 {
		t.Errorf("unexpected CPU usage: %f", result.Resources.CPUUsagePercent)
	}
}

// =============================================================================
// Error type handling
//
// These tests verify the consumer-facing error contract: errors should be
// type-assertable to *aura.Error and carry useful diagnostic methods.
// =============================================================================

func TestBlackBox_Error_TypeAssertion_WithErrorsAs(t *testing.T) {
	client := newTestClient(t)
	client.Instances = &mockInstanceService{
		GetErr: &aura.Error{StatusCode: 404, Message: "Instance not found"},
	}

	_, err := client.Instances.Get(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *aura.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As failed; got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestBlackBox_Error_IsNotFound(t *testing.T) {
	err := &aura.Error{StatusCode: 404, Message: "Not found"}
	if !err.IsNotFound() {
		t.Error("expected IsNotFound() = true")
	}

	err404 := &aura.Error{StatusCode: 200}
	if err404.IsNotFound() {
		t.Error("expected IsNotFound() = false for 200")
	}
}

func TestBlackBox_Error_IsUnauthorized(t *testing.T) {
	err := &aura.Error{StatusCode: 401, Message: "Unauthorized"}
	if !err.IsUnauthorized() {
		t.Error("expected IsUnauthorized() = true")
	}

	err200 := &aura.Error{StatusCode: 200}
	if err200.IsUnauthorized() {
		t.Error("expected IsUnauthorized() = false for 200")
	}
}

func TestBlackBox_Error_IsBadRequest(t *testing.T) {
	err := &aura.Error{StatusCode: 400, Message: "Bad request"}
	if !err.IsBadRequest() {
		t.Error("expected IsBadRequest() = true")
	}
}

func TestBlackBox_Error_HasMultipleErrors_False(t *testing.T) {
	err := &aura.Error{
		StatusCode: 400,
		Message:    "Single error",
		Details:    []aura.ErrorDetail{{Message: "one detail"}},
	}
	if err.HasMultipleErrors() {
		t.Error("expected HasMultipleErrors() = false for single detail")
	}
}

func TestBlackBox_Error_HasMultipleErrors_True(t *testing.T) {
	err := &aura.Error{
		StatusCode: 422,
		Message:    "Validation failed",
		Details: []aura.ErrorDetail{
			{Message: "field A required"},
			{Message: "field B invalid"},
		},
	}
	if !err.HasMultipleErrors() {
		t.Error("expected HasMultipleErrors() = true for two details")
	}
}

func TestBlackBox_Error_AllErrors(t *testing.T) {
	err := &aura.Error{
		StatusCode: 422,
		Message:    "Root error",
		Details: []aura.ErrorDetail{
			{Message: "sub-error one"},
			{Message: "sub-error two"},
		},
	}

	all := err.AllErrors()
	// AllErrors returns the top-level message plus each detail message
	if len(all) != 3 {
		t.Fatalf("expected 3 messages (1 root + 2 details), got %d", len(all))
	}
	if all[0] != "Root error" {
		t.Errorf("first entry should be root message, got %q", all[0])
	}
}

func TestBlackBox_Error_Message_WithoutDetails(t *testing.T) {
	err := &aura.Error{StatusCode: 503, Message: "Service unavailable"}
	msg := err.Error()
	if msg != "API error (status 503): Service unavailable" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestBlackBox_Error_Message_WithSingleDetail(t *testing.T) {
	err := &aura.Error{
		StatusCode: 400,
		Message:    "Bad request",
		Details:    []aura.ErrorDetail{{Message: "name is required"}},
	}
	msg := err.Error()
	if msg != "API error (status 400): Bad request - name is required" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestBlackBox_ErrorDetail_Fields(t *testing.T) {
	detail := aura.ErrorDetail{
		Message: "must be positive",
		Reason:  "invalid_value",
		Field:   "memory",
	}
	if detail.Message != "must be positive" {
		t.Error("Message field not set correctly")
	}
	if detail.Reason != "invalid_value" {
		t.Error("Reason field not set correctly")
	}
	if detail.Field != "memory" {
		t.Error("Field field not set correctly")
	}
}

// =============================================================================
// Exported constants
// =============================================================================

func TestBlackBox_Constants_StatusValues(t *testing.T) {
	tests := []struct {
		name     string
		constant aura.InstanceStatus
		expected string
	}{
		{"StatusRunning", aura.StatusRunning, "running"},
		{"StatusStopped", aura.StatusStopped, "stopped"},
		{"StatusPaused", aura.StatusPaused, "paused"},
		{"StatusAvailable", aura.StatusAvailable, "available"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

func TestBlackBox_Constants_ClientVersion_NonEmpty(t *testing.T) {
	if aura.ClientVersion == "" {
		t.Error("ClientVersion must not be empty")
	}
}

// =============================================================================
// Context propagation through injected mocks
// =============================================================================

func TestBlackBox_Context_AlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before any call

	// The mock does not check context itself; we rely on the real
	// instanceService (beneath the interface) for context enforcement.
	// Here we verify that a service mock honouring a cancelled context returns
	// an appropriate error, as a real implementation would.
	client := newTestClient(t)
	client.Instances = &mockCancelAwareInstanceService{}

	_, err := client.Instances.List(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestBlackBox_Context_Deadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // ensure deadline has passed

	client := newTestClient(t)
	client.Instances = &mockCancelAwareInstanceService{}

	_, err := client.Instances.List(ctx)
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected a context error, got: %v", err)
	}
}

// mockCancelAwareInstanceService is a minimal InstanceService that checks the
// context before doing any work – simulating correct consumer-side behaviour.
type mockCancelAwareInstanceService struct{}

func (m *mockCancelAwareInstanceService) List(ctx context.Context) (*aura.ListInstancesResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.ListInstancesResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Get(ctx context.Context, _ string) (*aura.GetInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.GetInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Create(ctx context.Context, _ *aura.CreateInstanceConfigData) (*aura.CreateInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.CreateInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Delete(ctx context.Context, _ string) (*aura.DeleteInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.DeleteInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Pause(ctx context.Context, _ string) (*aura.GetInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.GetInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Resume(ctx context.Context, _ string) (*aura.GetInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.GetInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) Update(ctx context.Context, _ string, _ *aura.UpdateInstanceData) (*aura.GetInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.GetInstanceResponse{}, nil
}
func (m *mockCancelAwareInstanceService) OverwriteFromInstance(ctx context.Context, _, _ string) (*aura.OverwriteInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.OverwriteInstanceResponse{}, nil
}

func (m *mockCancelAwareInstanceService) OverwriteFromSnapshot(ctx context.Context, _, _ string) (*aura.OverwriteInstanceResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &aura.OverwriteInstanceResponse{}, nil
}
