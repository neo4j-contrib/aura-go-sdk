package aura

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/LackOfMorals/aura-client/internal/api"
)

// createTestTenantService creates a tenantService with a mock API service for testing
func createTestTenantService(mock *mockAPIService) *tenantService {
	return &tenantService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestTenantServiceWithTimeout creates a tenantService with a specific timeout.
// Pass the desired context directly to each method call.
func createTestTenantServiceWithTimeout(mock api.RequestService, timeout time.Duration) *tenantService {
	return &tenantService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// TestTenantService_List_Success verifies successful tenant listing
func TestTenantService_List_Success(t *testing.T) {
	expectedResponse := ListTenantsResponse{
		Data: []TenantsResponseData{
			{ID: "tenant-1", Name: "Development Team"},
			{ID: "tenant-2", Name: "Production Team"},
			{ID: "tenant-3", Name: "Testing Team"},
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	if mock.lastPath != "tenants" {
		t.Errorf("expected path 'tenants', got '%s'", mock.lastPath)
	}
	if len(result.Data) != 3 {
		t.Errorf("expected 3 tenants, got %d", len(result.Data))
	}
	if result.Data[0].ID != "tenant-1" {
		t.Errorf("expected first tenant ID 'tenant-1', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Name != "Development Team" {
		t.Errorf("expected first tenant name 'Development Team', got '%s'", result.Data[0].Name)
	}
}

// TestTenantService_List_EmptyResult verifies empty tenant list
func TestTenantService_List_EmptyResult(t *testing.T) {
	responseBody, _ := json.Marshal(ListTenantsResponse{Data: []TenantsResponseData{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 tenants, got %d", len(result.Data))
	}
}

// TestTenantService_Get_Success verifies retrieving a specific tenant
func TestTenantService_Get_Success(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	expectedResponse := GetTenantResponse{
		Data: TenantResponseData{
			ID:   tenantID,
			Name: "Development Team",
			InstanceConfigurations: []TenantInstanceConfiguration{
				{CloudProvider: "gcp", Region: "us-central1", RegionName: "Iowa", Type: "enterprise-db", Memory: "8GB", Storage: "256GB", Version: "5"},
				{CloudProvider: "aws", Region: "us-east-1", RegionName: "N. Virginia", Type: "enterprise-db", Memory: "16GB", Storage: "512GB", Version: "5"},
			},
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.Get(context.Background(), tenantID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastPath != "tenants/"+tenantID {
		t.Errorf("expected path 'tenants/%s', got '%s'", tenantID, mock.lastPath)
	}
	if result.Data.ID != tenantID {
		t.Errorf("expected tenant ID '%s', got '%s'", tenantID, result.Data.ID)
	}
	if result.Data.Name != "Development Team" {
		t.Errorf("expected tenant name 'Development Team', got '%s'", result.Data.Name)
	}
	if len(result.Data.InstanceConfigurations) != 2 {
		t.Errorf("expected 2 instance configurations, got %d", len(result.Data.InstanceConfigurations))
	}
}

// TestTenantService_Get_InstanceConfigurations verifies instance configuration details
func TestTenantService_Get_InstanceConfigurations(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	expectedResponse := GetTenantResponse{
		Data: TenantResponseData{
			ID:   tenantID,
			Name: "Test Tenant",
			InstanceConfigurations: []TenantInstanceConfiguration{
				{CloudProvider: "gcp", Region: "europe-west2", RegionName: "London", Type: "enterprise-db", Memory: "32GB", Storage: "1024GB", Version: "5"},
			},
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.Get(context.Background(), tenantID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	config := result.Data.InstanceConfigurations[0]
	if config.CloudProvider != "gcp" {
		t.Errorf("expected cloud provider 'gcp', got '%s'", config.CloudProvider)
	}
	if config.Region != "europe-west2" {
		t.Errorf("expected region 'europe-west2', got '%s'", config.Region)
	}
	if config.RegionName != "London" {
		t.Errorf("expected region name 'London', got '%s'", config.RegionName)
	}
	if config.Memory != "32GB" {
		t.Errorf("expected memory '32GB', got '%s'", config.Memory)
	}
}

// TestTenantService_Get_NotFound verifies 404 handling
func TestTenantService_Get_NotFound(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Tenant not found"},
	}

	service := createTestTenantService(mock)
	result, err := service.Get(context.Background(), "00000000-0000-0000-0000-000000000000")

	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
	if result != nil {
		t.Error("expected result to be nil on error")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatal("expected Error type")
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestTenantService_AuthenticationError verifies auth error handling
func TestTenantService_AuthenticationError(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestTenantService(mock)
	_, err := service.List(context.Background())

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

// TestTenantService_Get_NoInstanceConfigurations verifies tenant without configurations
func TestTenantService_Get_NoInstanceConfigurations(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	responseBody, _ := json.Marshal(GetTenantResponse{
		Data: TenantResponseData{ID: tenantID, Name: "Empty Tenant", InstanceConfigurations: []TenantInstanceConfiguration{}},
	})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.Get(context.Background(), tenantID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data.InstanceConfigurations) != 0 {
		t.Errorf("expected 0 instance configurations, got %d", len(result.Data.InstanceConfigurations))
	}
}

// TestTenantService_SingleTenant verifies list with single tenant
func TestTenantService_SingleTenant(t *testing.T) {
	responseBody, _ := json.Marshal(ListTenantsResponse{
		Data: []TenantsResponseData{{ID: "tenant-single", Name: "Only Tenant"}},
	})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(result.Data))
	}
	if result.Data[0].ID != "tenant-single" {
		t.Errorf("expected tenant ID 'tenant-single', got '%s'", result.Data[0].ID)
	}
}

// ============================================================================
// Context-Specific Tests for TenantService
// ============================================================================

// TestTenantService_Get_ContextTimeout verifies timeout enforcement
func TestTenantService_Get_ContextTimeout(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	responseBody, _ := json.Marshal(GetTenantResponse{
		Data: TenantResponseData{ID: tenantID, Name: "Test"},
	})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: responseBody},
		delay:    2 * time.Second,
	}

	service := createTestTenantServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.Get(context.Background(), tenantID)
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

// ============================================================================
// GetMetrics Tests
// ============================================================================

// TestTenantService_GetMetrics_Success verifies successful GetMetrics call.
func TestTenantService_GetMetrics_Success(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	expectedResponse := GetTenantMetricsURLResponse{
		Data: GetTenantMetricsURLData{
			Endpoint: "https://metrics.example.com/tenant/prometheus",
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestTenantService(mock)
	result, err := service.GetMetrics(context.Background(), tenantID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "tenants/" + tenantID + "/metrics-integration"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.Endpoint != expectedResponse.Data.Endpoint {
		t.Errorf("expected endpoint %q, got %q", expectedResponse.Data.Endpoint, result.Data.Endpoint)
	}
}

// TestTenantService_GetMetrics_APIError verifies that GetMetrics logs an ErrorContext
// with tenantID and error fields when the API call fails.
func TestTenantService_GetMetrics_APIError(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	apiErr := &api.Error{StatusCode: 500, Message: "internal server error"}
	mock := &mockAPIService{err: apiErr}

	handler := &capturingHandler{}
	service := &tenantService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  slog.New(handler),
	}

	result, err := service.GetMetrics(context.Background(), tenantID)

	if err == nil {
		t.Fatal("expected error from GetMetrics")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
	if !errors.Is(err, apiErr) {
		t.Errorf("expected api error, got %v", err)
	}

	if !handler.hasRecord(slog.LevelError, "failed to get tenant metrics url") {
		t.Error("expected ErrorContext log with 'failed to get tenant metrics url' message")
	}
	if !handler.hasAttr("tenantID", tenantID) {
		t.Errorf("expected log attr tenantID=%q", tenantID)
	}
	if !handler.hasAttr("error", apiErr.Error()) {
		t.Errorf("expected log attr error=%q", apiErr.Error())
	}
}
