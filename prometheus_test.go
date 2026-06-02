package aura

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/v2/internal/api"
)

// newTestPrometheusService creates a prometheusService for unit testing.
// It does not make real network calls; tests that exercise parsing
// or health logic call internal methods directly.
func newTestPrometheusService() *prometheusService {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	apiSvc := api.NewRequestService(api.Config{
		ClientID:     "test",
		ClientSecret: "test",
		BaseURL:      "https://api.neo4j.io",
		APIVersion:   "v1",
	}, logger)

	return &prometheusService{
		api:     apiSvc,
		timeout: 30 * time.Second,
		logger:  logger,
	}
}

func TestPrometheusService_FetchRawMetrics(t *testing.T) {
	svc := newTestPrometheusService()
	ctx := context.Background()

	t.Run("EmptyURL", func(t *testing.T) {
		_, err := svc.FetchRawMetrics(ctx, "")
		if err == nil {
			t.Error("Expected error for empty URL, got nil")
		}
	})
}

func TestPrometheusService_ParsePrometheusMetrics(t *testing.T) {
	svc := newTestPrometheusService()

	tests := []struct {
		name          string
		input         string
		expectedName  string
		expectedValue float64
		expectError   bool
	}{
		{
			name: "Valid gauge metric",
			input: `# HELP neo4j_aura_cpu_usage CPU usage
# TYPE neo4j_aura_cpu_usage gauge
neo4j_aura_cpu_usage{aggregation="MAX",availability_zone="europe-west2-c",instance_mode="PRIMARY",instance_id="c9f0d13a"} 0.023206 1769766720469
`,
			expectedName:  "neo4j_aura_cpu_usage",
			expectedValue: 0.023206,
		},
		{
			name: "Valid counter metric",
			input: `# HELP neo4j_database_count_node Node count
# TYPE neo4j_database_count_node counter
neo4j_database_count_node{database="neo4j",instance_id="c9f0d13a"} 171.000000 1769766720469
`,
			expectedName:  "neo4j_database_count_node",
			expectedValue: 171.0,
		},
		{
			name: "Multiple metrics",
			input: `# HELP test_metric Test metric
# TYPE test_metric gauge
test_metric{label="a"} 10.0
test_metric{label="b"} 20.0
`,
			expectedName:  "test_metric",
			expectedValue: 10.0,
		},
		{
			name:  "Empty input",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.parsePrometheusMetrics([]byte(tt.input))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if tt.expectedName != "" {
				metrics, ok := result.Metrics[tt.expectedName]
				if !ok {
					t.Fatalf("Expected metric %s not found", tt.expectedName)
				}
				if len(metrics) == 0 {
					t.Fatal("Expected at least one metric value")
				}
				if metrics[0].Value != tt.expectedValue {
					t.Errorf("Expected value %f, got %f", tt.expectedValue, metrics[0].Value)
				}
			}
		})
	}
}

func TestPrometheusService_GetInstanceHealth(t *testing.T) {
	svc := newTestPrometheusService()
	ctx := context.Background()

	t.Run("InvalidInstanceID", func(t *testing.T) {
		_, err := svc.GetInstanceHealth(ctx, "", "https://example.com/prometheus")
		if err == nil {
			t.Error("Expected error for empty instance ID, got nil")
		}
	})

	t.Run("EmptyPrometheusURL", func(t *testing.T) {
		_, err := svc.GetInstanceHealth(ctx, "c9f0d13a", "")
		if err == nil {
			t.Error("Expected error for empty Prometheus URL, got nil")
		}
	})
}

func TestPrometheusService_GetMetricValue(t *testing.T) {
	svc := newTestPrometheusService()
	ctx := context.Background()

	testMetrics := &PrometheusMetricsResponse{
		Metrics: map[string][]PrometheusMetric{
			"test_metric": {
				{Name: "test_metric", Value: 10.0, Labels: map[string]string{"zone": "zone-a", "type": "primary"}},
				{Name: "test_metric", Value: 20.0, Labels: map[string]string{"zone": "zone-b", "type": "primary"}},
				{Name: "test_metric", Value: 15.0, Labels: map[string]string{"zone": "zone-c", "type": "secondary"}},
			},
		},
	}

	t.Run("Average all metrics", func(t *testing.T) {
		value, err := svc.GetMetricValue(ctx, testMetrics, "test_metric", nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expected := (10.0 + 20.0 + 15.0) / 3.0
		if value != expected {
			t.Errorf("Expected average %f, got %f", expected, value)
		}
	})

	t.Run("Filter by label", func(t *testing.T) {
		value, err := svc.GetMetricValue(ctx, testMetrics, "test_metric", map[string]string{"type": "primary"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expected := (10.0 + 20.0) / 2.0
		if value != expected {
			t.Errorf("Expected average %f, got %f", expected, value)
		}
	})

	t.Run("Filter by multiple labels", func(t *testing.T) {
		value, err := svc.GetMetricValue(ctx, testMetrics, "test_metric", map[string]string{"zone": "zone-a", "type": "primary"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if value != 10.0 {
			t.Errorf("Expected value 10.0, got %f", value)
		}
	})

	t.Run("Metric not found", func(t *testing.T) {
		_, err := svc.GetMetricValue(ctx, testMetrics, "nonexistent_metric", nil)
		if err == nil {
			t.Error("Expected error for nonexistent metric, got nil")
		}
	})

	t.Run("No matching labels", func(t *testing.T) {
		_, err := svc.GetMetricValue(ctx, testMetrics, "test_metric", map[string]string{"zone": "zone-d"})
		if err == nil {
			t.Error("Expected error for non-matching label filter, got nil")
		}
	})
}

func TestAssessHealth(t *testing.T) {
	svc := newTestPrometheusService()

	tests := []struct {
		name           string
		metrics        *PrometheusHealthMetrics
		expectedStatus string
		expectIssues   bool
	}{
		{
			name: "Healthy system",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "healthy",
		},
		{
			name: "High CPU",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 85, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "warning",
			expectIssues:   true,
		},
		{
			name: "Critical CPU",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 97, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "critical",
			expectIssues:   true,
		},
		{
			name: "High Memory",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 90},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "warning",
			expectIssues:   true,
		},
		{
			name: "Critical Memory",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 97},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "critical",
			expectIssues:   true,
		},
		{
			name: "Low Page Cache",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 30},
			},
			expectedStatus: "warning",
			expectIssues:   true,
		},
		{
			name: "Critical Page Cache",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 10},
			},
			expectedStatus: "critical",
			expectIssues:   true,
		},
		{
			name: "High connections warning",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 85, MaxConnections: 100, UsagePercent: 85},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "warning",
			expectIssues:   true,
		},
		{
			name: "Critical connections",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 97, MaxConnections: 100, UsagePercent: 97},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "critical",
			expectIssues:   true,
		},
		{
			// A critical condition must not be downgraded to warning by a subsequent
			// lower-severity condition. This exercises the elevate() monotonicity.
			name: "Critical beats warning — CPU critical, memory warning",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 97, MemoryUsagePercent: 90},
				Connections: ConnectionMetrics{ActiveConnections: 30, MaxConnections: 100, UsagePercent: 30},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "critical",
			expectIssues:   true,
		},
		{
			// Connections unknown (MaxConnections == 0) — threshold must be skipped.
			name: "High connections skipped when MaxConnections unknown",
			metrics: &PrometheusHealthMetrics{
				Resources:   ResourceMetrics{CPUUsagePercent: 50, MemoryUsagePercent: 60},
				Connections: ConnectionMetrics{ActiveConnections: 97, MaxConnections: 0, UsagePercent: 97},
				Storage:     StorageMetrics{PageCacheHitRate: 90},
			},
			expectedStatus: "healthy",
			expectIssues:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.metrics.Issues = []string{}
			tt.metrics.Recommendations = []string{}

			status := svc.assessHealth(tt.metrics)

			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}
			if tt.expectIssues && len(tt.metrics.Issues) == 0 {
				t.Error("Expected issues to be reported, but none were found")
			}
			if !tt.expectIssues && len(tt.metrics.Issues) > 0 {
				t.Errorf("Expected no issues, but found: %v", tt.metrics.Issues)
			}
		})
	}
}
