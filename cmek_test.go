package aura

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
)

// createTestCmekService creates a cmekService with a mock API service for testing
func createTestCmekService(mock *mockAPIService) *cmekService {
	return &cmekService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestCmekServiceWithTimeout creates a cmekService with a specific timeout.
// Pass the desired context directly to each method call.
func createTestCmekServiceWithTimeout(mock api.RequestService, timeout time.Duration) *cmekService {
	return &cmekService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// TestCmekService_List_Success verifies successful CMEK listing
func TestCmekService_List_Success(t *testing.T) {
	expectedResponse := GetCmeksResponse{
		Data: []GetCmeksData{
			{ID: "cmek-1", Name: "Production Key", TenantID: "tenant-1"},
			{ID: "cmek-2", Name: "Development Key", TenantID: "tenant-1"},
			{ID: "cmek-3", Name: "Testing Key", TenantID: "tenant-2"},
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestCmekService(mock)
	result, err := service.List(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	if mock.lastPath != "customer-managed-keys" {
		t.Errorf("expected path 'customer-managed-keys', got '%s'", mock.lastPath)
	}
	if len(result.Data) != 3 {
		t.Errorf("expected 3 CMEKs, got %d", len(result.Data))
	}
}

// TestCmekService_List_WithTenantFilter verifies tenant ID filtering
func TestCmekService_List_WithTenantFilter(t *testing.T) {
	TenantID := "c1e2c556-a924-5fac-b7f8-bb624ad9761d"
	responseBody, _ := json.Marshal(GetCmeksResponse{
		Data: []GetCmeksData{
			{ID: "cmek-filtered-1", Name: "Filtered Key 1", TenantID: TenantID},
		},
	})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestCmekService(mock)
	result, err := service.List(context.Background(), TenantID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastPath != "customer-managed-keys?tenant_id="+TenantID {
		t.Errorf("expected path with tenant filter, got '%s'", mock.lastPath)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 CMEK, got %d", len(result.Data))
	}
}

// TestCmekService_List_InvalidTenantID verifies tenant ID validation
func TestCmekService_List_InvalidTenantID(t *testing.T) {
	tests := []struct {
		name     string
		TenantID string
	}{
		{"too short", "abc"},
		{"wrong length", "not-valid-uuid"},
	}

	mock := &mockAPIService{}
	service := createTestCmekService(mock)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.List(context.Background(), tt.TenantID)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

// TestCmekService_List_EmptyResult verifies empty CMEK list
func TestCmekService_List_EmptyResult(t *testing.T) {
	responseBody, _ := json.Marshal(GetCmeksResponse{Data: []GetCmeksData{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestCmekService(mock)
	result, err := service.List(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 CMEKs, got %d", len(result.Data))
	}
}

// TestCmekService_List_AuthenticationError verifies auth error handling
func TestCmekService_List_AuthenticationError(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: http.StatusUnauthorized, Message: "Invalid credentials"},
	}

	service := createTestCmekService(mock)
	_, err := service.List(context.Background(), "")

	if err == nil {
		t.Fatal("expected authentication error")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatal("expected Error type")
	}
	if !apiErr.IsUnauthorized() {
		t.Error("expected IsUnauthorized() to be true")
	}
}

// ============================================================================
// Context-Specific Tests for CmekService
// ============================================================================

// TestCmekService_List_ContextCancelled verifies cancellation handling
func TestCmekService_List_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	responseBody, _ := json.Marshal(GetCmeksResponse{Data: []GetCmeksData{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestCmekService(mock)

	start := time.Now()
	_, err := service.List(ctx, "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context cancelled error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("cancellation took too long: %v", elapsed)
	}
}

// TestCmekService_List_ContextTimeout verifies timeout enforcement
func TestCmekService_List_ContextTimeout(t *testing.T) {
	responseBody, _ := json.Marshal(GetCmeksResponse{Data: []GetCmeksData{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: responseBody},
		delay:    2 * time.Second,
	}

	service := createTestCmekServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.List(context.Background(), "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}
