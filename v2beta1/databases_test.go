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

func createTestDatabaseService(mock *mockAPIService) *databaseService {
	return &databaseService{api: mock, timeout: 30 * time.Second, logger: testLogger()}
}

func createTestDatabaseServiceWithTimeout(mock api.RequestService, timeout time.Duration) *databaseService {
	return &databaseService{api: mock, timeout: timeout, logger: testLogger()}
}

func TestDatabaseList_Success(t *testing.T) {
	// orgID/projectID must be valid UUIDs; instanceID must be 8-char hex.
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	expected := ListDatabasesResponse{
		Data: []DatabaseResponse{
			{ID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"},
			{ID: "cccccccc-dddd-eeee-ffff-000000000000"},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(result.Data))
	}
	if result.Data[0].ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected first database ID %q, got %q", "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", result.Data[0].ID)
	}
	if result.Data[1].ID != "cccccccc-dddd-eeee-ffff-000000000000" {
		t.Errorf("expected second database ID %q, got %q", "cccccccc-dddd-eeee-ffff-000000000000", result.Data[1].ID)
	}
}

func TestDatabaseList_EmptyResult(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 databases, got %d", len(result.Data))
	}
}

func TestDatabaseList_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.List(context.Background(), tc.instanceID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabaseList_MissingOrgID(t *testing.T) {
	const instanceID = "abcdef01"

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)
	// no client set — resolveOrgProject returns empty strings

	result, err := service.List(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to contain 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabaseList_MissingProjectID(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithDefaultOrg(orgID),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	service := &databaseService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
		client:  client,
	}

	result, err := service.List(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to contain 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabaseList_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID)

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

func TestDatabaseList_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.List(context.Background(), instanceID)

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

func TestDatabaseList_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.List(context.Background(), instanceID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestDatabaseList_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.List(ctx, instanceID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// Create tests
// ============================================================================

func TestDatabasesCreate_Success(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	expected := CreateDatabaseResponse{
		Data: DatabaseResponse{ID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), instanceID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected database ID %q, got %q", "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", result.Data.ID)
	}
}

func TestDatabasesCreate_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Create(context.Background(), tc.instanceID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabasesCreate_MissingOrgID(t *testing.T) {
	const instanceID = "abcdef01"

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Create(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to contain 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesCreate_MissingProjectID(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithDefaultOrg(orgID),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	service := &databaseService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
		client:  client,
	}

	result, err := service.Create(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to contain 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesCreate_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), instanceID)

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

func TestDatabasesCreate_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(context.Background(), instanceID)

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

func TestDatabasesCreate_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(CreateDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Create(context.Background(), instanceID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestDatabasesCreate_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(CreateDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(ctx, instanceID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// Get tests
// ============================================================================

func TestDatabasesGet_Success(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	expected := GetDatabaseResponse{
		Data: DatabaseResponse{ID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Get(context.Background(), instanceID, databaseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID + "/backups"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected database ID %q, got %q", "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", result.Data.ID)
	}
}

func TestDatabasesGet_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		databaseID = "12345678"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Get(context.Background(), tc.instanceID, databaseID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabasesGet_InvalidDatabaseID(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Get(context.Background(), instanceID, tc.databaseID)

			if err == nil {
				t.Fatal("expected error for invalid database ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabasesGet_MissingOrgID(t *testing.T) {
	const (
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Get(context.Background(), instanceID, databaseID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to contain 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesGet_MissingProjectID(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithDefaultOrg(orgID),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	service := &databaseService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
		client:  client,
	}

	result, err := service.Get(context.Background(), instanceID, databaseID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to contain 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesGet_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Get(context.Background(), instanceID, databaseID)

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

func TestDatabasesGet_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Get(context.Background(), instanceID, databaseID)

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

func TestDatabasesGet_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(GetDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Get(context.Background(), instanceID, databaseID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestDatabasesGet_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(GetDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Get(ctx, instanceID, databaseID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// Delete tests
// ============================================================================

func TestDatabasesDelete_Success(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	expected := DeleteDatabaseResponse{
		Data: []DatabaseResponse{
			{ID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Delete(context.Background(), instanceID, databaseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID + "/backups"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 database in response, got %d", len(result.Data))
	}
	if result.Data[0].ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected database ID %q, got %q", "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", result.Data[0].ID)
	}
}

func TestDatabasesDelete_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		databaseID = "12345678"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Delete(context.Background(), tc.instanceID, databaseID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabasesDelete_InvalidDatabaseID(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
	}{
		{"empty", ""},
		{"non-hex", "not-hex!"},
		{"too-short", "1234"},
		{"too-long", "abcdef012"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Delete(context.Background(), instanceID, tc.databaseID)

			if err == nil {
				t.Fatal("expected error for invalid database ID, got nil")
			}
			if result != nil {
				t.Error("expected nil result on error")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call, but got path %q", mock.lastPath)
			}
		})
	}
}

func TestDatabasesDelete_MissingOrgID(t *testing.T) {
	const (
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Delete(context.Background(), instanceID, databaseID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to contain 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesDelete_MissingProjectID(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithDefaultOrg(orgID),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	service := &databaseService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
		client:  client,
	}

	result, err := service.Delete(context.Background(), instanceID, databaseID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to contain 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestDatabasesDelete_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Delete(context.Background(), instanceID, databaseID)

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

func TestDatabasesDelete_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Delete(context.Background(), instanceID, databaseID)

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

func TestDatabasesDelete_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(DeleteDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Delete(context.Background(), instanceID, databaseID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestDatabasesDelete_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(DeleteDatabaseResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Delete(ctx, instanceID, databaseID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
