package v2beta1

import (
	"testing"
	"time"
)

// TestNewClient_Success verifies that NewClient constructs successfully with
// valid credentials and that all service fields are non-nil.
func TestNewClient_Success(t *testing.T) {
	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
		return
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
	if client.Instances == nil {
		t.Error("expected Instances service to be initialized")
	}
	if client.Databases == nil {
		t.Error("expected Databases service to be initialized")
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
