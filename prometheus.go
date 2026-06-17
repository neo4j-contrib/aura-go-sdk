package aura

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/internal/api"
	"github.com/neo4j-contrib/aura-go-sdk/internal/utils"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// ============================================================================
// Types
// ============================================================================

// PrometheusHealthMetrics contains parsed health metrics for an instance.
type PrometheusHealthMetrics struct {
	InstanceID      string            `json:"instance_id"`
	Timestamp       time.Time         `json:"timestamp"`
	Resources       ResourceMetrics   `json:"resources"`
	Query           QueryMetrics      `json:"query"`
	Connections     ConnectionMetrics `json:"connections"`
	Storage         StorageMetrics    `json:"storage"`
	OverallStatus   string            `json:"overall_status"`
	Issues          []string          `json:"issues"`
	Recommendations []string          `json:"recommendations"`
}

// ResourceMetrics contains CPU and memory usage.
type ResourceMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
}

// QueryMetrics contains query performance statistics.
type QueryMetrics struct {
	// QueryExecutionTotal is a cumulative counter sourced from the
	// neo4j_db_query_execution_success_total Prometheus metric. It represents
	// the total number of successfully executed queries since the instance
	// started, not a per-second rate.
	QueryExecutionTotal float64 `json:"query_execution_total"`
	AvgLatencyMS        float64 `json:"avg_latency_ms"`
}

// ConnectionMetrics contains connection pool information.
type ConnectionMetrics struct {
	ActiveConnections int     `json:"active_connections"`
	MaxConnections    int     `json:"max_connections"`
	UsagePercent      float64 `json:"usage_percent"`
}

// StorageMetrics contains storage usage information.
type StorageMetrics struct {
	PageCacheHitRate float64 `json:"page_cache_hit_rate,omitempty"`
}

// PrometheusMetric represents a single parsed metric from Prometheus exposition format.
type PrometheusMetric struct {
	Name      string
	Labels    map[string]string
	Value     float64
	Timestamp int64
}

// PrometheusMetricsResponse contains parsed metrics from the raw endpoint.
type PrometheusMetricsResponse struct {
	Metrics map[string][]PrometheusMetric
}

// ============================================================================
// Service
// ============================================================================

// prometheusService handles Prometheus metrics operations.
type prometheusService struct {
	api     api.RequestService
	timeout time.Duration
	logger  *slog.Logger
}

// FetchRawMetrics fetches and parses raw Prometheus metrics from an Aura metrics endpoint.
func (p *prometheusService) FetchRawMetrics(ctx context.Context, prometheusURL string) (*PrometheusMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	p.logger.DebugContext(ctx, "fetching raw Prometheus metrics", slog.String("url", prometheusURL))
	return p.doFetchRawMetrics(ctx, prometheusURL)
}

// doFetchRawMetrics performs the HTTP fetch and metric parse. It assumes the
// caller has already applied an appropriate context deadline — no additional
// timeout is set here. This separation ensures that composed callers such as
// GetInstanceHealth apply the deadline exactly once rather than stacking two
// independent timeouts that would shorten the effective budget unexpectedly.
func (p *prometheusService) doFetchRawMetrics(ctx context.Context, prometheusURL string) (*PrometheusMetricsResponse, error) {
	if prometheusURL == "" {
		return nil, errors.New("prometheus URL cannot be empty")
	}

	resp, err := p.api.Get(ctx, prometheusURL)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to fetch raw metrics", slog.String("error", err.Error()))
		return nil, err
	}

	metrics, err := p.parsePrometheusMetrics(resp.Body)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to parse metrics", slog.String("error", err.Error()))
		return nil, err
	}

	p.logger.DebugContext(ctx, "raw metrics fetched successfully", slog.Int("metricCount", len(metrics.Metrics)))
	return metrics, nil
}

// parsePrometheusMetrics parses Prometheus metrics using the official client library.
func (p *prometheusService) parsePrometheusMetrics(data []byte) (*PrometheusMetricsResponse, error) {
	result := &PrometheusMetricsResponse{
		Metrics: make(map[string][]PrometheusMetric),
	}

	reader := bytes.NewReader(data)
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to parse Prometheus metrics: %w", err)
	}

	for name, mf := range metricFamilies {
		for _, m := range mf.Metric {
			metric := PrometheusMetric{
				Name:   name,
				Labels: make(map[string]string),
			}

			for _, label := range m.Label {
				if label.Name != nil && label.Value != nil {
					metric.Labels[*label.Name] = *label.Value
				}
			}

			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				if m.Counter != nil && m.Counter.Value != nil {
					metric.Value = *m.Counter.Value
				}
			case dto.MetricType_GAUGE:
				if m.Gauge != nil && m.Gauge.Value != nil {
					metric.Value = *m.Gauge.Value
				}
			case dto.MetricType_UNTYPED:
				if m.Untyped != nil && m.Untyped.Value != nil {
					metric.Value = *m.Untyped.Value
				}
			case dto.MetricType_SUMMARY:
				if m.Summary != nil && m.Summary.SampleSum != nil {
					metric.Value = *m.Summary.SampleSum
				}
			case dto.MetricType_HISTOGRAM:
				if m.Histogram != nil && m.Histogram.SampleSum != nil {
					metric.Value = *m.Histogram.SampleSum
				}
			}

			if m.TimestampMs != nil {
				metric.Timestamp = *m.TimestampMs
			}

			result.Metrics[name] = append(result.Metrics[name], metric)
		}
	}

	return result, nil
}

// GetInstanceHealth retrieves comprehensive health metrics for an instance.
func (p *prometheusService) GetInstanceHealth(ctx context.Context, instanceID string, prometheusURL string) (*PrometheusHealthMetrics, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	p.logger.DebugContext(ctx, "getting instance health metrics", slog.String("instanceID", instanceID))

	if err := utils.ValidateInstanceID(instanceID); err != nil {
		return nil, err
	}

	if prometheusURL == "" {
		return nil, errors.New("prometheus URL cannot be empty")
	}

	// doFetchRawMetrics is used directly here so the context deadline set above
	// is applied exactly once.
	rawMetrics, err := p.doFetchRawMetrics(ctx, prometheusURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}

	metrics := &PrometheusHealthMetrics{
		InstanceID:      instanceID,
		Timestamp:       time.Now(),
		Issues:          []string{},
		Recommendations: []string{},
	}

	if cpuUsage, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_aura_cpu_usage", nil); err == nil {
		if cpuLimit, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_aura_cpu_limit", nil); err == nil && cpuLimit > 0 {
			metrics.Resources.CPUUsagePercent = (cpuUsage / cpuLimit) * 100
		}
	} else {
		p.logger.WarnContext(ctx, "failed to get CPU usage", slog.String("error", err.Error()))
	}

	if heapRatio, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_dbms_vm_heap_used_ratio", nil); err == nil {
		metrics.Resources.MemoryUsagePercent = heapRatio * 100
	} else {
		p.logger.WarnContext(ctx, "failed to get memory usage", slog.String("error", err.Error()))
	}

	if successCount, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_db_query_execution_success_total", nil); err == nil {
		metrics.Query.QueryExecutionTotal = successCount
	} else {
		p.logger.WarnContext(ctx, "failed to get query count", slog.String("error", err.Error()))
	}

	if latency, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_db_query_execution_internal_latency_q50", nil); err == nil {
		metrics.Query.AvgLatencyMS = latency
	} else {
		p.logger.WarnContext(ctx, "failed to get query latency", slog.String("error", err.Error()))
	}

	if idle, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_dbms_bolt_connections_idle", nil); err == nil {
		if running, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_dbms_bolt_connections_running", nil); err == nil {
			metrics.Connections.ActiveConnections = int(idle + running)
		}
	}

	// Attempt to read the configured maximum from a Prometheus metric.
	// If the metric is not available (varies by Aura plan / Neo4j version)
	// MaxConnections stays at 0 and UsagePercent is left at 0 (unknown);
	// the connection threshold check in assessHealth is skipped in that case.
	if maxConns, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_dbms_bolt_connections_max_count", nil); err == nil && maxConns > 0 {
		metrics.Connections.MaxConnections = int(maxConns)
		metrics.Connections.UsagePercent = float64(metrics.Connections.ActiveConnections) / maxConns * 100
	}

	if hitRate, err := p.GetMetricValue(ctx, rawMetrics, "neo4j_dbms_page_cache_hit_ratio_per_minute", nil); err == nil {
		metrics.Storage.PageCacheHitRate = hitRate * 100
	} else {
		p.logger.WarnContext(ctx, "failed to get page cache hit rate", slog.String("error", err.Error()))
	}

	metrics.OverallStatus = p.assessHealth(metrics)

	p.logger.InfoContext(ctx, "instance health metrics retrieved",
		slog.String("instanceID", instanceID),
		slog.String("status", metrics.OverallStatus))

	return metrics, nil
}

// GetMetricValue retrieves a specific metric value by name and optional label filters.
// When no filters are provided it averages across all series for that metric name.
func (p *prometheusService) GetMetricValue(ctx context.Context, metrics *PrometheusMetricsResponse, name string, labelFilters map[string]string) (float64, error) {
	if metrics == nil {
		return 0, errors.New("metrics response must not be nil")
	}
	metricList, ok := metrics.Metrics[name]
	if !ok {
		p.logger.DebugContext(ctx, "metric not found", slog.String("metric", name))
		return 0, fmt.Errorf("metric %s not found", name)
	}

	if len(labelFilters) == 0 {
		if len(metricList) == 0 {
			p.logger.ErrorContext(ctx, "no values for metric", slog.String("metric", name))
			return 0, fmt.Errorf("no values for metric %s", name)
		}
		var sum float64
		for _, m := range metricList {
			sum += m.Value
		}
		return sum / float64(len(metricList)), nil
	}

	var matchingMetrics []PrometheusMetric
	for _, m := range metricList {
		match := true
		for key, value := range labelFilters {
			if m.Labels[key] != value {
				match = false
				break
			}
		}
		if match {
			matchingMetrics = append(matchingMetrics, m)
		}
	}

	if len(matchingMetrics) == 0 {
		p.logger.ErrorContext(ctx, "no matching metrics found", slog.String("metric", name), slog.Any("label filter", labelFilters))
		return 0, fmt.Errorf("no matching metrics found for %s with filters %v", name, labelFilters)
	}

	var sum float64
	for _, m := range matchingMetrics {
		sum += m.Value
	}
	return sum / float64(len(matchingMetrics)), nil
}

// assessHealth analyzes metrics and determines overall health status.
// Severity increases monotonically: healthy → warning → critical.
// A higher-severity condition always wins over a lower one.
func (p *prometheusService) assessHealth(metrics *PrometheusHealthMetrics) string {
	status := "healthy"

	// elevate raises status to the requested level only if it is a higher
	// severity than the current one. This ensures critical is never downgraded
	// to warning even when multiple conditions are evaluated.
	elevate := func(to string) {
		if to == "critical" || (to == "warning" && status == "healthy") {
			status = to
		}
	}

	switch {
	case metrics.Resources.CPUUsagePercent > 95:
		metrics.Issues = append(metrics.Issues, fmt.Sprintf("Critical CPU usage: %.1f%%", metrics.Resources.CPUUsagePercent))
		metrics.Recommendations = append(metrics.Recommendations, "Scale to a larger instance size immediately")
		elevate("critical")
	case metrics.Resources.CPUUsagePercent > 80:
		metrics.Issues = append(metrics.Issues, fmt.Sprintf("High CPU usage: %.1f%%", metrics.Resources.CPUUsagePercent))
		metrics.Recommendations = append(metrics.Recommendations, "Consider scaling to a larger instance size")
		elevate("warning")
	}

	switch {
	case metrics.Resources.MemoryUsagePercent > 95:
		metrics.Issues = append(metrics.Issues, fmt.Sprintf("Critical memory usage: %.1f%%", metrics.Resources.MemoryUsagePercent))
		metrics.Recommendations = append(metrics.Recommendations, "Scale to a larger memory instance immediately")
		elevate("critical")
	case metrics.Resources.MemoryUsagePercent > 85:
		metrics.Issues = append(metrics.Issues, fmt.Sprintf("High memory usage: %.1f%%", metrics.Resources.MemoryUsagePercent))
		metrics.Recommendations = append(metrics.Recommendations, "Consider scaling to a larger memory instance")
		elevate("warning")
	}

	if metrics.Connections.MaxConnections > 0 {
		switch {
		case metrics.Connections.UsagePercent > 95:
			metrics.Issues = append(metrics.Issues, fmt.Sprintf("Critical connection usage: %.1f%%", metrics.Connections.UsagePercent))
			metrics.Recommendations = append(metrics.Recommendations, "Reduce active connections immediately; review connection pooling")
			elevate("critical")
		case metrics.Connections.UsagePercent > 80:
			metrics.Issues = append(metrics.Issues, fmt.Sprintf("High connection usage: %.1f%%", metrics.Connections.UsagePercent))
			metrics.Recommendations = append(metrics.Recommendations, "Review connection pooling configuration in your application")
			elevate("warning")
		}
	}

	if metrics.Storage.PageCacheHitRate > 0 {
		switch {
		case metrics.Storage.PageCacheHitRate < 20:
			metrics.Issues = append(metrics.Issues, fmt.Sprintf("Critical page cache hit rate: %.1f%%", metrics.Storage.PageCacheHitRate))
			metrics.Recommendations = append(metrics.Recommendations, "Increase page cache size immediately; query performance is severely degraded")
			elevate("critical")
		case metrics.Storage.PageCacheHitRate < 50:
			metrics.Issues = append(metrics.Issues, fmt.Sprintf("Low page cache hit rate: %.1f%%", metrics.Storage.PageCacheHitRate))
			metrics.Recommendations = append(metrics.Recommendations, "Consider increasing page cache size for better performance")
			elevate("warning")
		}
	}

	return status
}
