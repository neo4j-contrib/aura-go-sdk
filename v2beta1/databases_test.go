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

// createTestDatabaseServiceWithDefaultScope wires a client-level default scope so
// inherited-scope behaviour can be exercised.
func createTestDatabaseServiceWithDefaultScope(mock *mockAPIService, def requestScope) *databaseService {
	return &databaseService{api: mock, timeout: 30 * time.Second, logger: testLogger(), defaultScope: def}
}

// ============================================================================
// List tests
// ============================================================================

func TestDatabaseList_Success(t *testing.T) {
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
	result, err := service.List(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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
	result, err := service.List(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 databases, got %d", len(result.Data))
	}
}

func TestDatabaseList_DefaultScope(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: body}}

	service := createTestDatabaseServiceWithDefaultScope(mock, requestScope{
		orgID:      orgID,
		projectID:  projectID,
		instanceID: instanceID,
	})

	// No per-call options: the scope is inherited entirely from the default.
	_, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases"
	if mock.lastPath != expectedPath {
		t.Errorf("expected default-scope path %q, got %q", expectedPath, mock.lastPath)
	}
}

func TestDatabaseList_PerCallOverride(t *testing.T) {
	const (
		defaultOrg  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		defaultProj = "11111111-2222-3333-4444-555555555555"
		defaultInst = "abcdef01"
		otherOrg    = "dddddddd-eeee-ffff-aaaa-111122223333"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: body}}

	service := createTestDatabaseServiceWithDefaultScope(mock, requestScope{
		orgID:      defaultOrg,
		projectID:  defaultProj,
		instanceID: defaultInst,
	})

	// Only the org is overridden; project and instance still come from the default.
	_, err := service.List(context.Background(), WithOrg(otherOrg))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPath := "organizations/" + otherOrg + "/projects/" + defaultProj + "/instances/" + defaultInst + "/databases"
	if mock.lastPath != expectedPath {
		t.Errorf("expected overridden path %q, got %q", expectedPath, mock.lastPath)
	}
}

func TestDatabaseList_LastOptionWins(t *testing.T) {
	const (
		orgA = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		orgB = "cccccccc-dddd-eeee-ffff-000011112222"
		proj = "11111111-2222-3333-4444-555555555555"
		inst = "abcdef01"
	)

	body, _ := json.Marshal(ListDatabasesResponse{Data: []DatabaseResponse{}})
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: body}}

	service := createTestDatabaseService(mock)

	_, err := service.List(context.Background(),
		WithOrg(orgA), WithOrg(orgB),
		WithProject(proj), WithInstance(inst),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPath := "organizations/" + orgB + "/projects/" + proj + "/instances/" + inst + "/databases"
	if mock.lastPath != expectedPath {
		t.Errorf("expected last-option-wins path %q, got %q", expectedPath, mock.lastPath)
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

			result, err := service.List(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(tc.instanceID),
			)

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
	const (
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.List(context.Background(),
		WithProject(projectID), WithInstance(instanceID),
	)

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
	service := createTestDatabaseService(mock)

	result, err := service.List(context.Background(),
		WithOrg(orgID), WithInstance(instanceID),
	)

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
	result, err := service.List(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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
	_, err := service.List(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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

	start := time.Now()
	_, err := service.List(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)
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
	_, err := service.List(ctx,
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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
	result, err := service.Create(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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

			result, err := service.Create(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(tc.instanceID),
			)

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
	const (
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Create(context.Background(),
		WithProject(projectID), WithInstance(instanceID),
	)

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
	service := createTestDatabaseService(mock)

	result, err := service.Create(context.Background(),
		WithOrg(orgID), WithInstance(instanceID),
	)

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
	result, err := service.Create(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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
	_, err := service.Create(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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

	start := time.Now()
	_, err := service.Create(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)
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
	_, err := service.Create(ctx,
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID),
	)

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
	result, err := service.Get(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected database ID %q, got %q", "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", result.Data.ID)
	}
}

func TestDatabasesGet_DefaultScopeAndPerCallDatabase(t *testing.T) {
	const (
		org        = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		proj       = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	expected := GetDatabaseResponse{Data: DatabaseResponse{ID: "12345678"}}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: body}}

	service := createTestDatabaseServiceWithDefaultScope(mock, requestScope{
		orgID:      org,
		projectID:  proj,
		instanceID: instanceID,
	})

	// org/project/instance are inherited; only the leaf database is supplied
	// per call.
	result, err := service.Get(context.Background(), WithDatabase(databaseID))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPath := "organizations/" + org + "/projects/" + proj + "/instances/" + instanceID + "/databases/" + databaseID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "12345678" {
		t.Errorf("expected database ID %q, got %q", "12345678", result.Data.ID)
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

			result, err := service.Get(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(tc.instanceID), WithDatabase(databaseID),
			)

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

			result, err := service.Get(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(tc.databaseID),
			)

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
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Get(context.Background(),
		WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	service := createTestDatabaseService(mock)

	result, err := service.Get(context.Background(),
		WithOrg(orgID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	result, err := service.Get(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	_, err := service.Get(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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

	start := time.Now()
	_, err := service.Get(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)
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
	_, err := service.Get(ctx,
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
		databaseID = "123456aa"
	)

	expected := DeleteDatabaseResponse{
		Data: DatabaseResponse{ID: "123456aa"},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseService(mock)
	result, err := service.Delete(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}

	if result.Data.ID != "123456aa" {
		t.Errorf("expected database ID %q, got %q", "123456aa", result.Data.ID)
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

			result, err := service.Delete(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(tc.instanceID), WithDatabase(databaseID),
			)

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

			result, err := service.Delete(context.Background(),
				WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(tc.databaseID),
			)

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
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseService(mock)

	result, err := service.Delete(context.Background(),
		WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	service := createTestDatabaseService(mock)

	result, err := service.Delete(context.Background(),
		WithOrg(orgID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	result, err := service.Delete(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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
	_, err := service.Delete(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

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

	start := time.Now()
	_, err := service.Delete(context.Background(),
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)
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
	_, err := service.Delete(ctx,
		WithOrg(orgID), WithProject(projectID), WithInstance(instanceID), WithDatabase(databaseID),
	)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
