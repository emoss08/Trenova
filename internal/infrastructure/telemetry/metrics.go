package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// Metric name prefixes following Prometheus naming conventions
	metricNamespace = "trenova"
	httpSubsystem   = "http"
	dbSubsystem     = "database"
	cacheSubsystem  = "cache"
	queueSubsystem  = "queue"

	// Common attribute keys
	attrMethod     = "method"
	attrPath       = "path"
	attrRoute      = "route"
	attrStatusCode = "status_code"
	attrOperation  = "operation"
	attrTable      = "table"
	attrJobType    = "job_type"
	attrError      = "error"
	attrHit        = "hit"
)

// MetricName constructs a metric name following Prometheus conventions
func MetricName(subsystem, name string) string {
	if subsystem == "" {
		return fmt.Sprintf("%s_%s", metricNamespace, name)
	}
	return fmt.Sprintf("%s_%s_%s", metricNamespace, subsystem, name)
}

// Metrics provides application-wide metrics collection
type Metrics struct {
	meter metric.Meter

	// HTTP metrics
	httpDuration     metric.Float64Histogram
	httpTotal        metric.Int64Counter
	httpErrors       metric.Int64Counter
	httpActive       metric.Int64UpDownCounter
	httpRequestSize  metric.Int64Histogram
	httpResponseSize metric.Int64Histogram

	// Database metrics
	dbOperationDuration metric.Float64Histogram
	dbOperationTotal    metric.Int64Counter
	dbOperationErrors   metric.Int64Counter
	dbConnectionsTotal  metric.Int64Counter
	dbConnectionsActive metric.Int64UpDownCounter
	dbTransactionTotal  metric.Int64Counter

	// Cache metrics
	cacheOperationDuration metric.Float64Histogram
	cacheOperationTotal    metric.Int64Counter
	cacheHitTotal          metric.Int64Counter
	cacheMissTotal         metric.Int64Counter
	cacheEvictionTotal     metric.Int64Counter

	// Queue metrics
	queueJobDuration metric.Float64Histogram
	queueJobTotal    metric.Int64Counter
	queueJobErrors   metric.Int64Counter
	queueJobRetries  metric.Int64Counter

	// Runtime metrics (registered as callbacks)
	runtimeCallbacks []metric.Registration

	// CPU tracking
	lastCPUTime  time.Time
	lastCPUUsage float64

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

var ErrNoMeterProvider = eris.New("meter provider is required")

// NewMetrics creates a new metrics instance with all metric definitions
func NewMetrics(provider metric.MeterProvider) (*Metrics, error) {
	if provider == nil {
		return nil, ErrNoMeterProvider
	}

	meter := provider.Meter(
		metricNamespace,
		metric.WithInstrumentationVersion("1.0.0"),
	)

	m := &Metrics{
		meter:            meter,
		runtimeCallbacks: make([]metric.Registration, 0),
	}

	if err := m.initHTTPMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP metrics: %w", err)
	}

	if err := m.initDatabaseMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize database metrics: %w", err)
	}

	if err := m.initCacheMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache metrics: %w", err)
	}

	if err := m.initQueueMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize queue metrics: %w", err)
	}

	if err := m.initRuntimeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime metrics: %w", err)
	}

	// Register Prometheus collectors for compatibility
	registerPrometheusCollectors()

	return m, nil
}

// initHTTPMetrics initializes HTTP-related metrics
func (m *Metrics) initHTTPMetrics() error {
	var err error

	m.httpDuration, err = m.meter.Float64Histogram(
		MetricName(httpSubsystem, "request_duration_seconds"),
		metric.WithDescription("HTTP request latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005,
			0.01,
			0.025,
			0.05,
			0.1,
			0.25,
			0.5,
			1,
			2.5,
			5,
			10,
		),
	)
	if err != nil {
		return err
	}

	m.httpTotal, err = m.meter.Int64Counter(
		MetricName(httpSubsystem, "requests_total"),
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return err
	}

	m.httpErrors, err = m.meter.Int64Counter(
		MetricName(httpSubsystem, "errors_total"),
		metric.WithDescription("Total number of HTTP errors"),
	)
	if err != nil {
		return err
	}

	m.httpActive, err = m.meter.Int64UpDownCounter(
		MetricName(httpSubsystem, "requests_active"),
		metric.WithDescription("Number of HTTP requests currently being processed"),
	)
	if err != nil {
		return err
	}

	m.httpRequestSize, err = m.meter.Int64Histogram(
		MetricName(httpSubsystem, "request_size_bytes"),
		metric.WithDescription("HTTP request body size in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 1000, 10000, 100000, 1000000, 10000000),
	)
	if err != nil {
		return err
	}

	m.httpResponseSize, err = m.meter.Int64Histogram(
		MetricName(httpSubsystem, "response_size_bytes"),
		metric.WithDescription("HTTP response body size in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 1000, 10000, 100000, 1000000, 10000000),
	)
	if err != nil {
		return err
	}

	return nil
}

// initDatabaseMetrics initializes database-related metrics
func (m *Metrics) initDatabaseMetrics() error {
	var err error

	m.dbOperationDuration, err = m.meter.Float64Histogram(
		MetricName(dbSubsystem, "operation_duration_seconds"),
		metric.WithDescription("Database operation duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.001,
			0.005,
			0.01,
			0.025,
			0.05,
			0.1,
			0.25,
			0.5,
			1,
			2.5,
			5,
		),
	)
	if err != nil {
		return err
	}

	m.dbOperationTotal, err = m.meter.Int64Counter(
		MetricName(dbSubsystem, "operations_total"),
		metric.WithDescription("Total number of database operations"),
	)
	if err != nil {
		return err
	}

	m.dbOperationErrors, err = m.meter.Int64Counter(
		MetricName(dbSubsystem, "errors_total"),
		metric.WithDescription("Total number of database errors"),
	)
	if err != nil {
		return err
	}

	m.dbConnectionsTotal, err = m.meter.Int64Counter(
		MetricName(dbSubsystem, "connections_total"),
		metric.WithDescription("Total number of database connection attempts"),
	)
	if err != nil {
		return err
	}

	m.dbConnectionsActive, err = m.meter.Int64UpDownCounter(
		MetricName(dbSubsystem, "connections_active"),
		metric.WithDescription("Number of active database connections"),
	)
	if err != nil {
		return err
	}

	m.dbTransactionTotal, err = m.meter.Int64Counter(
		MetricName(dbSubsystem, "transactions_total"),
		metric.WithDescription("Total number of database transactions"),
	)
	if err != nil {
		return err
	}

	return nil
}

// initCacheMetrics initializes cache-related metrics
func (m *Metrics) initCacheMetrics() error {
	var err error

	m.cacheOperationDuration, err = m.meter.Float64Histogram(
		MetricName(cacheSubsystem, "operation_duration_seconds"),
		metric.WithDescription("Cache operation duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1),
	)
	if err != nil {
		return err
	}

	m.cacheOperationTotal, err = m.meter.Int64Counter(
		MetricName(cacheSubsystem, "operations_total"),
		metric.WithDescription("Total number of cache operations"),
	)
	if err != nil {
		return err
	}

	m.cacheHitTotal, err = m.meter.Int64Counter(
		MetricName(cacheSubsystem, "hits_total"),
		metric.WithDescription("Total number of cache hits"),
	)
	if err != nil {
		return err
	}

	m.cacheMissTotal, err = m.meter.Int64Counter(
		MetricName(cacheSubsystem, "misses_total"),
		metric.WithDescription("Total number of cache misses"),
	)
	if err != nil {
		return err
	}

	m.cacheEvictionTotal, err = m.meter.Int64Counter(
		MetricName(cacheSubsystem, "evictions_total"),
		metric.WithDescription("Total number of cache evictions"),
	)
	if err != nil {
		return err
	}

	return nil
}

// initQueueMetrics initializes queue-related metrics
func (m *Metrics) initQueueMetrics() error {
	var err error

	m.queueJobDuration, err = m.meter.Float64Histogram(
		MetricName(queueSubsystem, "job_duration_seconds"),
		metric.WithDescription("Queue job processing duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600),
	)
	if err != nil {
		return err
	}

	m.queueJobTotal, err = m.meter.Int64Counter(
		MetricName(queueSubsystem, "jobs_total"),
		metric.WithDescription("Total number of queue jobs processed"),
	)
	if err != nil {
		return err
	}

	m.queueJobErrors, err = m.meter.Int64Counter(
		MetricName(queueSubsystem, "errors_total"),
		metric.WithDescription("Total number of queue job errors"),
	)
	if err != nil {
		return err
	}

	m.queueJobRetries, err = m.meter.Int64Counter(
		MetricName(queueSubsystem, "retries_total"),
		metric.WithDescription("Total number of queue job retries"),
	)
	if err != nil {
		return err
	}

	return nil
}

// initRuntimeMetrics initializes runtime observability metrics
func (m *Metrics) initRuntimeMetrics() error {
	// Memory metrics
	_, err := m.meter.Float64ObservableGauge(
		MetricName("runtime", "memory_alloc_bytes"),
		metric.WithDescription("Bytes of allocated heap objects"),
		metric.WithUnit("By"),
		metric.WithFloat64Callback(m.observeMemoryStats),
	)
	if err != nil {
		return err
	}

	// Goroutine metrics
	_, err = m.meter.Int64ObservableGauge(
		MetricName("runtime", "goroutines"),
		metric.WithDescription("Number of goroutines"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	// GC metrics
	_, err = m.meter.Float64ObservableCounter(
		MetricName("runtime", "gc_pause_seconds_total"),
		metric.WithDescription("Total time spent in GC pause"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(m.observeGCStats),
	)
	if err != nil {
		return err
	}

	// CPU metrics
	_, err = m.meter.Float64ObservableGauge(
		MetricName("runtime", "cpu_percent"),
		metric.WithDescription("CPU usage percentage"),
		metric.WithUnit("%"),
		metric.WithFloat64Callback(m.observeCPUStats),
	)
	if err != nil {
		return err
	}

	return nil
}

// observeMemoryStats is a callback for memory statistics
func (m *Metrics) observeMemoryStats(_ context.Context, o metric.Float64Observer) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	o.Observe(float64(memStats.Alloc), metric.WithAttributes(
		attribute.String("type", "alloc"),
	))
	o.Observe(float64(memStats.TotalAlloc), metric.WithAttributes(
		attribute.String("type", "total_alloc"),
	))
	o.Observe(float64(memStats.Sys), metric.WithAttributes(
		attribute.String("type", "sys"),
	))
	o.Observe(float64(memStats.HeapAlloc), metric.WithAttributes(
		attribute.String("type", "heap_alloc"),
	))
	o.Observe(float64(memStats.HeapInuse), metric.WithAttributes(
		attribute.String("type", "heap_inuse"),
	))

	return nil
}

// observeGCStats is a callback for GC statistics
func (m *Metrics) observeGCStats(_ context.Context, o metric.Float64Observer) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	o.Observe(float64(memStats.PauseTotalNs) / 1e9)
	return nil
}

// observeCPUStats is a callback for CPU statistics
func (m *Metrics) observeCPUStats(_ context.Context, o metric.Float64Observer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get current process CPU time using syscall
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return fmt.Errorf("failed to get rusage: %w", err)
	}

	// Convert CPU time to seconds
	currentCPUTime := float64(rusage.Utime.Sec) + float64(rusage.Utime.Usec)/1e6 +
		float64(rusage.Stime.Sec) + float64(rusage.Stime.Usec)/1e6

	// Calculate CPU percentage based on time elapsed
	now := time.Now()
	if !m.lastCPUTime.IsZero() {
		// Calculate elapsed time in seconds
		elapsed := now.Sub(m.lastCPUTime).Seconds()

		// Calculate CPU usage delta
		cpuDelta := currentCPUTime - m.lastCPUUsage

		// Calculate CPU percentage (0-100)
		// cpuDelta is in seconds, elapsed is in seconds
		// Multiply by 100 to get percentage
		cpuPercent := (cpuDelta / elapsed) * 100

		// Clamp between 0 and 100 (can exceed 100 on multi-core systems)
		if cpuPercent < 0 {
			cpuPercent = 0
		}

		o.Observe(cpuPercent)
	} else {
		// First call, no previous data
		o.Observe(0)
	}

	// Update last values
	m.lastCPUTime = now
	m.lastCPUUsage = currentCPUTime

	return nil
}

// RecordHTTPRequest records metrics for an HTTP request
func (m *Metrics) RecordHTTPRequest(
	ctx context.Context,
	method, route string,
	statusCode int,
	duration time.Duration,
	requestSize, responseSize int64,
) {
	attrs := []attribute.KeyValue{
		attribute.String(attrMethod, method),
		attribute.String(attrRoute, route),
		attribute.Int(attrStatusCode, statusCode),
	}

	// Record request metrics
	m.httpTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.httpDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	// Record size metrics
	if requestSize > 0 {
		m.httpRequestSize.Record(ctx, requestSize, metric.WithAttributes(attrs...))
	}
	if responseSize > 0 {
		m.httpResponseSize.Record(ctx, responseSize, metric.WithAttributes(attrs...))
	}

	// Record errors for 4xx and 5xx status codes
	if statusCode >= 400 {
		errorAttrs := []attribute.KeyValue{
			attribute.String(attrMethod, method),
			attribute.String(attrRoute, route),
			attribute.Int(attrStatusCode, statusCode),
			attribute.String(attrError, httpStatusClass(statusCode)),
		}
		m.httpErrors.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

// RecordActiveHTTPRequest increments/decrements the active request counter
func (m *Metrics) RecordActiveHTTPRequest(ctx context.Context, delta int64) {
	m.httpActive.Add(ctx, delta)
}

// RecordDatabaseOperation records metrics for a database operation
func (m *Metrics) RecordDatabaseOperation(
	ctx context.Context,
	operation, table string,
	duration time.Duration,
	err error,
) {
	attrs := []attribute.KeyValue{
		attribute.String(attrOperation, operation),
		attribute.String(attrTable, table),
	}

	m.dbOperationTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.dbOperationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		errorAttrs := []attribute.KeyValue{
			attribute.String(attrOperation, operation),
			attribute.String(attrTable, table),
			attribute.Bool(attrError, true),
		}
		m.dbOperationErrors.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

// RecordDatabaseConnection records database connection metrics
func (m *Metrics) RecordDatabaseConnection(
	ctx context.Context,
	connectionType string,
	success bool,
) {
	attrs := []attribute.KeyValue{
		attribute.String("connection_type", connectionType),
		attribute.String("status", boolToStatus(success)),
	}

	m.dbConnectionsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordActiveDatabaseConnections updates the active connection counter
func (m *Metrics) RecordActiveDatabaseConnections(ctx context.Context, delta int64) {
	m.dbConnectionsActive.Add(ctx, delta)
}

// RecordCacheOperation records metrics for a cache operation
func (m *Metrics) RecordCacheOperation(
	ctx context.Context,
	operation string,
	hit bool,
	duration time.Duration,
) {
	attrs := []attribute.KeyValue{
		attribute.String(attrOperation, operation),
		attribute.Bool(attrHit, hit),
	}

	m.cacheOperationTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.cacheOperationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if hit {
		m.cacheHitTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.cacheMissTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordCacheEviction records a cache eviction
func (m *Metrics) RecordCacheEviction(ctx context.Context, reason string) {
	attrs := []attribute.KeyValue{
		attribute.String("reason", reason),
	}

	m.cacheEvictionTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordQueueJob records metrics for a queue job
func (m *Metrics) RecordQueueJob(
	ctx context.Context,
	jobType string,
	duration time.Duration,
	err error,
	retryCount int,
) {
	attrs := []attribute.KeyValue{
		attribute.String(attrJobType, jobType),
		attribute.Int("retry_count", retryCount),
	}

	m.queueJobTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.queueJobDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		errorAttrs := []attribute.KeyValue{
			attribute.String(attrJobType, jobType),
			attribute.Int("retry_count", retryCount),
			attribute.Bool(attrError, true),
		}
		m.queueJobErrors.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}

	if retryCount > 0 {
		m.queueJobRetries.Add(ctx, int64(retryCount), metric.WithAttributes(attrs...))
	}
}

// Helper functions

// httpStatusClass returns the status class (e.g., "4xx", "5xx")
func httpStatusClass(statusCode int) string {
	return fmt.Sprintf("%dxx", statusCode/100)
}

// boolToStatus converts a boolean to a status string
func boolToStatus(b bool) string {
	if b {
		return "success"
	}
	return "failure"
}

// registerPrometheusCollectors registers standard Prometheus collectors
func registerPrometheusCollectors() {
	// Register only once
	registrationOnce.Do(func() {
		// Use Register instead of MustRegister to handle errors gracefully
		buildInfoCollector := collectors.NewBuildInfoCollector()
		goCollector := collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		)
		processCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			Namespace: metricNamespace,
		})

		// Try to register each collector, ignoring AlreadyRegisteredError
		_ = prometheus.Register(buildInfoCollector)
		_ = prometheus.Register(goCollector)
		_ = prometheus.Register(processCollector)
	})
}

var registrationOnce sync.Once

// Convenience methods for common use cases

// RecordHTTPRequestSimple is a simplified version of RecordHTTPRequest
func (m *Metrics) RecordHTTPRequestSimple(
	method, route string,
	statusCode int,
	duration time.Duration,
) {
	m.RecordHTTPRequest(context.Background(), method, route, statusCode, duration, 0, 0)
}

// RecordDatabaseOperationSimple is a simplified version of RecordDatabaseOperation
func (m *Metrics) RecordDatabaseOperationSimple(
	operation, table string,
	duration time.Duration,
	err error,
) {
	m.RecordDatabaseOperation(context.Background(), operation, table, duration, err)
}

// RecordCacheOperationSimple is a simplified version of RecordCacheOperation
func (m *Metrics) RecordCacheOperationSimple(operation string, hit bool, duration time.Duration) {
	m.RecordCacheOperation(context.Background(), operation, hit, duration)
}

// RecordQueueJobSimple is a simplified version of RecordQueueJob
func (m *Metrics) RecordQueueJobSimple(jobType string, duration time.Duration, err error) {
	m.RecordQueueJob(context.Background(), jobType, duration, err, 0)
}
