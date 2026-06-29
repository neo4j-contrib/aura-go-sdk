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

// createTestInstanceService creates an instanceService with a mock API service
// for testing. No client pointer is set so defaultOrgID and defaultProjectID are always "".
func createTestInstanceService(mock *mockAPIService) *instanceService {
	return &instanceService{api: mock, timeout: 30 * time.Second, logger: testLogger()}
}

// createTestInstanceServiceWithTimeout creates an instanceService with a specific
// timeout. Pass the desired context directly to each method call.
func createTestInstanceServiceWithTimeout(mock api.RequestService, timeout time.Duration) *instanceService {
	return &instanceService{api: mock, timeout: timeout, logger: testLogger()}
}

// ============================================================================
// instanceService.List tests
// ============================================================================

func TestInstanceService_List_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"

	expected := ListInstancesResponse{
		Data: []InstanceSummary{
			{ID: instanceID, Name: "test-instance", Status: InstanceStatusRunning, CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB"},
		},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(result.Data))
	}
	if result.Data[0].ID != instanceID {
		t.Errorf("expected instance ID %q, got %q", instanceID, result.Data[0].ID)
	}
	if result.Data[0].Name != "test-instance" {
		t.Errorf("expected instance name %q, got %q", "test-instance", result.Data[0].Name)
	}
	if result.Data[0].Status != InstanceStatusRunning {
		t.Errorf("expected status %q, got %q", InstanceStatusRunning, result.Data[0].Status)
	}
}

func TestInstanceService_List_EmptyResult(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"

	body, _ := json.Marshal(ListInstancesResponse{Data: []InstanceSummary{}})
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected 0 instances, got %d", len(result.Data))
	}
}

func TestInstanceService_List_MissingOrgID(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)

	result, err := service.List(context.Background())

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to mention 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_List_MissingProjectID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID}

	result, err := service.List(context.Background())

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to mention 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_List_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.List(context.Background())

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

func TestInstanceService_List_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

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

func TestInstanceService_List_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	body, _ := json.Marshal(ListInstancesResponse{Data: []InstanceSummary{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestInstanceServiceWithTimeout(mock, 100*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

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

func TestInstanceService_List_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	body, _ := json.Marshal(ListInstancesResponse{Data: []InstanceSummary{}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestInstanceServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.List(ctx)

	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// instanceService.Get tests
// ============================================================================

func TestInstanceService_Get_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"

	expected := GetInstanceResponse{
		Data: InstanceDetails{
			ID:            instanceID,
			Name:          "prod-instance",
			Status:        InstanceStatusRunning,
			CloudProvider: "gcp",
			Region:        "us-central1",
			Type:          "enterprise-db",
			Memory:        "8GB",
			ConnectionURL: "neo4j+s://abcdef.databases.neo4j.io",
		},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Get(context.Background(), instanceID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != instanceID {
		t.Errorf("expected instance ID %q, got %q", instanceID, result.Data.ID)
	}
	if result.Data.Name != "prod-instance" {
		t.Errorf("expected instance name %q, got %q", "prod-instance", result.Data.Name)
	}
	if result.Data.ConnectionURL != "neo4j+s://abcdef.databases.neo4j.io" {
		t.Errorf("expected connection URL %q, got %q", "neo4j+s://abcdef.databases.neo4j.io", result.Data.ConnectionURL)
	}
}

func TestInstanceService_Get_InvalidID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"

	cases := []struct {
		name       string
		instanceID string
	}{
		{name: "empty", instanceID: ""},
		{name: "not-a-uuid", instanceID: "not-a-uuid"},
		{name: "too-short", instanceID: "abc123"},
		{name: "wrong-format", instanceID: "aaaaaaaa-bbbb-cccc-dddd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestInstanceService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Get(context.Background(), tc.instanceID)

			if err == nil {
				t.Fatalf("expected error for instanceID %q, got nil", tc.instanceID)
			}
			if result != nil {
				t.Error("expected nil result for invalid instance ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for invalid instanceID %q, but got path %q", tc.instanceID, mock.lastPath)
			}
		})
	}
}

func TestInstanceService_Get_MissingOrgID(t *testing.T) {
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)

	result, err := service.Get(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to mention 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Get_MissingProjectID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID}

	result, err := service.Get(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to mention 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Get_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Instance not found"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Get(context.Background(), instanceID)

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

func TestInstanceService_Get_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Get(context.Background(), instanceID)

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

func TestInstanceService_Get_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	body, _ := json.Marshal(GetInstanceResponse{Data: InstanceDetails{ID: instanceID}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    2 * time.Second,
	}

	service := createTestInstanceServiceWithTimeout(mock, 100*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Get(context.Background(), instanceID)
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

func TestInstanceService_Get_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	body, _ := json.Marshal(GetInstanceResponse{Data: InstanceDetails{ID: instanceID}})
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: body},
		delay:    0,
	}

	service := createTestInstanceServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Get(ctx, instanceID)

	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// instanceService.Create tests
// ============================================================================

func TestInstanceService_Create_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"

	req := &CreateInstanceRequest{
		Name:          "new-instance",
		CloudProvider: "aws",
		Region:        "us-east-1",
		Type:          "enterprise-db",
		Memory:        "4GB",
	}

	expected := CreateInstanceResponse{
		Data: CreateInstanceData{
			ID:            instanceID,
			Name:          "new-instance",
			CloudProvider: "aws",
			Region:        "us-east-1",
			Type:          "enterprise-db",
			Username:      "neo4j",
			Password:      "secret",
			ConnectionURL: "neo4j+s://abcdef.databases.neo4j.io",
		},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 201, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}

	var sentReq CreateInstanceRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentReq); err != nil {
		t.Fatalf("failed to unmarshal mock.lastBody: %v", err)
	}
	if sentReq.Name != "new-instance" {
		t.Errorf("expected request name %q, got %q", "new-instance", sentReq.Name)
	}
	if sentReq.CloudProvider != "aws" {
		t.Errorf("expected request cloud_provider %q, got %q", "aws", sentReq.CloudProvider)
	}
	if sentReq.Region != "us-east-1" {
		t.Errorf("expected request region %q, got %q", "us-east-1", sentReq.Region)
	}
	if sentReq.Type != "enterprise-db" {
		t.Errorf("expected request type %q, got %q", "enterprise-db", sentReq.Type)
	}
	if sentReq.Memory != "4GB" {
		t.Errorf("expected request memory %q, got %q", "4GB", sentReq.Memory)
	}

	if result.Data.ID != instanceID {
		t.Errorf("expected instance ID %q, got %q", instanceID, result.Data.ID)
	}
	if result.Data.Password != "secret" {
		t.Errorf("expected password %q, got %q", "secret", result.Data.Password)
	}
}

func TestInstanceService_Create_NilRequest(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), nil)

	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
	if result != nil {
		t.Error("expected nil result for nil request")
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Create_InvalidRequest(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"

	cases := []struct {
		name string
		req  *CreateInstanceRequest
	}{
		{
			name: "empty name",
			req:  &CreateInstanceRequest{CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB"},
		},
		{
			name: "empty cloud_provider",
			req:  &CreateInstanceRequest{Name: "test", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB"},
		},
		{
			name: "empty region",
			req:  &CreateInstanceRequest{Name: "test", CloudProvider: "aws", Type: "enterprise-db", Memory: "4GB"},
		},
		{
			name: "empty type",
			req:  &CreateInstanceRequest{Name: "test", CloudProvider: "aws", Region: "us-east-1", Memory: "4GB"},
		},
		{
			name: "empty memory",
			req:  &CreateInstanceRequest{Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestInstanceService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Create(context.Background(), tc.req)

			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if result != nil {
				t.Error("expected nil result for invalid request")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for %s, but got path %q", tc.name, mock.lastPath)
			}
		})
	}
}

func TestInstanceService_Create_MissingOrgID(t *testing.T) {
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)

	result, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to mention 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Create_MissingProjectID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID}

	result, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to mention 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Create_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Project not found"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Create(context.Background(), req)

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

func TestInstanceService_Create_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Invalid credentials"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(context.Background(), req)

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

func TestInstanceService_Create_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: []byte(`{}`)},
		delay:    2 * time.Second,
	}

	service := createTestInstanceServiceWithTimeout(mock, 100*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Create(context.Background(), req)
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

func TestInstanceService_Create_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &CreateInstanceRequest{
		Name: "test", CloudProvider: "aws", Region: "us-east-1", Type: "enterprise-db", Memory: "4GB",
	}
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 201, Body: []byte(`{}`)},
		delay:    0,
	}

	service := createTestInstanceServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Create(ctx, req)

	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// instanceService.Update tests
// ============================================================================

func TestInstanceService_Update_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"

	req := &UpdateInstanceRequest{
		Name:   "updated-name",
		Memory: "8GB",
	}

	expected := GetInstanceResponse{
		Data: InstanceDetails{
			ID:     instanceID,
			Name:   "updated-name",
			Memory: "8GB",
			Status: InstanceStatusUpdating,
		},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Update(context.Background(), instanceID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "PATCH" {
		t.Errorf("expected PATCH method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}

	var sentReq UpdateInstanceRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentReq); err != nil {
		t.Fatalf("failed to unmarshal mock.lastBody: %v", err)
	}
	if sentReq.Name != "updated-name" {
		t.Errorf("expected request name %q, got %q", "updated-name", sentReq.Name)
	}
	if sentReq.Memory != "8GB" {
		t.Errorf("expected request memory %q, got %q", "8GB", sentReq.Memory)
	}

	if result.Data.ID != instanceID {
		t.Errorf("expected instance ID %q, got %q", instanceID, result.Data.ID)
	}
}

func TestInstanceService_Update_InvalidID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateInstanceRequest{Name: "updated"}

	cases := []struct {
		name       string
		instanceID string
	}{
		{name: "empty", instanceID: ""},
		{name: "not-a-uuid", instanceID: "not-a-uuid"},
		{name: "too-short", instanceID: "abc123"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestInstanceService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Update(context.Background(), tc.instanceID, req)

			if err == nil {
				t.Fatalf("expected error for instanceID %q, got nil", tc.instanceID)
			}
			if result != nil {
				t.Error("expected nil result for invalid instance ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for instanceID %q, but got path %q", tc.instanceID, mock.lastPath)
			}
		})
	}
}

func TestInstanceService_Update_NilRequest(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Update(context.Background(), instanceID, nil)

	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
	if result != nil {
		t.Error("expected nil result for nil request")
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Update_MissingOrgID(t *testing.T) {
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	req := &UpdateInstanceRequest{Name: "updated"}
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)

	result, err := service.Update(context.Background(), instanceID, req)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to mention 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Update_MissingProjectID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	req := &UpdateInstanceRequest{Name: "updated"}
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID}

	result, err := service.Update(context.Background(), instanceID, req)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to mention 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Update_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	req := &UpdateInstanceRequest{Name: "updated"}
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{}`)},
		delay:    2 * time.Second,
	}

	service := createTestInstanceServiceWithTimeout(mock, 100*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Update(context.Background(), instanceID, req)
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

func TestInstanceService_Update_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	req := &UpdateInstanceRequest{Name: "updated"}
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{}`)},
		delay:    0,
	}

	service := createTestInstanceServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Update(ctx, instanceID, req)

	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// instanceService.Delete tests
// ============================================================================

func TestInstanceService_Delete_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"

	expected := DeleteInstanceResponse{
		Data: InstanceDetails{ID: instanceID, Status: InstanceStatusDestroying},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Delete(context.Background(), instanceID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID + "/instances/" + instanceID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, mock.lastPath)
	}
	if result.Data.ID != instanceID {
		t.Errorf("expected instance ID %q, got %q", instanceID, result.Data.ID)
	}
}

func TestInstanceService_Delete_InvalidID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"

	cases := []struct {
		name       string
		instanceID string
	}{
		{name: "empty", instanceID: ""},
		{name: "not-a-uuid", instanceID: "not-a-uuid"},
		{name: "too-short", instanceID: "abc123"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockAPIService{}
			service := createTestInstanceService(mock)
			service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

			result, err := service.Delete(context.Background(), tc.instanceID)

			if err == nil {
				t.Fatalf("expected error for instanceID %q, got nil", tc.instanceID)
			}
			if result != nil {
				t.Error("expected nil result for invalid instance ID")
			}
			if mock.lastPath != "" {
				t.Errorf("expected no API call for instanceID %q, but got path %q", tc.instanceID, mock.lastPath)
			}
		})
	}
}

func TestInstanceService_Delete_MissingOrgID(t *testing.T) {
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)

	result, err := service.Delete(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing org ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when org ID is missing")
	}
	if !strings.Contains(err.Error(), "organization ID") {
		t.Errorf("expected error to mention 'organization ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Delete_MissingProjectID(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{}
	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID}

	result, err := service.Delete(context.Background(), instanceID)

	if err == nil {
		t.Fatal("expected error for missing project ID, got nil")
	}
	if result != nil {
		t.Error("expected nil result when project ID is missing")
	}
	if !strings.Contains(err.Error(), "project ID") {
		t.Errorf("expected error to mention 'project ID', got: %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, but got path %q", mock.lastPath)
	}
}

func TestInstanceService_Delete_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Instance not found"},
	}

	service := createTestInstanceService(mock)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	result, err := service.Delete(context.Background(), instanceID)

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

func TestInstanceService_Delete_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{}`)},
		delay:    2 * time.Second,
	}

	service := createTestInstanceServiceWithTimeout(mock, 100*time.Millisecond)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	start := time.Now()
	_, err := service.Delete(context.Background(), instanceID)
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

func TestInstanceService_Delete_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	instanceID := "cccccccc-dddd-eeee-ffff-000000000000"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{}`)},
		delay:    0,
	}

	service := createTestInstanceServiceWithTimeout(mock, 30*time.Second)
	service.client = &Client{defaultOrgID: orgID, defaultProjectID: projectID}

	_, err := service.Delete(ctx, instanceID)

	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}
