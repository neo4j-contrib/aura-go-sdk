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

// createTestProjectService creates a projectService with a mock API service for
// testing.
func createTestProjectService(mock *mockAPIService) *projectService {
	return &projectService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestProjectServiceWithTimeout creates a projectService with a specific
// timeout. Pass the desired context directly to each method call.
func createTestProjectServiceWithTimeout(mock api.RequestService, timeout time.Duration) *projectService {
	return &projectService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// ============================================================================
// projectService.List tests
// ============================================================================

// TestProjectService_List_Success verifies that List calls
// GET /organizations/{orgID}/projects and correctly maps all response fields.
func TestProjectService_List_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	expected := ListProjectsResponse{
		Data: []Project{
			{ID: "11111111-2222-3333-4444-555555555555", Name: "Production"},
			{ID: "66666666-7777-8888-9999-aaaaaaaaaaaa", Name: "Development"},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestProjectService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(result.Data))
	}
	if result.Data[0].ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected first project ID '11111111-2222-3333-4444-555555555555', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Name != "Production" {
		t.Errorf("expected first project name 'Production', got '%s'", result.Data[0].Name)
	}
	if result.Data[1].ID != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected second project ID '66666666-7777-8888-9999-aaaaaaaaaaaa', got '%s'", result.Data[1].ID)
	}
	if result.Data[1].Name != "Development" {
		t.Errorf("expected second project name 'Development', got '%s'", result.Data[1].Name)
	}
}

// TestProjectService_List_EmptyResult verifies that an empty projects list is
// returned without error.
func TestProjectService_List_EmptyResult(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListProjectsResponse{Data: []Project{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestProjectService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 projects, got %d", len(result.Data))
	}
}

// TestProjectService_List_NotFound verifies that a 404 API error is returned as
// *api.Error with IsNotFound() == true.
func TestProjectService_List_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestProjectService(mock)
	result, err := service.List(context.Background(), orgID)

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

// TestProjectService_List_AuthenticationError verifies that a 401 API error
// exposes IsUnauthorized() == true.
func TestProjectService_List_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestProjectService(mock)
	_, err := service.List(context.Background(), orgID)

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

// TestProjectService_List_MissingOrgID verifies that List returns a descriptive
// error without calling the API when an empty org ID is passed.
func TestProjectService_List_MissingOrgID(t *testing.T) {
	mock := &mockAPIService{}

	service := createTestProjectService(mock)
	result, err := service.List(context.Background(), "")

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected result to be nil when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID is required") {
		t.Errorf("expected error to contain 'organization ID is required', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call to be made, but got path '%s'", mock.lastPath)
	}
}

// TestProjectService_List_ContextTimeout verifies that the service timeout fires
// before the mock delay, returning context.DeadlineExceeded.
func TestProjectService_List_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListProjectsResponse{Data: []Project{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestProjectServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.List(context.Background(), orgID)
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

// TestProjectService_List_QuickCancellation verifies that a pre-expired context
// causes List to fail immediately with a context error.
func TestProjectService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListProjectsResponse{Data: []Project{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestProjectServiceWithTimeout(mock, 30*time.Second)

	_, err := service.List(ctx, orgID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
