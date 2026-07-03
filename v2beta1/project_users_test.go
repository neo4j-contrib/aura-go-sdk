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

// createTestProjectUserService creates a projectUserService with a mock API
// service for testing.
func createTestProjectUserService(mock *mockAPIService) *projectUserService {
	return &projectUserService{api: mock, timeout: 30 * time.Second, logger: testLogger()}
}

// createTestProjectUserServiceWithTimeout creates a projectUserService with a
// specific timeout.
func createTestProjectUserServiceWithTimeout(mock api.RequestService, timeout time.Duration) *projectUserService {
	return &projectUserService{api: mock, timeout: timeout, logger: testLogger()}
}

// ============================================================================
// projectUserService.List tests
// ============================================================================

// TestProjectUserService_List_Success verifies that List calls
// GET organizations/{orgID}/projects/{projectID}/users and maps all response fields.
func TestProjectUserService_List_Success(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	expected := ListProjectUsersResponse{
		Data: []ProjectUser{
			{
				UserID:       userID,
				Email:        "alice@example.com",
				ProjectRoles: []string{"admin"},
			},
			{
				UserID:       "cccccccc-dddd-eeee-ffff-000000000002",
				Email:        "bob@example.com",
				ProjectRoles: []string{"viewer"},
			},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestProjectUserService(mock)
	result, err := service.List(context.Background(), orgID, projectID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/users"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(result.Data))
	}
	if result.Data[0].UserID != userID {
		t.Errorf("expected first user ID %q, got %q", userID, result.Data[0].UserID)
	}
	if result.Data[0].Email != "alice@example.com" {
		t.Errorf("expected first user email %q, got %q", "alice@example.com", result.Data[0].Email)
	}
	if len(result.Data[0].ProjectRoles) != 1 || result.Data[0].ProjectRoles[0] != "admin" {
		t.Errorf("expected first user project roles [admin], got %v", result.Data[0].ProjectRoles)
	}
}

// TestProjectUserService_List_EmptyResult verifies that an empty user list is
// returned without error.
func TestProjectUserService_List_EmptyResult(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	body, _ := json.Marshal(ListProjectUsersResponse{Data: []ProjectUser{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestProjectUserService(mock)
	result, err := service.List(context.Background(), orgID, projectID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 users, got %d", len(result.Data))
	}
}

// TestProjectUserService_List_InvalidOrgID verifies that List rejects invalid
// org IDs without making an API call.
func TestProjectUserService_List_InvalidOrgID(t *testing.T) {
	const projectID = "11111111-2222-3333-4444-555555555555"

	cases := []struct {
		name  string
		orgID string
	}{
		{name: "empty", orgID: ""},
		{name: "not-a-uuid", orgID: "not-a-uuid"},
		{name: "too-short", orgID: "abc123"},
		{name: "wrong-format", orgID: "aaaaaaaa-bbbb-cccc-dddd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			result, err := service.List(context.Background(), tc.orgID, projectID)

			if err == nil {
				t.Fatalf("expected error for orgID %q, got nil", tc.orgID)
			}
			if result != nil {
				t.Error("expected nil result for invalid org ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid orgID %q, but got path %q", tc.orgID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "organization ID") {
				t.Errorf("expected error to mention 'organization ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_List_InvalidProjectID verifies that List rejects invalid
// project IDs without making an API call.
func TestProjectUserService_List_InvalidProjectID(t *testing.T) {
	const orgID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	cases := []struct {
		name      string
		projectID string
	}{
		{name: "empty", projectID: ""},
		{name: "not-a-uuid", projectID: "not-a-uuid"},
		{name: "too-short", projectID: "abc123"},
		{name: "wrong-format", projectID: "11111111-2222-3333-4444"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			result, err := service.List(context.Background(), orgID, tc.projectID)

			if err == nil {
				t.Fatalf("expected error for projectID %q, got nil", tc.projectID)
			}
			if result != nil {
				t.Error("expected nil result for invalid project ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid projectID %q, but got path %q", tc.projectID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "project ID") {
				t.Errorf("expected error to mention 'project ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_List_NotFound verifies that a 404 API error is returned
// as *api.Error with IsNotFound() == true.
func TestProjectUserService_List_NotFound(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestProjectUserService(mock)
	result, err := service.List(context.Background(), orgID, projectID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestProjectUserService_List_AuthenticationError verifies that a 401 API error
// exposes IsUnauthorized() == true.
func TestProjectUserService_List_AuthenticationError(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestProjectUserService(mock)
	_, err := service.List(context.Background(), orgID, projectID)

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

// TestProjectUserService_List_ContextTimeout verifies that the service timeout
// fires before the mock delay, returning context.DeadlineExceeded.
func TestProjectUserService_List_ContextTimeout(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	body, _ := json.Marshal(ListProjectUsersResponse{Data: []ProjectUser{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.List(context.Background(), orgID, projectID)
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

// TestProjectUserService_List_QuickCancellation verifies that a pre-expired
// context causes List to fail immediately with a context error.
func TestProjectUserService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	body, _ := json.Marshal(ListProjectUsersResponse{Data: []ProjectUser{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 30*time.Second)
	_, err := service.List(ctx, orgID, projectID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// projectUserService.Add tests
// ============================================================================

// TestProjectUserService_Add_Success verifies that Add calls
// POST organizations/{orgID}/projects/{projectID}/users, serialises the request
// body correctly, and returns nil on success.
func TestProjectUserService_Add_Success(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	mock := &mockAPIService{
		response: &api.Response{StatusCode: 201, Body: []byte{}},
	}

	service := createTestProjectUserService(mock)
	req := &AddProjectUserRequest{
		UserID:       userID,
		ProjectRoles: []string{"admin"},
	}
	err := service.Add(context.Background(), orgID, projectID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/users"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}

	// Verify request body was serialised correctly.
	var sent AddProjectUserRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sent); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if sent.UserID != userID {
		t.Errorf("expected request user_id %q, got %q", userID, sent.UserID)
	}
	if len(sent.ProjectRoles) != 1 || sent.ProjectRoles[0] != "admin" {
		t.Errorf("expected request project_roles [admin], got %v", sent.ProjectRoles)
	}
}

// TestProjectUserService_Add_InvalidOrgID verifies that Add rejects invalid
// org IDs without making an API call.
func TestProjectUserService_Add_InvalidOrgID(t *testing.T) {
	const projectID = "11111111-2222-3333-4444-555555555555"

	cases := []struct {
		name  string
		orgID string
	}{
		{name: "empty", orgID: ""},
		{name: "not-a-uuid", orgID: "not-a-uuid"},
		{name: "too-short", orgID: "abc123"},
		{name: "wrong-format", orgID: "aaaaaaaa-bbbb-cccc-dddd"},
	}

	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			err := service.Add(context.Background(), tc.orgID, projectID, req)

			if err == nil {
				t.Fatalf("expected error for orgID %q, got nil", tc.orgID)
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid orgID %q, but got path %q", tc.orgID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "organization ID") {
				t.Errorf("expected error to mention 'organization ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_Add_InvalidProjectID verifies that Add rejects invalid
// project IDs without making an API call.
func TestProjectUserService_Add_InvalidProjectID(t *testing.T) {
	const orgID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	cases := []struct {
		name      string
		projectID string
	}{
		{name: "empty", projectID: ""},
		{name: "not-a-uuid", projectID: "not-a-uuid"},
		{name: "too-short", projectID: "abc123"},
		{name: "wrong-format", projectID: "11111111-2222-3333-4444"},
	}

	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			err := service.Add(context.Background(), orgID, tc.projectID, req)

			if err == nil {
				t.Fatalf("expected error for projectID %q, got nil", tc.projectID)
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid projectID %q, but got path %q", tc.projectID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "project ID") {
				t.Errorf("expected error to mention 'project ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_Add_NotFound verifies that a 404 API error is surfaced
// as *api.Error with IsNotFound() == true.
func TestProjectUserService_Add_NotFound(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestProjectUserService(mock)
	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}
	err := service.Add(context.Background(), orgID, projectID, req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestProjectUserService_Add_AuthenticationError verifies that a 401 API error
// exposes IsUnauthorized() == true.
func TestProjectUserService_Add_AuthenticationError(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestProjectUserService(mock)
	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}
	err := service.Add(context.Background(), orgID, projectID, req)

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

// TestProjectUserService_Add_ContextTimeout verifies that the service timeout
// fires before the mock delay, returning context.DeadlineExceeded.
func TestProjectUserService_Add_ContextTimeout(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: []byte{}},
		delay:    2 * time.Second,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 100*time.Millisecond)
	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}

	start := time.Now()
	err := service.Add(context.Background(), orgID, projectID, req)
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

// TestProjectUserService_Add_QuickCancellation verifies that a pre-expired
// context causes Add to fail immediately with a context error.
func TestProjectUserService_Add_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: []byte{}},
		delay:    0,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 30*time.Second)
	req := &AddProjectUserRequest{
		UserID:       "cccccccc-dddd-eeee-ffff-000000000001",
		ProjectRoles: []string{"admin"},
	}
	err := service.Add(ctx, orgID, projectID, req)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// projectUserService.UpdateRole tests
// ============================================================================

// TestProjectUserService_UpdateRole_Success verifies that UpdateRole calls
// PATCH organizations/{orgID}/projects/{projectID}/users/{userID}, serialises
// the request body correctly, and returns the updated user.
func TestProjectUserService_UpdateRole_Success(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	expected := GetProjectUserResponse{
		Data: ProjectUser{
			UserID:       userID,
			Email:        "alice@example.com",
			ProjectRoles: []string{"viewer"},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestProjectUserService(mock)
	req := &UpdateProjectUserRequest{
		ProjectRoles: []string{"viewer"},
	}
	result, err := service.UpdateRole(context.Background(), orgID, projectID, userID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "PATCH" {
		t.Errorf("expected PATCH method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/users/" + userID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.UserID != userID {
		t.Errorf("expected user ID %q, got %q", userID, result.Data.UserID)
	}
	if len(result.Data.ProjectRoles) != 1 || result.Data.ProjectRoles[0] != "viewer" {
		t.Errorf("expected project roles [viewer], got %v", result.Data.ProjectRoles)
	}

	// Verify request body was serialised correctly.
	var sent UpdateProjectUserRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sent); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if len(sent.ProjectRoles) != 1 || sent.ProjectRoles[0] != "viewer" {
		t.Errorf("expected request project_roles [viewer], got %v", sent.ProjectRoles)
	}
}

// TestProjectUserService_UpdateRole_InvalidOrgID verifies that UpdateRole rejects
// invalid org IDs without making an API call.
func TestProjectUserService_UpdateRole_InvalidOrgID(t *testing.T) {
	const (
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	cases := []struct {
		name  string
		orgID string
	}{
		{name: "empty", orgID: ""},
		{name: "not-a-uuid", orgID: "not-a-uuid"},
		{name: "too-short", orgID: "abc123"},
		{name: "wrong-format", orgID: "aaaaaaaa-bbbb-cccc-dddd"},
	}

	req := &UpdateProjectUserRequest{ProjectRoles: []string{"viewer"}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			result, err := service.UpdateRole(context.Background(), tc.orgID, projectID, userID, req)

			if err == nil {
				t.Fatalf("expected error for orgID %q, got nil", tc.orgID)
			}
			if result != nil {
				t.Error("expected nil result for invalid org ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid orgID %q, but got path %q", tc.orgID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "organization ID") {
				t.Errorf("expected error to mention 'organization ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_UpdateRole_InvalidProjectID verifies that UpdateRole
// rejects invalid project IDs without making an API call.
func TestProjectUserService_UpdateRole_InvalidProjectID(t *testing.T) {
	const (
		orgID  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		userID = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	cases := []struct {
		name      string
		projectID string
	}{
		{name: "empty", projectID: ""},
		{name: "not-a-uuid", projectID: "not-a-uuid"},
		{name: "too-short", projectID: "abc123"},
		{name: "wrong-format", projectID: "11111111-2222-3333-4444"},
	}

	req := &UpdateProjectUserRequest{ProjectRoles: []string{"viewer"}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			result, err := service.UpdateRole(context.Background(), orgID, tc.projectID, userID, req)

			if err == nil {
				t.Fatalf("expected error for projectID %q, got nil", tc.projectID)
			}
			if result != nil {
				t.Error("expected nil result for invalid project ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid projectID %q, but got path %q", tc.projectID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "project ID") {
				t.Errorf("expected error to mention 'project ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_UpdateRole_InvalidUserID verifies that UpdateRole rejects
// invalid user IDs without making an API call.
func TestProjectUserService_UpdateRole_InvalidUserID(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	cases := []struct {
		name   string
		userID string
	}{
		{name: "empty", userID: ""},
		{name: "not-a-uuid", userID: "not-a-uuid"},
		{name: "too-short", userID: "abc123"},
		{name: "wrong-format", userID: "cccccccc-dddd-eeee-ffff"},
	}

	req := &UpdateProjectUserRequest{ProjectRoles: []string{"viewer"}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			result, err := service.UpdateRole(context.Background(), orgID, projectID, tc.userID, req)

			if err == nil {
				t.Fatalf("expected error for userID %q, got nil", tc.userID)
			}
			if result != nil {
				t.Error("expected nil result for invalid user ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid userID %q, but got path %q", tc.userID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "user ID") {
				t.Errorf("expected error to mention 'user ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_UpdateRole_NotFound verifies that a 404 API error is
// surfaced as *api.Error with IsNotFound() == true.
func TestProjectUserService_UpdateRole_NotFound(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "User not found"},
	}

	service := createTestProjectUserService(mock)
	req := &UpdateProjectUserRequest{ProjectRoles: []string{"viewer"}}
	result, err := service.UpdateRole(context.Background(), orgID, projectID, userID, req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestProjectUserService_UpdateRole_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestProjectUserService_UpdateRole_ContextTimeout(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	body, _ := json.Marshal(GetProjectUserResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 100*time.Millisecond)
	req := &UpdateProjectUserRequest{ProjectRoles: []string{"viewer"}}

	start := time.Now()
	_, err := service.UpdateRole(context.Background(), orgID, projectID, userID, req)
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

// ============================================================================
// projectUserService.Remove tests
// ============================================================================

// TestProjectUserService_Remove_Success verifies that Remove calls
// DELETE organizations/{orgID}/projects/{projectID}/users/{userID} and returns nil.
func TestProjectUserService_Remove_Success(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	mock := &mockAPIService{
		response: &api.Response{StatusCode: 204, Body: []byte{}},
	}

	service := createTestProjectUserService(mock)
	err := service.Remove(context.Background(), orgID, projectID, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/users/" + userID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
}

// TestProjectUserService_Remove_InvalidOrgID verifies that Remove rejects invalid
// org IDs without making an API call.
func TestProjectUserService_Remove_InvalidOrgID(t *testing.T) {
	const (
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	cases := []struct {
		name  string
		orgID string
	}{
		{name: "empty", orgID: ""},
		{name: "not-a-uuid", orgID: "not-a-uuid"},
		{name: "too-short", orgID: "abc123"},
		{name: "wrong-format", orgID: "aaaaaaaa-bbbb-cccc-dddd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			err := service.Remove(context.Background(), tc.orgID, projectID, userID)

			if err == nil {
				t.Fatalf("expected error for orgID %q, got nil", tc.orgID)
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid orgID %q, but got path %q", tc.orgID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "organization ID") {
				t.Errorf("expected error to mention 'organization ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_Remove_InvalidProjectID verifies that Remove rejects
// invalid project IDs without making an API call.
func TestProjectUserService_Remove_InvalidProjectID(t *testing.T) {
	const (
		orgID  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		userID = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	cases := []struct {
		name      string
		projectID string
	}{
		{name: "empty", projectID: ""},
		{name: "not-a-uuid", projectID: "not-a-uuid"},
		{name: "too-short", projectID: "abc123"},
		{name: "wrong-format", projectID: "11111111-2222-3333-4444"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			err := service.Remove(context.Background(), orgID, tc.projectID, userID)

			if err == nil {
				t.Fatalf("expected error for projectID %q, got nil", tc.projectID)
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid projectID %q, but got path %q", tc.projectID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "project ID") {
				t.Errorf("expected error to mention 'project ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_Remove_InvalidUserID verifies that Remove rejects invalid
// user IDs without making an API call.
func TestProjectUserService_Remove_InvalidUserID(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	cases := []struct {
		name   string
		userID string
	}{
		{name: "empty", userID: ""},
		{name: "not-a-uuid", userID: "not-a-uuid"},
		{name: "too-short", userID: "abc123"},
		{name: "wrong-format", userID: "cccccccc-dddd-eeee-ffff"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestProjectUserService(mock)

			err := service.Remove(context.Background(), orgID, projectID, tc.userID)

			if err == nil {
				t.Fatalf("expected error for userID %q, got nil", tc.userID)
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid userID %q, but got path %q", tc.userID, mock.lastPath)
			}
			if !strings.Contains(err.Error(), "user ID") {
				t.Errorf("expected error to mention 'user ID', got: %v", err)
			}
		})
	}
}

// TestProjectUserService_Remove_NotFound verifies that a 404 API error is surfaced
// as *api.Error with IsNotFound() == true.
func TestProjectUserService_Remove_NotFound(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "User not found"},
	}

	service := createTestProjectUserService(mock)
	err := service.Remove(context.Background(), orgID, projectID, userID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

// TestProjectUserService_Remove_ContextTimeout verifies that the service timeout
// fires before the mock delay, returning context.DeadlineExceeded.
func TestProjectUserService_Remove_ContextTimeout(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
		userID    = "cccccccc-dddd-eeee-ffff-000000000001"
	)

	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 204, Body: []byte{}},
		delay:    2 * time.Second,
	}

	service := createTestProjectUserServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	err := service.Remove(context.Background(), orgID, projectID, userID)
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
