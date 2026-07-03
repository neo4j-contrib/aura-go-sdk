package v2beta1

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
)

// ============================================================================
// Constructor helpers
// ============================================================================

// createTestOrganizationInviteService creates an organizationInviteService with
// a mock API service for testing.
func createTestOrganizationInviteService(mock *mockAPIService) *organizationInviteService {
	return &organizationInviteService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestOrganizationInviteServiceWithTimeout creates an
// organizationInviteService with a specific timeout.
func createTestOrganizationInviteServiceWithTimeout(mock api.RequestService, timeout time.Duration) *organizationInviteService {
	return &organizationInviteService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// ============================================================================
// organizationInviteService.List tests
// ============================================================================

// TestOrganizationInviteService_List_Success verifies that List calls
// GET /organizations/{orgID}/invites and correctly maps all response fields.
func TestOrganizationInviteService_List_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	expected := ListOrganizationInvitesResponse{
		Data: []OrganizationInvite{
			{
				ID:                "11111111-2222-3333-4444-555555555555",
				Email:             "alice@example.com",
				InvitedBy:         "bob@example.com",
				OrganizationID:    orgID,
				OrganizationRoles: []string{"organization:admin"},
				Status:            "pending",
				ExpiresAt:         "2026-08-01T00:00:00Z",
			},
			{
				ID:                "66666666-7777-8888-9999-aaaaaaaaaaaa",
				Email:             "carol@example.com",
				InvitedBy:         "bob@example.com",
				OrganizationID:    orgID,
				OrganizationRoles: []string{"organization:member"},
				Status:            "pending",
				ExpiresAt:         "2026-08-01T00:00:00Z",
			},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationInviteService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/invites"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(result.Data))
	}
	if result.Data[0].ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected first invite ID '11111111-2222-3333-4444-555555555555', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Email != "alice@example.com" {
		t.Errorf("expected first invite email 'alice@example.com', got '%s'", result.Data[0].Email)
	}
	if result.Data[1].ID != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected second invite ID '66666666-7777-8888-9999-aaaaaaaaaaaa', got '%s'", result.Data[1].ID)
	}
}

// TestOrganizationInviteService_List_EmptyResult verifies that an empty invites
// list is returned without error.
func TestOrganizationInviteService_List_EmptyResult(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationInvitesResponse{Data: []OrganizationInvite{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestOrganizationInviteService(mock)
	result, err := service.List(context.Background(), orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 invites, got %d", len(result.Data))
	}
}

// TestOrganizationInviteService_List_InvalidOrgID verifies that List returns a
// descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationInviteService_List_InvalidOrgID(t *testing.T) {
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
			service := createTestOrganizationInviteService(mock)

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

// TestOrganizationInviteService_List_NotFound verifies that a 404 API error is
// returned as *api.Error with IsNotFound() == true.
func TestOrganizationInviteService_List_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestOrganizationInviteService(mock)
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

// TestOrganizationInviteService_List_AuthenticationError verifies that a 401
// API error exposes IsUnauthorized() == true.
func TestOrganizationInviteService_List_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationInviteService(mock)
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

// TestOrganizationInviteService_List_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationInviteService_List_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationInvitesResponse{Data: []OrganizationInvite{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 100*time.Millisecond)

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

// TestOrganizationInviteService_List_QuickCancellation verifies that a
// pre-expired context causes List to fail immediately with a context error.
func TestOrganizationInviteService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body, _ := json.Marshal(ListOrganizationInvitesResponse{Data: []OrganizationInvite{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 30*time.Second)
	_, err := service.List(ctx, orgID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationInviteService.Create tests
// ============================================================================

// TestOrganizationInviteService_Create_Success verifies that Create calls
// POST /organizations/{orgID}/invites with the correct serialised body and maps
// the response correctly.
func TestOrganizationInviteService_Create_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{
		Email: "alice@example.com",
		Roles: []string{"organization:admin"},
		ProjectInvites: []ProjectInviteEntry{
			{ProjectID: "66666666-7777-8888-9999-aaaaaaaaaaaa", ProjectRoles: []string{"project:viewer"}},
		},
	}

	expectedResp := CreateOrganizationInviteResponse{
		Data: OrganizationInvite{
			ID:                "11111111-2222-3333-4444-555555555555",
			Email:             "alice@example.com",
			OrganizationID:    orgID,
			OrganizationRoles: []string{"organization:admin"},
			Status:            "pending",
		},
	}

	body, _ := json.Marshal(expectedResp)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 201, Body: body},
	}

	service := createTestOrganizationInviteService(mock)
	result, err := service.Create(context.Background(), orgID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/invites"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected invite ID '11111111-2222-3333-4444-555555555555', got '%s'", result.Data.ID)
	}
	if result.Data.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got '%s'", result.Data.Email)
	}

	// Assert the request body was serialised correctly.
	var sentBody CreateOrganizationInviteRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentBody); err != nil {
		t.Fatalf("could not unmarshal sent body: %v", err)
	}
	if sentBody.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com' in body, got '%s'", sentBody.Email)
	}
	if len(sentBody.Roles) != 1 || sentBody.Roles[0] != "organization:admin" {
		t.Errorf("expected roles ['organization:admin'] in body, got %v", sentBody.Roles)
	}
	if len(sentBody.ProjectInvites) != 1 {
		t.Fatalf("expected 1 project invite in body, got %d", len(sentBody.ProjectInvites))
	}
	if sentBody.ProjectInvites[0].ProjectID != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected project ID '66666666-7777-8888-9999-aaaaaaaaaaaa' in body, got '%s'", sentBody.ProjectInvites[0].ProjectID)
	}
}

// TestOrganizationInviteService_Create_Success_OmitEmptyProjectInvites verifies
// that project_invites is omitted from the serialised body when the field is nil.
func TestOrganizationInviteService_Create_Success_OmitEmptyProjectInvites(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{
		Email: "bob@example.com",
		Roles: []string{"organization:member"},
		// ProjectInvites intentionally omitted (nil).
	}

	expectedResp := CreateOrganizationInviteResponse{
		Data: OrganizationInvite{
			ID:    "22222222-3333-4444-5555-666666666666",
			Email: "bob@example.com",
		},
	}

	body, _ := json.Marshal(expectedResp)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 201, Body: body},
	}

	service := createTestOrganizationInviteService(mock)
	_, err := service.Create(context.Background(), orgID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// project_invites key must be absent when nil (omitempty).
	var rawBody map[string]any
	if err := json.Unmarshal([]byte(mock.lastBody), &rawBody); err != nil {
		t.Fatalf("could not unmarshal sent body: %v", err)
	}
	if _, present := rawBody["project_invites"]; present {
		t.Error("expected project_invites to be omitted from body when nil, but it was present")
	}
}

// TestOrganizationInviteService_Create_InvalidOrgID verifies that Create returns
// a descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationInviteService_Create_InvalidOrgID(t *testing.T) {
	req := &CreateOrganizationInviteRequest{Email: "alice@example.com", Roles: []string{"organization:member"}}
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
			service := createTestOrganizationInviteService(mock)

			result, err := service.Create(context.Background(), tc.orgID, req)

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

// TestOrganizationInviteService_Create_NotFound verifies that a 404 API error
// is surfaced as *api.Error with IsNotFound() == true.
func TestOrganizationInviteService_Create_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{Email: "alice@example.com", Roles: []string{"organization:member"}}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Organization not found"},
	}

	service := createTestOrganizationInviteService(mock)
	result, err := service.Create(context.Background(), orgID, req)

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

// TestOrganizationInviteService_Create_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationInviteService_Create_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{Email: "alice@example.com", Roles: []string{"organization:member"}}
	body, _ := json.Marshal(CreateOrganizationInviteResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	_, err := service.Create(context.Background(), orgID, req)
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

// TestOrganizationInviteService_Create_AuthenticationError verifies that a 401
// API error exposes IsUnauthorized() == true.
func TestOrganizationInviteService_Create_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{Email: "alice@example.com", Roles: []string{"organization:member"}}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationInviteService(mock)
	_, err := service.Create(context.Background(), orgID, req)

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

// TestOrganizationInviteService_Create_QuickCancellation verifies that a
// pre-expired context causes Create to fail immediately with a context error.
func TestOrganizationInviteService_Create_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateOrganizationInviteRequest{Email: "alice@example.com", Roles: []string{"organization:member"}}
	body, _ := json.Marshal(CreateOrganizationInviteResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: body},
		delay:    0,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 30*time.Second)
	_, err := service.Create(ctx, orgID, req)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// organizationInviteService.Delete tests
// ============================================================================

// TestOrganizationInviteService_Delete_Success verifies that Delete calls
// DELETE /organizations/{orgID}/invites/{inviteID} and returns nil on success.
func TestOrganizationInviteService_Delete_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inviteID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 204, Body: nil},
	}

	service := createTestOrganizationInviteService(mock)
	err := service.Delete(context.Background(), orgID, inviteID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/invites/" + inviteID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
}

// TestOrganizationInviteService_Delete_InvalidOrgID verifies that Delete returns
// a descriptive error without calling the API for empty or malformed org IDs.
func TestOrganizationInviteService_Delete_InvalidOrgID(t *testing.T) {
	inviteID := "11111111-2222-3333-4444-555555555555"
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
			service := createTestOrganizationInviteService(mock)

			err := service.Delete(context.Background(), tc.orgID, inviteID)

			if err == nil {
				t.Fatal("expected error for invalid org ID, got nil")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationInviteService_Delete_InvalidInviteID verifies that Delete
// returns a descriptive error without calling the API for empty or malformed
// invite IDs.
func TestOrganizationInviteService_Delete_InvalidInviteID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tests := []struct {
		name     string
		inviteID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestOrganizationInviteService(mock)

			err := service.Delete(context.Background(), orgID, tc.inviteID)

			if err == nil {
				t.Fatal("expected error for invalid invite ID, got nil")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
			}
		})
	}
}

// TestOrganizationInviteService_Delete_NotFound verifies that a 404 API error
// is surfaced as *api.Error with IsNotFound() == true.
func TestOrganizationInviteService_Delete_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inviteID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Invite not found"},
	}

	service := createTestOrganizationInviteService(mock)
	err := service.Delete(context.Background(), orgID, inviteID)

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

// TestOrganizationInviteService_Delete_AuthenticationError verifies that a 401
// API error exposes IsUnauthorized() == true.
func TestOrganizationInviteService_Delete_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inviteID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestOrganizationInviteService(mock)
	err := service.Delete(context.Background(), orgID, inviteID)

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

// TestOrganizationInviteService_Delete_ContextTimeout verifies that the service
// timeout fires before the mock delay, returning context.DeadlineExceeded.
func TestOrganizationInviteService_Delete_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inviteID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 204, Body: nil},
		delay:    2 * time.Second,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 100*time.Millisecond)

	start := time.Now()
	err := service.Delete(context.Background(), orgID, inviteID)
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

// TestOrganizationInviteService_Delete_QuickCancellation verifies that a
// pre-expired context causes Delete to fail immediately with a context error.
func TestOrganizationInviteService_Delete_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let deadline expire.

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inviteID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 204, Body: nil},
		delay:    0,
	}

	service := createTestOrganizationInviteServiceWithTimeout(mock, 30*time.Second)
	err := service.Delete(ctx, orgID, inviteID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
