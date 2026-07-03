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

// createTestOrganizationUserService creates an organizationUserService with a
// mock API service for testing.
func createTestOrganizationUserService(mock *mockAPIService) *organizationUserService {
	return &organizationUserService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestOrganizationUserServiceWithTimeout creates an organizationUserService
// with a specific timeout.
func createTestOrganizationUserServiceWithTimeout(mock api.RequestService, timeout time.Duration) *organizationUserService {
	return &organizationUserService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// ============================================================================
// organizationUserService.List tests
// ============================================================================

// TestOrganizationUserService_List_Success verifies that List calls
// GET /organizations/{orgID}/users and correctly maps all response fields.
func TestOrganizationUserService_List_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	expected := ListOrganizationUsersResponse{
		Data: []OrganizationUser{
			{
				UserID:              "11111111-2222-3333-4444-555555555555",
				Email:               "alice@example.com",
				OrganizationRoles:   []string{"organization:admin"},
				MFAEnrollmentStatus: "enrolled",
			},
			{
				UserID:              "66666666-7777-8888-9999-aaaaaaaaaaaa",
				Email:               "bob@example.com",
				OrganizationRoles:   []string{"organization:member"},
				MFAEnrollmentStatus: "not_enrolled",
			},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/users"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(result.Data))
	}
	if result.Data[0].UserID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected first user ID '11111111-2222-3333-4444-555555555555', got '%s'", result.Data[0].UserID)
	}
	if result.Data[0].Email != "alice@example.com" {
		t.Errorf("expected first user email 'alice@example.com', got '%s'", result.Data[0].Email)
	}
	if result.Data[1].UserID != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected second user ID '66666666-7777-8888-9999-aaaaaaaaaaaa', got '%s'", result.Data[1].UserID)
	}
}

// TestOrganizationUserService_List_EmptyResult verifies that an empty users list
// is returned without error.
func TestOrganizationUserService_List_EmptyResult(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationUsersResponse{Data: []OrganizationUser{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 users, got %d", len(result.Data))
	}
}

// TestOrganizationUserService_List_InvalidOrgID verifies that List returns a
// descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationUserService_List_InvalidOrgID(t *testing.T) {
	tests := []struct {
		name  string
		orgID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			result, err := service.List(context.Background(), tc.orgID)

			if err == nil {
				t.Fatal("expected error for invalid org ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_List_NotFound verifies that a 404 API error is
// returned as *api.Error with IsNotFound() == true.
func TestOrganizationUserService_List_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestOrganizationUserService(mock)
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

// TestOrganizationUserService_List_AuthenticationError verifies that a 401 API
// error exposes IsUnauthorized() == true.
func TestOrganizationUserService_List_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationUserService(mock)
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

// TestOrganizationUserService_List_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationUserService_List_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationUsersResponse{Data: []OrganizationUser{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 100*time.Millisecond)

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

// TestOrganizationUserService_List_QuickCancellation verifies that a
// pre-expired context causes List to fail immediately with a context error.
func TestOrganizationUserService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationUsersResponse{Data: []OrganizationUser{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 30*time.Second)
	_, err := service.List(ctx, orgID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationUserService.Get tests
// ============================================================================

// TestOrganizationUserService_Get_Success verifies that Get calls
// GET /organizations/{orgID}/users/{userID} and maps all response fields.
func TestOrganizationUserService_Get_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	expected := GetOrganizationUserResponse{
		Data: OrganizationUserDetails{
			OrganizationUser: OrganizationUser{
				UserID:              userID,
				Email:               "alice@example.com",
				OrganizationRoles:   []string{"organization:admin"},
				MFAEnrollmentStatus: "enrolled",
			},
			Projects: []OrganizationUserProject{
				{ID: "66666666-7777-8888-9999-aaaaaaaaaaaa", Name: "Production", ProjectRoles: []string{"project:admin"}},
			},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.Get(context.Background(), orgID, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/users/" + userID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if result.Data.UserID != userID {
		t.Errorf("expected user ID '%s', got '%s'", userID, result.Data.UserID)
	}
	if result.Data.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got '%s'", result.Data.Email)
	}
	if len(result.Data.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(result.Data.Projects))
	}
	if result.Data.Projects[0].ID != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected project ID '66666666-7777-8888-9999-aaaaaaaaaaaa', got '%s'", result.Data.Projects[0].ID)
	}
}

// TestOrganizationUserService_Get_InvalidOrgID verifies that Get returns a
// descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationUserService_Get_InvalidOrgID(t *testing.T) {
	userID := "11111111-2222-3333-4444-555555555555"
	tests := []struct {
		name  string
		orgID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			result, err := service.Get(context.Background(), tc.orgID, userID)

			if err == nil {
				t.Fatal("expected error for invalid org ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_Get_InvalidUserID verifies that Get returns a
// descriptive error without calling the API for empty or malformed user IDs.
func TestOrganizationUserService_Get_InvalidUserID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tests := []struct {
		name   string
		userID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			result, err := service.Get(context.Background(), orgID, tc.userID)

			if err == nil {
				t.Fatal("expected error for invalid user ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_Get_NotFound verifies that a 404 API error is
// surfaced as *api.Error with IsNotFound() == true.
func TestOrganizationUserService_Get_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "User not found"},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.Get(context.Background(), orgID, userID)

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

// TestOrganizationUserService_Get_AuthenticationError verifies that a 401 API
// error exposes IsUnauthorized() == true.
func TestOrganizationUserService_Get_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationUserService(mock)
	_, err := service.Get(context.Background(), orgID, userID)

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

// TestOrganizationUserService_Get_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationUserService_Get_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	body, _ := json.Marshal(GetOrganizationUserResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.Get(context.Background(), orgID, userID)
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

// TestOrganizationUserService_Get_QuickCancellation verifies that a
// pre-expired context causes Get to fail immediately with a context error.
func TestOrganizationUserService_Get_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	body, _ := json.Marshal(GetOrganizationUserResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 30*time.Second)
	_, err := service.Get(ctx, orgID, userID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationUserService.UpdateRole tests
// ============================================================================

// TestOrganizationUserService_UpdateRole_Success verifies that UpdateRole calls
// PATCH /organizations/{orgID}/users/{userID} with the correct serialised body.
func TestOrganizationUserService_UpdateRole_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{
		OrganizationRoles: []string{"organization:admin"},
	}

	expectedResp := GetOrganizationUserResponse{
		Data: OrganizationUserDetails{
			OrganizationUser: OrganizationUser{
				UserID:            userID,
				Email:             "alice@example.com",
				OrganizationRoles: []string{"organization:admin"},
			},
		},
	}

	body, _ := json.Marshal(expectedResp)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.UpdateRole(context.Background(), orgID, userID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "PATCH" {
		t.Errorf("expected PATCH method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/users/" + userID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if result.Data.UserID != userID {
		t.Errorf("expected user ID '%s', got '%s'", userID, result.Data.UserID)
	}

	// Assert the request body was serialised correctly.
	var sentBody UpdateOrganizationUserRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentBody); err != nil {
		t.Fatalf("could not unmarshal sent body: %v", err)
	}
	if len(sentBody.OrganizationRoles) != 1 || sentBody.OrganizationRoles[0] != "organization:admin" {
		t.Errorf("expected organization_roles ['organization:admin'], got %v", sentBody.OrganizationRoles)
	}
}

// TestOrganizationUserService_UpdateRole_InvalidOrgID verifies that UpdateRole
// returns a descriptive error without calling the API for invalid org IDs.
func TestOrganizationUserService_UpdateRole_InvalidOrgID(t *testing.T) {
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	tests := []struct {
		name  string
		orgID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			result, err := service.UpdateRole(context.Background(), tc.orgID, userID, req)

			if err == nil {
				t.Fatal("expected error for invalid org ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_UpdateRole_NotFound verifies that a 404 API
// error is surfaced as *api.Error with IsNotFound() == true.
func TestOrganizationUserService_UpdateRole_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "User not found"},
	}

	service := createTestOrganizationUserService(mock)
	result, err := service.UpdateRole(context.Background(), orgID, userID, req)

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

// TestOrganizationUserService_UpdateRole_ContextTimeout verifies that the
// service timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationUserService_UpdateRole_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	body, _ := json.Marshal(GetOrganizationUserResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.UpdateRole(context.Background(), orgID, userID, req)
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

// TestOrganizationUserService_UpdateRole_InvalidUserID verifies that UpdateRole
// returns a descriptive error without calling the API for empty or malformed user IDs.
func TestOrganizationUserService_UpdateRole_InvalidUserID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	tests := []struct {
		name   string
		userID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			result, err := service.UpdateRole(context.Background(), orgID, tc.userID, req)

			if err == nil {
				t.Fatal("expected error for invalid user ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_UpdateRole_AuthenticationError verifies that a
// 401 API error exposes IsUnauthorized() == true.
func TestOrganizationUserService_UpdateRole_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationUserService(mock)
	_, err := service.UpdateRole(context.Background(), orgID, userID, req)

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

// TestOrganizationUserService_UpdateRole_QuickCancellation verifies that a
// pre-expired context causes UpdateRole to fail immediately with a context error.
func TestOrganizationUserService_UpdateRole_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateOrganizationUserRequest{OrganizationRoles: []string{"organization:member"}}
	body, _ := json.Marshal(GetOrganizationUserResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 30*time.Second)
	_, err := service.UpdateRole(ctx, orgID, userID, req)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationUserService.Remove tests
// ============================================================================

// TestOrganizationUserService_Remove_Success verifies that Remove calls
// DELETE /organizations/{orgID}/users/{userID} and returns nil on success.
func TestOrganizationUserService_Remove_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 204, Body: nil},
	}

	service := createTestOrganizationUserService(mock)
	err := service.Remove(context.Background(), orgID, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/users/" + userID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
}

// TestOrganizationUserService_Remove_InvalidOrgID verifies that Remove returns
// a descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationUserService_Remove_InvalidOrgID(t *testing.T) {
	userID := "11111111-2222-3333-4444-555555555555"
	tests := []struct {
		name  string
		orgID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			err := service.Remove(context.Background(), tc.orgID, userID)

			if err == nil {
				t.Fatal("expected error for invalid org ID, got nil")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_Remove_InvalidUserID verifies that Remove returns
// a descriptive error without calling the API for empty or malformed user IDs.
func TestOrganizationUserService_Remove_InvalidUserID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tests := []struct {
		name   string
		userID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationUserService(mock)

			err := service.Remove(context.Background(), orgID, tc.userID)

			if err == nil {
				t.Fatal("expected error for invalid user ID, got nil")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationUserService_Remove_NotFound verifies that a 404 API error is
// surfaced as *api.Error with IsNotFound() == true.
func TestOrganizationUserService_Remove_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "User not found"},
	}

	service := createTestOrganizationUserService(mock)
	err := service.Remove(context.Background(), orgID, userID)

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

// TestOrganizationUserService_Remove_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationUserService_Remove_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 204, Body: nil},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	err := service.Remove(context.Background(), orgID, userID)
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

// TestOrganizationUserService_Remove_QuickCancellation verifies that a
// pre-expired context causes Remove to fail immediately with a context error.
func TestOrganizationUserService_Remove_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	userID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 204, Body: nil},
		delay:    0,
	}

	service := createTestOrganizationUserServiceWithTimeout(mock, 30*time.Second)
	err := service.Remove(ctx, orgID, userID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// TestOrganizationUserService_Remove_InvalidUserID_ErrorMessage verifies that
// the error message for an invalid user ID contains the right text.
func TestOrganizationUserService_Remove_InvalidUserID_ErrorMessage(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{}
	service := createTestOrganizationUserService(mock)

	err := service.Remove(context.Background(), orgID, "")

	if err == nil {
		t.Fatal("expected error for empty user ID, got nil")
	}
	if !strings.Contains(err.Error(), "user ID") {
		t.Errorf("expected error to contain 'user ID', got: %v", err)
	}
}
