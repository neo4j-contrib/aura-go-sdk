package aura

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
)

// createTestGDSSessionService creates a gdsSessionService with a mock API service for testing
func createTestGDSSessionService(mock *mockAPIService) *gdsSessionService {
	return &gdsSessionService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
	}
}

// createTestGDSSessionServiceWithTimeout creates a gdsSessionService with a specific timeout.
// Pass the desired context directly to each method call.
func createTestGDSSessionServiceWithTimeout(mock api.RequestService, timeout time.Duration) *gdsSessionService {
	return &gdsSessionService{
		api:     mock,
		timeout: timeout,
		logger:  testLogger(),
	}
}

// TestGDSSessionService_List_Success verifies successful GDS session listing
func TestGDSSessionService_List_Success(t *testing.T) {
	expectedResponse := GetGDSSessionListResponse{
		Data: []GetGDSSessionData{
			{
				ID: "session-1", Name: "analytics-session-1", Memory: "8GB",
				InstanceID: "instance-1", DatabaseID: "db-uuid-1", Status: "running",
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Host: "session1.gds.neo4j.io",
				ExpiresAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), TTL: "24h", UserID: "user-1",
				TenantID: "tenant-1", CloudProvider: "gcp", Region: "us-central1",
			},
			{
				ID: "session-2", Name: "analytics-session-2", Memory: "16GB",
				InstanceID: "instance-2", Status: "stopped", CloudProvider: "aws", Region: "us-east-1",
			},
		},
	}

	responseBody, _ := json.Marshal(expectedResponse)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	if mock.lastPath != "graph-analytics/sessions" {
		t.Errorf("expected path 'graph-analytics/sessions', got '%s'", mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 GDS sessions, got %d", len(result.Data))
	}
	if result.Data[0].ID != "session-1" {
		t.Errorf("expected first session ID 'session-1', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Name != "analytics-session-1" {
		t.Errorf("expected first session name 'analytics-session-1', got '%s'", result.Data[0].Name)
	}
}

// TestGDSSessionService_List_EmptyResult verifies empty session list
func TestGDSSessionService_List_EmptyResult(t *testing.T) {
	responseBody, _ := json.Marshal(GetGDSSessionListResponse{Data: []GetGDSSessionData{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 GDS sessions, got %d", len(result.Data))
	}
}

// TestGDSSessionService_List_SingleSession verifies listing with single session
func TestGDSSessionService_List_SingleSession(t *testing.T) {
	responseBody, _ := json.Marshal(GetGDSSessionListResponse{
		Data: []GetGDSSessionData{
			{ID: "session-single", Name: "only-session", Memory: "32GB", Status: "running", CloudProvider: "gcp", Region: "europe-west2"},
		},
	})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 GDS session, got %d", len(result.Data))
	}
	if result.Data[0].ID != "session-single" {
		t.Errorf("expected session ID 'session-single', got '%s'", result.Data[0].ID)
	}
}

// TestGDSSessionService_List_MultipleStatuses verifies sessions with different statuses
func TestGDSSessionService_List_MultipleStatuses(t *testing.T) {
	responseBody, _ := json.Marshal(GetGDSSessionListResponse{
		Data: []GetGDSSessionData{
			{ID: "session-1", Status: "running"},
			{ID: "session-2", Status: "stopped"},
			{ID: "session-3", Status: "creating"},
			{ID: "session-4", Status: "failed"},
		},
	})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 4 {
		t.Errorf("expected 4 GDS sessions, got %d", len(result.Data))
	}

	statuses := make(map[string]bool)
	for _, session := range result.Data {
		statuses[session.Status] = true
	}
	for _, status := range []string{"running", "stopped", "creating", "failed"} {
		if !statuses[status] {
			t.Errorf("expected to find status '%s' in results", status)
		}
	}
}

// TestGDSSessionService_List_FullSessionDetails verifies all session fields
func TestGDSSessionService_List_FullSessionDetails(t *testing.T) {
	expectedSession := GetGDSSessionData{
		ID: "session-full", Name: "complete-session", Memory: "16GB",
		InstanceID: "instance-abc123", DatabaseID: "db-uuid-xyz789", Status: "running",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), Host: "session-full.gds.neo4j.io",
		ExpiresAt: time.Date(2024, 1, 22, 10, 30, 0, 0, time.UTC), TTL: "7d", UserID: "user-abc",
		TenantID: "tenant-xyz", CloudProvider: "gcp", Region: "europe-west2",
	}

	responseBody, _ := json.Marshal(GetGDSSessionListResponse{Data: []GetGDSSessionData{expectedSession}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: responseBody},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result.Data))
	}

	session := result.Data[0]
	if session.ID != expectedSession.ID {
		t.Errorf("expected ID '%s', got '%s'", expectedSession.ID, session.ID)
	}
	if session.Memory != expectedSession.Memory {
		t.Errorf("expected memory '%s', got '%s'", expectedSession.Memory, session.Memory)
	}
	if session.Status != expectedSession.Status {
		t.Errorf("expected status '%s', got '%s'", expectedSession.Status, session.Status)
	}
	if session.CloudProvider != expectedSession.CloudProvider {
		t.Errorf("expected cloud provider '%s', got '%s'", expectedSession.CloudProvider, session.CloudProvider)
	}
	if session.Region != expectedSession.Region {
		t.Errorf("expected region '%s', got '%s'", expectedSession.Region, session.Region)
	}
}

// TestGDSSessionService_List_AuthenticationError verifies auth error handling
func TestGDSSessionService_List_AuthenticationError(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: http.StatusUnauthorized, Message: "Invalid credentials"},
	}

	service := createTestGDSSessionService(mock)
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

// TestGDSSessionService_List_ServerError verifies server error handling
func TestGDSSessionService_List_ServerError(t *testing.T) {
	mock := &mockAPIService{
		err: &api.Error{StatusCode: http.StatusBadRequest, Message: "Bad request error"},
	}

	service := createTestGDSSessionService(mock)
	result, err := service.List(context.Background())

	if err == nil {
		t.Fatal("expected server error")
	}
	if result != nil {
		t.Error("expected result to be nil on error")
	}

	apiErr, ok := err.(*api.Error)
	if !ok {
		t.Fatal("expected Error type")
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

// TestGDSSessionService_List_ContextTimeout verifies timeout enforcement
func TestGDSSessionService_List_ContextTimeout(t *testing.T) {
	responseBody, _ := json.Marshal(GetGDSSessionListResponse{Data: []GetGDSSessionData{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: responseBody},
		delay:    2 * time.Second,
	}

	service := createTestGDSSessionServiceWithTimeout(mock, 100*time.Millisecond)

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
		t.Errorf("timeout took too long: %v", elapsed)
	}
}
