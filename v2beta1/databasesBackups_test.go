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

func createTestDatabaseBackupService(mock *mockAPIService) *databaseBackupService {
	return &databaseBackupService{api: mock, timeout: 30 * time.Second, logger: testLogger()}
}

func createTestDatabaseBackupServiceWithTimeout(mock api.RequestService, timeout time.Duration) *databaseBackupService {
	return &databaseBackupService{api: mock, timeout: timeout, logger: testLogger()}
}

func TestDatabaseListBackups_Success(t *testing.T) {
	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	expected := ListBackupsResponse{
		Data: []DatabaseBackup{
			{
				ID:         "bbbbbbbb-cccc-dddd-eeee-ffffffffffff",
				Timestamp:  "2026-06-29T00:00:00Z",
				Status:     BackupStatusCompleted,
				Exportable: true,
			},
			{
				ID:         "cccccccc-dddd-eeee-ffff-000000000000",
				Timestamp:  "2026-06-28T00:00:00Z",
				Status:     BackupStatusFailed,
				Exportable: false,
			},
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID, databaseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID + "/backups"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(result.Data))
	}
	if result.Data[0].ID != "bbbbbbbb-cccc-dddd-eeee-ffffffffffff" {
		t.Errorf("expected first backup ID 'bbbbbbbb-cccc-dddd-eeee-ffffffffffff', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Status != BackupStatusCompleted {
		t.Errorf("expected first backup status %q, got %q", BackupStatusCompleted, result.Data[0].Status)
	}
	if !result.Data[0].Exportable {
		t.Error("expected first backup to be exportable")
	}
	if result.Data[1].Status != BackupStatusFailed {
		t.Errorf("expected second backup status %q, got %q", BackupStatusFailed, result.Data[1].Status)
	}
}

func TestDatabaseListBackups_EmptyResult(t *testing.T) {
	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	body, _ := json.Marshal(ListBackupsResponse{Data: []DatabaseBackup{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID, databaseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 backups, got %d", len(result.Data))
	}
}

func TestDatabaseListBackups_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseBackupService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.List(context.Background(), tc.instanceID, databaseID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
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

func TestDatabaseListBackups_InvalidDatabaseID(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseBackupService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.List(context.Background(), instanceID, tc.databaseID)

			if err == nil {
				t.Fatal("expected error for invalid database ID, got nil")
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

func TestDatabaseListBackups_MissingOrgID(t *testing.T) {
	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseBackupService(mock)

	result, err := service.List(context.Background(), instanceID, databaseID)

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
		t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
	}
}

func TestDatabaseListBackups_MissingProjectID(t *testing.T) {
	const (
		orgID      = "11111111-aabb-3333-4444-555555555555"
		projectID  = "aaaaaaaa-aabb-3333-4444-ffff1111aaaa"
		instanceID = "a1b2c3d4"
		databaseID = "aa22ff99"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithOrganization(orgID),
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
		t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
	}
}

func TestDatabaseListBackups_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background(), instanceID, databaseID)

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

func TestDatabaseListBackups_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.List(context.Background(), instanceID, databaseID)

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

func TestDatabaseListBackups_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(ListBackupsResponse{Data: []DatabaseBackup{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseBackupServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.List(context.Background(), instanceID, databaseID)
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

func TestDatabaseListBackups_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(ListBackupsResponse{Data: []DatabaseBackup{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseBackupServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.List(ctx, instanceID, databaseID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

func TestDatabaseCreate_Success(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	expected := CreateBackupResponse{
		Data: DatabaseBackup{
			ID:         "dddddddd-eeee-ffff-0000-111111111111",
			Timestamp:  "2026-06-29T12:00:00Z",
			Status:     BackupStatusPending,
			Exportable: false,
		},
	}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), instanceID, databaseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID + "/databases/" + databaseID + "/backups"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if result.Data.ID != "dddddddd-eeee-ffff-0000-111111111111" {
		t.Errorf("expected backup ID 'dddddddd-eeee-ffff-0000-111111111111', got '%s'", result.Data.ID)
	}
	if result.Data.Status != BackupStatusPending {
		t.Errorf("expected backup status %q, got %q", BackupStatusPending, result.Data.Status)
	}
}

func TestDatabaseCreate_InvalidInstanceID(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		databaseID = "12345678"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseBackupService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Create(context.Background(), tc.instanceID, databaseID)

			if err == nil {
				t.Fatal("expected error for invalid instance ID, got nil")
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

func TestDatabaseCreate_InvalidDatabaseID(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
	}{
		{"empty", ""},
		{"non-UUID", "not-a-uuid"},
		{"short", "1234"},
	}

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
	)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestDatabaseBackupService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Create(context.Background(), instanceID, tc.databaseID)

			if err == nil {
				t.Fatal("expected error for invalid database ID, got nil")
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

func TestDatabaseCreate_MissingOrgID(t *testing.T) {
	const (
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	service := createTestDatabaseBackupService(mock)

	result, err := service.Create(context.Background(), instanceID, databaseID)

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
		t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
	}
}

func TestDatabaseCreate_MissingProjectID(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{}
	client, err := NewClient(
		WithCredentials("id", "secret"),
		WithOrganization(orgID),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	service := &databaseBackupService{
		api:     mock,
		timeout: 30 * time.Second,
		logger:  testLogger(),
		client:  client,
	}

	result, err := service.Create(context.Background(), instanceID, databaseID)

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
		t.Errorf("expected no API call, but got path '%s'", mock.lastPath)
	}
}

func TestDatabaseCreate_NotFound(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), instanceID, databaseID)

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

func TestDatabaseCreate_AuthenticationError(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestDatabaseBackupService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(context.Background(), instanceID, databaseID)

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

func TestDatabaseCreate_ContextTimeout(t *testing.T) {
	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(CreateBackupResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestDatabaseBackupServiceWithTimeout(mock, 10*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Create(context.Background(), instanceID, databaseID)
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

func TestDatabaseCreate_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	const (
		orgID      = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID  = "11111111-2222-3333-4444-555555555555"
		instanceID = "abcdef01"
		databaseID = "12345678"
	)

	body, _ := json.Marshal(CreateBackupResponse{})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestDatabaseBackupServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(ctx, instanceID, databaseID)

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
