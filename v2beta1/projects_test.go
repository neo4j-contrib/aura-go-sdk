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
	result, err := service.List(context.Background(), WithOrg(orgID))

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
	result, err := service.List(context.Background(), WithOrg(orgID))

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
	result, err := service.List(context.Background(), WithOrg(orgID))

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
	_, err := service.List(context.Background(), WithOrg(orgID))

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
	result, err := service.List(context.Background(), WithOrg(""))

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
	_, err := service.List(context.Background(), WithOrg(orgID))
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

	_, err := service.List(ctx, WithOrg(orgID))

	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// ============================================================================
// Default scope precedence tests (project)
// ----------------------------------------------------------------------------
// Projects list is scoped through org. These prove the client default org is used
// when none is supplied per call, and that a per-call option overrides it.
// ============================================================================

func createTestProjectServiceWithDefaultScope(mock *mockAPIService, def requestScope) *projectService {
	return &projectService{api: mock, timeout: 30 * time.Second, logger: testLogger(), defaultScope: def}
}

func TestProjects_List_DefaultScope(t *testing.T) {
	const orgID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":[]}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: orgID})

	_, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgID + "/projects"
	if mock.lastPath != want {
		t.Errorf("expected default-scope path %q, got %q", want, mock.lastPath)
	}
}

func TestProjects_List_PerCallOverride(t *testing.T) {
	const (
		defaultOrg = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		otherOrg   = "cccccccc-dddd-eeee-ffff-000011112222"
	)

	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":[]}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: defaultOrg})

	_, err := service.List(context.Background(), WithOrg(otherOrg))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + otherOrg + "/projects"
	if mock.lastPath != want {
		t.Errorf("expected overridden path %q, got %q", want, mock.lastPath)
	}
}

func TestProjects_List_LastOptionWins(t *testing.T) {
	const (
		orgA = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		orgB = "cccccccc-dddd-eeee-ffff-000011112222"
	)

	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":[]}`)}}
	service := createTestProjectService(mock)

	_, err := service.List(context.Background(), WithOrg(orgA), WithOrg(orgB))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgB + "/projects"
	if mock.lastPath != want {
		t.Errorf("expected last-option-wins path %q, got %q", want, mock.lastPath)
	}
}

// ============================================================================
// projectService.Create tests
// ============================================================================

func TestProjectService_Create_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateProjectRequest{Name: "Operation Dark Koala", BillingAccountID: "b7c1e2d4-3f5a-4a6b-8c9d-0e1f2a3b4c5d", SSOConfigIDs: []string{"2f8d1c3e-6a4b-4e9f-b0c1-d2e3f4a5b6c7"}}
	expected := CreateProjectResponse{Data: Project{ID: "9aa289f6-1ccf-57c8-bc90-a0c3cecc82ac", Name: "Operation Dark Koala"}}

	body, _ := json.Marshal(expected)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: body}}

	service := createTestProjectService(mock)
	result, err := service.Create(context.Background(), req, WithOrg(orgID))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects"
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	var sentReq CreateProjectRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentReq); err != nil {
		t.Fatalf("failed to unmarshal sent body: %v", err)
	}
	if sentReq.Name != req.Name || sentReq.BillingAccountID != req.BillingAccountID {
		t.Errorf("sent request mismatch: got %+v, want %+v", sentReq, *req)
	}
	if result.Data.ID != expected.Data.ID || result.Data.Name != expected.Data.Name {
		t.Errorf("unexpected result: got %+v, want %+v", result.Data, expected.Data)
	}
}

func TestProjectService_Create_NilRequest(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestProjectService(mock)
	_, err := service.Create(context.Background(), nil, WithOrg("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	if err == nil || !strings.Contains(err.Error(), "request must not be nil") {
		t.Fatalf("expected nil request error, got %v", err)
	}
}

func TestProjectService_Create_MissingOrgID(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestProjectService(mock)
	_, err := service.Create(context.Background(), &CreateProjectRequest{Name: "x"}, WithOrg(""))
	if err == nil || !strings.Contains(err.Error(), "organization ID is required") {
		t.Fatalf("expected missing org ID error, got %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, got path %s", mock.lastPath)
	}
}

func TestProjectService_Create_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{err: &api.Error{StatusCode: 404, Message: "Not found"}}
	service := createTestProjectService(mock)
	_, err := service.Create(context.Background(), &CreateProjectRequest{Name: "x"}, WithOrg(orgID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsNotFound() {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestProjectService_Create_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIService{err: &api.Error{StatusCode: 401, Message: "Unauthorized"}}
	service := createTestProjectService(mock)
	_, err := service.Create(context.Background(), &CreateProjectRequest{Name: "x"}, WithOrg(orgID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsUnauthorized() {
		t.Errorf("expected Unauthorized error, got %v", err)
	}
}

func TestProjectService_Create_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mock := &mockAPIServiceWithDelay{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"1","name":"x"}}`)}, delay: 2 * time.Second}
	service := createTestProjectServiceWithTimeout(mock, 100*time.Millisecond)
	_, err := service.Create(context.Background(), &CreateProjectRequest{Name: "x"}, WithOrg(orgID))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestProjectService_Create_DefaultScope(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := &CreateProjectRequest{Name: "Test"}
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"1","name":"Test"}}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: orgID})
	_, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgID + "/projects"
	if mock.lastPath != want {
		t.Errorf("expected path %q, got %q", want, mock.lastPath)
	}
}

// ============================================================================
// projectService.Get tests
// ============================================================================

func TestProjectService_Get_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	expected := GetProjectResponse{
		Data: Project{ID: projectID, Name: "Operation Dark Koala"},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}
	service := createTestProjectService(mock)
	result, err := service.Get(context.Background(), WithOrg(orgID), WithProject(projectID))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "GET" {
		t.Errorf("expected GET method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	if result.Data.ID != projectID || result.Data.Name != "Operation Dark Koala" {
		t.Errorf("unexpected result: got %+v, want %+v", result.Data, expected.Data)
	}
}

func TestProjectService_Get_InvalidID(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestProjectService(mock)

	// Missing org ID
	_, err := service.Get(context.Background(), WithOrg(""), WithProject("11111111-2222-3333-4444-555555555555"))
	if err == nil || !strings.Contains(err.Error(), "organization ID is required") {
		t.Fatalf("expected missing org ID error, got %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, got path %s", mock.lastPath)
	}

	// Missing project ID
	_, err = service.Get(context.Background(), WithOrg("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"), WithProject(""))
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected missing project ID error, got %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, got path %s", mock.lastPath)
	}
}

func TestProjectService_Get_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}
	service := createTestProjectService(mock)
	_, err := service.Get(context.Background(), WithOrg(orgID), WithProject(projectID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsNotFound() {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestProjectService_Get_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Unauthorized"},
	}
	service := createTestProjectService(mock)
	_, err := service.Get(context.Background(), WithOrg(orgID), WithProject(projectID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsUnauthorized() {
		t.Errorf("expected Unauthorized error, got %v", err)
	}
}

func TestProjectService_Get_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"1","name":"x"}}`)},
		delay:    2 * time.Second,
	}
	service := createTestProjectServiceWithTimeout(mock, 100*time.Millisecond)
	_, err := service.Get(context.Background(), WithOrg(orgID), WithProject(projectID))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestProjectService_Get_QuickCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"1","name":"x"}}`)},
		delay:    0,
	}
	service := createTestProjectServiceWithTimeout(mock, 30*time.Second)
	_, err := service.Get(ctx, WithOrg(orgID), WithProject(projectID))
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got %v", err)
	}
}

// ============================================================================
// Default scope precedence tests (project Get)
// ============================================================================

func TestProjectService_Get_DefaultScope(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"` + projectID + `","name":"x"}}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: orgID, projectID: projectID})
	_, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgID + "/projects/" + projectID
	if mock.lastPath != want {
		t.Errorf("expected default-scope path %q, got %q", want, mock.lastPath)
	}
}

func TestProjectService_Get_PerCallOverride(t *testing.T) {
	const (
		defaultOrg     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		defaultProject = "11111111-2222-3333-4444-555555555555"
		otherOrg       = "cccccccc-dddd-eeee-ffff-000011112222"
		otherProject   = "22222222-3333-4444-5555-666666666666"
	)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"` + otherProject + `","name":"x"}}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: defaultOrg, projectID: defaultProject})
	_, err := service.Get(context.Background(), WithOrg(otherOrg), WithProject(otherProject))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + otherOrg + "/projects/" + otherProject
	if mock.lastPath != want {
		t.Errorf("expected overridden path %q, got %q", want, mock.lastPath)
	}
}

func TestProjectService_Get_LastOptionWins(t *testing.T) {
	const (
		orgA  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		orgB  = "cccccccc-dddd-eeee-ffff-000011112222"
		projA = "11111111-2222-3333-4444-555555555555"
		projB = "22222222-3333-4444-5555-666666666666"
	)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"` + projB + `","name":"x"}}`)}}
	service := createTestProjectService(mock)
	_, err := service.Get(context.Background(), WithOrg(orgA), WithProject(projA), WithOrg(orgB), WithProject(projB))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgB + "/projects/" + projB
	if mock.lastPath != want {
		t.Errorf("expected last-option-wins path %q, got %q", want, mock.lastPath)
	}
}

// ============================================================================
// projectService.Update tests
// ============================================================================

func TestProjectService_Update_Success(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	req := &UpdateProjectRequest{Name: "New Name"}
	expected := GetProjectResponse{
		Data: Project{ID: projectID, Name: "New Name"},
	}
	body, _ := json.Marshal(expected)
	mock := &mockAPIService{
		response: &api.Response{StatusCode: 200, Body: body},
	}
	service := createTestProjectService(mock)
	result, err := service.Update(context.Background(), req, WithOrg(orgID), WithProject(projectID))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastMethod != "PATCH" {
		t.Errorf("expected PATCH method, got %s", mock.lastMethod)
	}
	expectedPath := "organizations/" + orgID + "/projects/" + projectID
	if mock.lastPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, mock.lastPath)
	}
	var sentReq UpdateProjectRequest
	if err := json.Unmarshal([]byte(mock.lastBody), &sentReq); err != nil {
		t.Fatalf("failed to unmarshal sent body: %v", err)
	}
	if sentReq.Name != req.Name {
		t.Errorf("sent request mismatch: got %+v, want %+v", sentReq, *req)
	}
	if result.Data.ID != projectID || result.Data.Name != "New Name" {
		t.Errorf("unexpected result: got %+v, want %+v", result.Data, expected.Data)
	}
}

func TestProjectService_Update_NilRequest(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestProjectService(mock)
	_, err := service.Update(context.Background(), nil, WithOrg("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"), WithProject("11111111-2222-3333-4444-555555555555"))
	if err == nil || !strings.Contains(err.Error(), "request must not be nil") {
		t.Fatalf("expected nil request error, got %v", err)
	}
}

func TestProjectService_Update_InvalidID(t *testing.T) {
	mock := &mockAPIService{}
	service := createTestProjectService(mock)

	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg(""), WithProject("11111111-2222-3333-4444-555555555555"))
	if err == nil || !strings.Contains(err.Error(), "organization ID is required") {
		t.Fatalf("expected missing org ID error, got %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, got path %s", mock.lastPath)
	}

	_, err = service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"), WithProject(""))
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected missing project ID error, got %v", err)
	}
	if mock.lastPath != "" {
		t.Errorf("expected no API call, got path %s", mock.lastPath)
	}
}

func TestProjectService_Update_NotFound(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 404, Message: "Not found"},
	}
	service := createTestProjectService(mock)
	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg(orgID), WithProject(projectID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsNotFound() {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestProjectService_Update_AuthenticationError(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIService{
		err: &api.Error{StatusCode: 401, Message: "Unauthorized"},
	}
	service := createTestProjectService(mock)
	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg(orgID), WithProject(projectID))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*api.Error)
	if !ok || !apiErr.IsUnauthorized() {
		t.Errorf("expected Unauthorized error, got %v", err)
	}
}

func TestProjectService_Update_ContextTimeout(t *testing.T) {
	orgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectID := "11111111-2222-3333-4444-555555555555"
	mock := &mockAPIServiceWithDelay{
		response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"1","name":"x"}}`)},
		delay:    2 * time.Second,
	}
	service := createTestProjectServiceWithTimeout(mock, 100*time.Millisecond)
	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg(orgID), WithProject(projectID))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestProjectService_Update_DefaultScope(t *testing.T) {
	const (
		orgID     = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		projectID = "11111111-2222-3333-4444-555555555555"
	)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"` + projectID + `","name":"x"}}`)}}
	service := createTestProjectServiceWithDefaultScope(mock, requestScope{orgID: orgID, projectID: projectID})
	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "New"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgID + "/projects/" + projectID
	if mock.lastPath != want {
		t.Errorf("expected default-scope path %q, got %q", want, mock.lastPath)
	}
}

func TestProjectService_Update_LastOptionWins(t *testing.T) {
	const (
		orgA  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		orgB  = "cccccccc-dddd-eeee-ffff-000011112222"
		projA = "11111111-2222-3333-4444-555555555555"
		projB = "22222222-3333-4444-5555-666666666666"
	)
	mock := &mockAPIService{response: &api.Response{StatusCode: 200, Body: []byte(`{"data":{"id":"` + projB + `","name":"x"}}`)}}
	service := createTestProjectService(mock)
	_, err := service.Update(context.Background(), &UpdateProjectRequest{Name: "x"}, WithOrg(orgA), WithProject(projA), WithOrg(orgB), WithProject(projB))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "organizations/" + orgB + "/projects/" + projB
	if mock.lastPath != want {
		t.Errorf("expected last-option-wins path %q, got %q", want, mock.lastPath)
	}
}
