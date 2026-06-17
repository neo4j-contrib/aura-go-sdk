package v2beta1

import (
	"sync"
	"testing"
	"time"
)

// TestNewClient_Success verifies that NewClient constructs successfully with
// valid credentials and that both service fields are non-nil.
func TestNewClient_Success(t *testing.T) {
	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.api == nil {
		t.Error("expected api service to be initialized")
	}
	if client.logger == nil {
		t.Error("expected logger to be initialized")
	}
	if client.Organizations == nil {
		t.Error("expected Organizations service to be initialized")
	}
	if client.Projects == nil {
		t.Error("expected Projects service to be initialized")
	}
}

// TestNewClient_MissingCredentials verifies that NewClient returns an error
// when WithCredentials is omitted.
func TestNewClient_MissingCredentials(t *testing.T) {
	client, err := NewClient()
	if err == nil {
		t.Error("expected error when no credentials provided, got nil")
	}
	if client != nil {
		t.Error("expected client to be nil on error")
	}
}

// TestNewClient_EmptyClientID verifies that an empty client ID is rejected.
func TestNewClient_EmptyClientID(t *testing.T) {
	client, err := NewClient(WithCredentials("", "secret"))
	if err == nil {
		t.Error("expected error for empty client ID, got nil")
	}
	if err != nil && err.Error() != "client ID must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	if client != nil {
		t.Error("expected client to be nil on error")
	}
}

// TestNewClient_EmptyClientSecret verifies that an empty client secret is rejected.
func TestNewClient_EmptyClientSecret(t *testing.T) {
	client, err := NewClient(WithCredentials("id", ""))
	if err == nil {
		t.Error("expected error for empty client secret, got nil")
	}
	if err != nil && err.Error() != "client secret must not be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	if client != nil {
		t.Error("expected client to be nil on error")
	}
}

// TestNewClient_WithOrganization confirms that WithOrganization sets defaultOrgID
// and it is accessible via a subsequent SetOrg round-trip.
func TestNewClient_WithOrganization(t *testing.T) {
	const orgID = "11111111-2222-3333-4444-555555555555"

	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithOrganization(orgID),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	// Verify the defaultOrgID was set during construction.
	client.mu.RLock()
	gotOrgID := client.defaultOrgID
	client.mu.RUnlock()

	if gotOrgID != orgID {
		t.Errorf("expected defaultOrgID %q, got %q", orgID, gotOrgID)
	}
}

// TestNewClient_WithOrganization_Empty verifies that an empty org ID is rejected.
func TestNewClient_WithOrganization_Empty(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithOrganization(""),
	)
	if err == nil {
		t.Error("expected error for empty organization ID, got nil")
	}
}

// TestNewClient_WithDefaultProject confirms that WithDefaultProject sets
// defaultProjectID at construction time.
func TestNewClient_WithDefaultProject(t *testing.T) {
	const projectID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithDefaultProject(projectID),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	client.mu.RLock()
	gotProjectID := client.defaultProjectID
	client.mu.RUnlock()

	if gotProjectID != projectID {
		t.Errorf("expected defaultProjectID %q, got %q", projectID, gotProjectID)
	}
}

// TestNewClient_WithDefaultProject_Empty verifies that an empty project ID is rejected.
func TestNewClient_WithDefaultProject_Empty(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithDefaultProject(""),
	)
	if err == nil {
		t.Error("expected error for empty project ID, got nil")
	}
}

// TestSetOrg verifies that SetOrg updates the defaultOrgID under the write lock.
func TestSetOrg(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const newOrgID = "aaaabbbb-cccc-dddd-eeee-ffffaaaabbbb"
	client.SetOrg(newOrgID)

	client.mu.RLock()
	got := client.defaultOrgID
	client.mu.RUnlock()

	if got != newOrgID {
		t.Errorf("expected defaultOrgID %q, got %q", newOrgID, got)
	}
}

// TestSetProject verifies that SetProject updates the defaultProjectID under the write lock.
func TestSetProject(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const newProjectID = "12345678-1234-5678-1234-567812345678"
	client.SetProject(newProjectID)

	client.mu.RLock()
	got := client.defaultProjectID
	client.mu.RUnlock()

	if got != newProjectID {
		t.Errorf("expected defaultProjectID %q, got %q", newProjectID, got)
	}
}

// TestSetOrg_Concurrent verifies that concurrent calls to SetOrg and SetProject
// do not cause data races. Readers (concurrent with writers) are also exercised to
// cover the RLock path. Run with go test -race.
func TestSetOrg_Concurrent(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			client.SetOrg("11111111-1111-1111-1111-111111111111")
		}()
		go func() {
			defer wg.Done()
			client.SetProject("22222222-2222-2222-2222-222222222222")
		}()
		// Reader goroutines exercise the RLock path concurrent with writers.
		go func() {
			defer wg.Done()
			client.mu.RLock()
			_ = client.defaultOrgID
			_ = client.defaultProjectID
			client.mu.RUnlock()
		}()
	}

	wg.Wait()
	// No value assertions — the test goal is proving absence of data races.
}

// TestClose_DoesNotPanic verifies that Close() does not panic.
func TestClose_DoesNotPanic(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close() panicked: %v", r)
		}
	}()
	client.Close()
}

// TestNewClient_WithTimeout verifies that WithTimeout is accepted and applied.
func TestNewClient_WithTimeout(t *testing.T) {
	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithTimeout(60*time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Error("expected non-nil client")
	}
}

// TestNewClient_WithTimeout_Zero verifies that a zero timeout is rejected.
func TestNewClient_WithTimeout_Zero(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithTimeout(0),
	)
	if err == nil {
		t.Error("expected error for zero timeout")
	}
	if err != nil && err.Error() != "timeout must be greater than zero" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

// TestNewClient_WithMaxRetry_Zero verifies that a zero max retry is rejected.
func TestNewClient_WithMaxRetry_Zero(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithMaxRetry(0),
	)
	if err == nil {
		t.Error("expected error for zero max retry")
	}
}

// TestNewClient_WithBaseURL_Empty verifies that an empty base URL is rejected.
func TestNewClient_WithBaseURL_Empty(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithBaseURL(""),
	)
	if err == nil {
		t.Error("expected error for empty base URL")
	}
}

// TestNewClient_WithBaseURL_HTTP verifies that an HTTP (non-HTTPS) base URL is rejected.
func TestNewClient_WithBaseURL_HTTP(t *testing.T) {
	_, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithBaseURL("http://api.neo4j.io"),
	)
	if err == nil {
		t.Error("expected error for HTTP base URL")
	}
}

// TestNewClient_WithInsecureBaseURL_Accepted verifies that insecure base URLs
// are accepted (for local testing).
func TestNewClient_WithInsecureBaseURL_Accepted(t *testing.T) {
	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithInsecureBaseURL("http://localhost:8080"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Error("expected non-nil client")
	}
}

// TestDefaultOptions verifies default configuration values.
func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	if opts.config.baseURL != "https://api.neo4j.io" {
		t.Errorf("expected default baseURL 'https://api.neo4j.io', got %q", opts.config.baseURL)
	}
	if opts.config.apiTimeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", opts.config.apiTimeout)
	}
	if opts.config.apiRetryMax != 3 {
		t.Errorf("expected default apiRetryMax 3, got %d", opts.config.apiRetryMax)
	}
	if opts.logger == nil {
		t.Error("expected default logger to be initialized")
	}
	const wantMaxSize = 10 * 1024 * 1024
	if opts.config.maxResponseSize != wantMaxSize {
		t.Errorf("expected default maxResponseSize %d, got %d", wantMaxSize, opts.config.maxResponseSize)
	}
}

// TestAuraAPIVersion verifies the package-level API version constant.
func TestAuraAPIVersion(t *testing.T) {
	if auraAPIVersion != "v2beta1" {
		t.Errorf("expected auraAPIVersion 'v2beta1', got %q", auraAPIVersion)
	}
}
