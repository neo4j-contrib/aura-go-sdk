package v2beta1

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
)

// testLogger creates a logger for testing that writes warn+ to stderr.
func testLogger() *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// ============================================================================
// Mock types
// ============================================================================

// mockAPIService is a basic mock of api.RequestService.
// It records the last call details but does not inspect the context.
type mockAPIService struct {
	response   *api.Response
	err        error
	lastMethod string
	lastPath   string
	lastBody   string
}

// mockAPIServiceWithDelay is a mock that respects context cancellation/deadlines
// via executeWithDelay. mu guards recording fields for concurrent tests.
type mockAPIServiceWithDelay struct {
	mu       sync.Mutex
	response *api.Response
	err      error
	delay    time.Duration

	lastMethod string
	lastPath   string
	lastBody   string
	callCount  int
}

// ============================================================================
// mockAPIService — simple mock, does not check context
// ============================================================================

func (m *mockAPIService) Get(_ context.Context, endpoint string) (*api.Response, error) {
	m.lastMethod = "GET"
	m.lastPath = endpoint
	return m.response, m.err
}

func (m *mockAPIService) Post(_ context.Context, endpoint string, body string) (*api.Response, error) {
	m.lastMethod = "POST"
	m.lastPath = endpoint
	m.lastBody = body
	return m.response, m.err
}

func (m *mockAPIService) Put(_ context.Context, endpoint string, body string) (*api.Response, error) {
	m.lastMethod = "PUT"
	m.lastPath = endpoint
	m.lastBody = body
	return m.response, m.err
}

func (m *mockAPIService) Patch(_ context.Context, endpoint string, body string) (*api.Response, error) {
	m.lastMethod = "PATCH"
	m.lastPath = endpoint
	m.lastBody = body
	return m.response, m.err
}

func (m *mockAPIService) Delete(_ context.Context, endpoint string) (*api.Response, error) {
	m.lastMethod = "DELETE"
	m.lastPath = endpoint
	return m.response, m.err
}

func (m *mockAPIService) Close() {}

// ============================================================================
// mockAPIServiceWithDelay — respects context cancellation, can simulate slow APIs
// ============================================================================

func (m *mockAPIServiceWithDelay) Get(ctx context.Context, endpoint string) (*api.Response, error) {
	m.mu.Lock()
	m.lastMethod = "GET"
	m.lastPath = endpoint
	m.callCount++
	m.mu.Unlock()
	return m.executeWithDelay(ctx)
}

func (m *mockAPIServiceWithDelay) Post(ctx context.Context, endpoint string, body string) (*api.Response, error) {
	m.mu.Lock()
	m.lastMethod = "POST"
	m.lastPath = endpoint
	m.lastBody = body
	m.callCount++
	m.mu.Unlock()
	return m.executeWithDelay(ctx)
}

func (m *mockAPIServiceWithDelay) Put(ctx context.Context, endpoint string, body string) (*api.Response, error) {
	m.mu.Lock()
	m.lastMethod = "PUT"
	m.lastPath = endpoint
	m.lastBody = body
	m.callCount++
	m.mu.Unlock()
	return m.executeWithDelay(ctx)
}

func (m *mockAPIServiceWithDelay) Patch(ctx context.Context, endpoint string, body string) (*api.Response, error) {
	m.mu.Lock()
	m.lastMethod = "PATCH"
	m.lastPath = endpoint
	m.lastBody = body
	m.callCount++
	m.mu.Unlock()
	return m.executeWithDelay(ctx)
}

func (m *mockAPIServiceWithDelay) Delete(ctx context.Context, endpoint string) (*api.Response, error) {
	m.mu.Lock()
	m.lastMethod = "DELETE"
	m.lastPath = endpoint
	m.callCount++
	m.mu.Unlock()
	return m.executeWithDelay(ctx)
}

func (m *mockAPIServiceWithDelay) executeWithDelay(ctx context.Context) (*api.Response, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return m.response, m.err
}

func (m *mockAPIServiceWithDelay) Close() {}
