package v2beta1

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
)

// ============================================================================
// Constructor helpers
// ============================================================================

// createTestOrganizationService creates an organizationService with a mock API
// service for testing.
func createTestOrganizationService(mock *mockAPIService) *organizationService {
	return &organizationService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestOrganizationServiceWithTimeout creates an organizationService with a
// specific timeout.
func createTestOrganizationServiceWithTimeout(mock api.RequestService, timeout time.Duration) *organizationService {
	return &organizationService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// ============================================================================
// organizationService.List tests
// ============================================================================

// TestOrganizationService_List_Success verifies that List calls GET /organizations
// and correctly maps all response fields.
func TestOrganizationService_List_Success(t *testing.T) {
	expected := ListOrganizationsResponse{
		Data: []Organization{
			{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Name: "Acme Corp"},
			{ID: "11111111-2222-3333-4444-555555555555", Name: "Beta Inc"},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	if mock.lastPath != "organizations" {
		t.Errorf("expected path 'organizations', got '%s'", mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 organizations, got %d", len(result.Data))
	}
	if result.Data[0].ID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected first org ID 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Name != "Acme Corp" {
		t.Errorf("expected first org name 'Acme Corp', got '%s'", result.Data[0].Name)
	}
	if result.Data[1].ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected second org ID '11111111-2222-3333-4444-555555555555', got '%s'", result.Data[1].ID)
	}
}

// TestOrganizationService_List_EmptyResult verifies that an empty organizations list
// is returned without error.
func TestOrganizationService_List_EmptyResult(t *testing.T) {
	body, _ := json.Marshal(ListOrganizationsResponse{Data: []Organization{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 organizations, got %d", len(result.Data))
	}
}

// TestOrganizationService_List_NotFound verifies that a 404 API error is returned
// as *api.Error with IsNotFound() == true.
func TestOrganizationService_List_NotFound(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestOrganizationService(mock)
	result, err := service.List(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected result to be nil on error")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestOrganizationService_List_AuthenticationError verifies that a 401 API error
// exposes IsUnauthorized() == true.
func TestOrganizationService_List_AuthenticationError(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationService(mock)
	_, err := service.List(context.Background())

	if err == nil {
		t.Fatal("expected authentication error, got nil")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsUnauthorized() {
		t.Error("expected IsUnauthorized() to be true")
	}
}

// TestOrganizationService_List_ContextTimeout verifies that the service timeout
// fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationService_List_ContextTimeout(t *testing.T) {
	body, _ := json.Marshal(ListOrganizationsResponse{Data: []Organization{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.List(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~100ms)", elapsed)
	}
}

// TestOrganizationService_List_QuickCancellation verifies that a pre-expired
// context causes List to fail immediately with a context error.
func TestOrganizationService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	body, _ := json.Marshal(ListOrganizationsResponse{Data: []Organization{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestOrganizationServiceWithTimeout(mock, 30*time.Second)
	_, err := service.List(ctx)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationService.Get tests
// ============================================================================

// TestOrganizationService_Get_Success verifies that Get calls
// GET /organizations/{id} with the provided org ID and maps all response fields.
func TestOrganizationService_Get_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	expected := GetOrganizationResponse{
		Data: Organization{ID: orgID, Name: "Acme Corp"},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationService(mock)
	result, err := service.Get(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	if mock.lastPath != "organizations/"+orgID {
		t.Errorf("expected path 'organizations/%s', got '%s'", orgID, mock.lastPath)
	}
	if result.Data.ID != orgID {
		t.Errorf("expected org ID '%s', got '%s'", orgID, result.Data.ID)
	}
	if result.Data.Name != "Acme Corp" {
		t.Errorf("expected org name 'Acme Corp', got '%s'", result.Data.Name)
	}
}

// TestOrganizationService_Get_NotFound verifies that a 404 API error is surfaced
// as *api.Error with IsNotFound() == true.
func TestOrganizationService_Get_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Organization not found"},
	}

	service := createTestOrganizationService(mock)
	result, err := service.Get(context.Background(), orgID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected result to be nil on error")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestOrganizationService_Get_AuthenticationError verifies that a 401 API error
// exposes IsUnauthorized() == true.
func TestOrganizationService_Get_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationService(mock)
	_, err := service.Get(context.Background(), orgID)

	if err == nil {
		t.Fatal("expected authentication error, got nil")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsUnauthorized() {
		t.Error("expected IsUnauthorized() to be true")
	}
}

// TestOrganizationService_Get_EmptyOrgID verifies that Get returns a
// descriptive error without calling the API when an empty org ID is passed.
func TestOrganizationService_Get_EmptyOrgID(t *testing.T) {
	mock := &mockAPIService{}

	service := createTestOrganizationService(mock)
	result, err := service.Get(context.Background(), "")

	if err == nil {
		t.Fatal("expected error for empty org ID, got nil")
	}
	if result != nil {
		t.Error("expected result to be nil when org ID is empty")
	}
	if !strings.Contains(err.Error(), "organization ID is required") {
		t.Errorf("expected error to contain 'organization ID is required', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call to be made, but got path '%s'", mock.lastPath)
	}
}

// TestOrganizationService_Get_ContextTimeout verifies that the service timeout
// fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationService_Get_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(GetOrganizationResponse{Data: Organization{ID: orgID}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := &organizationService{
		api:     mock,
		timeout: 100 * time.Millisecond,
		logger:  testLogger(),
	}

	start := time.Now()
	_, err := service.Get(context.Background(), orgID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~100ms)", elapsed)
	}
}

// TestOrganizationService_Get_QuickCancellation verifies that a pre-expired
// context causes Get to fail immediately with a context error.
func TestOrganizationService_Get_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(GetOrganizationResponse{Data: Organization{ID: orgID}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := &organizationService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}

	_, err := service.Get(ctx, orgID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
