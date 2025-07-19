package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const dbSpanNameKey = contextKey("db.span_name")

// DatabaseHookConfig provides configuration for database hooks
type DatabaseHookConfig struct {
	ServiceName     string
	Metrics         *Metrics
	RecordStatement bool
	MaxStatementLen int
	SkipOperations  []string
}

// DefaultDatabaseHookConfig returns default configuration
func DefaultDatabaseHookConfig(serviceName string, metrics *Metrics) DatabaseHookConfig {
	return DatabaseHookConfig{
		ServiceName:     serviceName,
		Metrics:         metrics,
		RecordStatement: true,
		MaxStatementLen: 5000,
		SkipOperations:  []string{"ping"},
	}
}

// TracingHook implements Bun query hook for tracing
type TracingHook struct {
	tracer  trace.Tracer
	metrics *Metrics
	config  DatabaseHookConfig
	skipMap map[string]bool
}

// NewTracingHook creates a new tracing hook with configuration
func NewTracingHook(config DatabaseHookConfig) *TracingHook {
	skipMap := make(map[string]bool, len(config.SkipOperations))
	for _, op := range config.SkipOperations {
		skipMap[strings.ToLower(op)] = true
	}

	return &TracingHook{
		tracer:  otel.Tracer(fmt.Sprintf("%s.database", config.ServiceName)),
		metrics: config.Metrics,
		config:  config,
		skipMap: skipMap,
	}
}

// BeforeQuery starts a new span for the database query
func (h *TracingHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	operation := h.operationName(event)
	if h.skipMap[operation] {
		return ctx
	}

	tableName := h.extractTableName(event)
	spanName := h.buildSpanName(operation, tableName)

	attrs := []attribute.KeyValue{
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", operation),
	}

	if tableName != "" && tableName != "unknown" {
		attrs = append(attrs, attribute.String("db.sql.table", tableName))
	}

	if h.config.RecordStatement {
		stmt := h.sanitizeStatement(event.Query)
		if len(stmt) > h.config.MaxStatementLen {
			stmt = stmt[:h.config.MaxStatementLen] + "..."
		}
		attrs = append(attrs, attribute.String("db.statement", stmt))
	}

	// Add connection info if available from event
	// Note: Bun doesn't expose connection info directly in events

	ctx, span := h.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	return context.WithValue(ctx, dbSpanNameKey, spanName)
}

// AfterQuery completes the span and records metrics
func (h *TracingHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	operation := h.operationName(event)
	if h.skipMap[operation] {
		return
	}

	var duration time.Duration
	if startTime, ok := ctx.Value("db.start_time").(time.Time); ok {
		duration = time.Since(startTime)
	} else {
		duration = time.Since(event.StartTime)
	}

	if event.Result != nil {
		rowsAffected, _ := event.Result.RowsAffected()
		if rowsAffected > 0 {
			span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
		}
	}

	if event.Err != nil {
		if eris.Is(event.Err, sql.ErrNoRows) {
			span.SetAttributes(attribute.Bool("db.no_rows", true))
		} else {
			span.RecordError(event.Err)
			span.SetStatus(codes.Error, event.Err.Error())

			errorType := h.categorizeError(event.Err)
			span.SetAttributes(attribute.String("db.error_type", errorType))
		}
	} else {
		span.SetStatus(codes.Ok, "Query executed successfully")
	}

	span.End()

	if h.metrics != nil {
		tableName := h.extractTableName(event)
		h.metrics.RecordDatabaseOperation(ctx, operation, tableName, duration, event.Err)
	}
}

// operationName extracts the operation type from the query event
func (h *TracingHook) operationName(event *bun.QueryEvent) string {
	switch event.QueryAppender.(type) {
	case *bun.SelectQuery:
		return "select"
	case *bun.InsertQuery:
		return "insert"
	case *bun.UpdateQuery:
		return "update"
	case *bun.DeleteQuery:
		return "delete"
	case *bun.CreateTableQuery:
		return "create_table"
	case *bun.DropTableQuery:
		return "drop_table"
	case *bun.TruncateTableQuery:
		return "truncate"
	case *bun.CreateIndexQuery:
		return "create_index"
	case *bun.DropIndexQuery:
		return "drop_index"
	default:
		return h.extractOperationFromQuery(event.Query)
	}
}

// extractOperationFromQuery attempts to extract operation from raw SQL
func (h *TracingHook) extractOperationFromQuery(query string) string {
	query = strings.TrimSpace(strings.ToLower(query))
	switch {
	case strings.HasPrefix(query, "select"):
		return "select"
	case strings.HasPrefix(query, "insert"):
		return "insert"
	case strings.HasPrefix(query, "update"):
		return "update"
	case strings.HasPrefix(query, "delete"):
		return "delete"
	case strings.HasPrefix(query, "begin"):
		return "begin"
	case strings.HasPrefix(query, "commit"):
		return "commit"
	case strings.HasPrefix(query, "rollback"):
		return "rollback"
	default:
		return "query"
	}
}

// extractTableName extracts the table name from the query event
func (h *TracingHook) extractTableName(event *bun.QueryEvent) string {
	if event.Model != nil {
		if tableName, ok := event.Model.(interface{ Table() string }); ok {
			return tableName.Table()
		}
	}

	tableName := h.extractTableFromQuery(event.Query)
	if tableName != "" {
		return tableName
	}

	return "unknown"
}

func (h *TracingHook) extractTableFromQuery(query string) string {
	// (Todo: Wolfred): we need a proper SQL parser, but for now this works
	query = strings.ToLower(strings.TrimSpace(query))

	patterns := []struct {
		prefix string
		suffix string
	}{
		{"from ", " "},
		{"into ", " "},
		{"update ", " "},
		{"delete from ", " "},
	}

	for _, p := range patterns {
		idx := strings.Index(query, p.prefix)

		if idx == -1 {
			continue
		}

		start := idx + len(p.prefix)
		end := strings.Index(query[start:], p.suffix)
		if end == -1 {
			end = len(query) - start
		}

		tableName := strings.Trim(query[start:start+end], "\"'`")
		if parts := strings.Split(tableName, "."); len(parts) > 1 {
			tableName = parts[len(parts)-1]
		}
		return tableName
	}

	return ""
}

// buildSpanName creates a meaningful span name
func (h *TracingHook) buildSpanName(operation, tableName string) string {
	if tableName != "" && tableName != "unknown" {
		return fmt.Sprintf("db.%s.%s", operation, tableName)
	}
	return fmt.Sprintf("db.%s", operation)
}

// sanitizeStatement removes sensitive data from SQL statements
func (h *TracingHook) sanitizeStatement(query string) string {
	// ! TODO(wolfred): we need to write this
	return query
}

// categorizeError categorizes database errors for better tracking
func (h *TracingHook) categorizeError(err error) string {
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "connection"):
		return "connection_error"
	case strings.Contains(errStr, "timeout"):
		return "timeout_error"
	case strings.Contains(errStr, "constraint"):
		return "constraint_violation"
	case strings.Contains(errStr, "duplicate"):
		return "duplicate_key"
	case strings.Contains(errStr, "syntax"):
		return "syntax_error"
	case strings.Contains(errStr, "permission") || strings.Contains(errStr, "denied"):
		return "permission_error"
	default:
		return "unknown_error"
	}
}

// CombinedDatabaseHook combines tracing and metrics in a single hook for efficiency
type CombinedDatabaseHook struct {
	tracingHook *TracingHook
	config      DatabaseHookConfig
}

// NewCombinedDatabaseHook creates a hook that handles both tracing and metrics
func NewCombinedDatabaseHook(config DatabaseHookConfig) *CombinedDatabaseHook {
	return &CombinedDatabaseHook{
		tracingHook: NewTracingHook(config),
		config:      config,
	}
}

// BeforeQuery delegates to tracing hook
func (h *CombinedDatabaseHook) BeforeQuery(
	ctx context.Context,
	event *bun.QueryEvent,
) context.Context {
	return h.tracingHook.BeforeQuery(ctx, event)
}

// AfterQuery delegates to tracing hook (which also records metrics)
func (h *CombinedDatabaseHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	h.tracingHook.AfterQuery(ctx, event)
}

// DatabaseConnectionMonitor monitors database connection pool metrics
type DatabaseConnectionMonitor struct {
	db           *bun.DB
	metrics      *Metrics
	config       DatabaseMonitorConfig
	callbacks    []metric.Registration
	mu           sync.RWMutex
	lastStats    sql.DBStats
	statsUpdated time.Time
}

// DatabaseMonitorConfig provides configuration for database monitoring
type DatabaseMonitorConfig struct {
	ServiceName    string
	UpdateInterval time.Duration
}

// DefaultDatabaseMonitorConfig returns default configuration
func DefaultDatabaseMonitorConfig(serviceName string) DatabaseMonitorConfig {
	return DatabaseMonitorConfig{
		ServiceName:    serviceName,
		UpdateInterval: 10 * time.Second,
	}
}

// NewDatabaseConnectionMonitor creates a new connection monitor
func NewDatabaseConnectionMonitor(
	db *bun.DB,
	metrics *Metrics,
	config DatabaseMonitorConfig,
) (*DatabaseConnectionMonitor, error) {
	monitor := &DatabaseConnectionMonitor{
		db:        db,
		metrics:   metrics,
		config:    config,
		callbacks: make([]metric.Registration, 0),
	}

	if err := monitor.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize database metrics: %w", err)
	}

	return monitor, nil
}

// initMetrics initializes observable metrics for database connections
func (m *DatabaseConnectionMonitor) initMetrics() error {
	meter := otel.Meter(fmt.Sprintf("%s.database", m.config.ServiceName))

	_, err := meter.Int64ObservableGauge(
		MetricName(dbSubsystem, "connections_open"),
		metric.WithDescription("Number of open database connections"),
		metric.WithInt64Callback(m.observeOpenConnections),
	)
	if err != nil {
		return err
	}

	_, err = meter.Int64ObservableGauge(
		MetricName(dbSubsystem, "connections_in_use"),
		metric.WithDescription("Number of database connections currently in use"),
		metric.WithInt64Callback(m.observeInUseConnections),
	)
	if err != nil {
		return err
	}

	// Idle connections
	_, err = meter.Int64ObservableGauge(
		MetricName(dbSubsystem, "connections_idle"),
		metric.WithDescription("Number of idle database connections"),
		metric.WithInt64Callback(m.observeIdleConnections),
	)
	if err != nil {
		return err
	}

	// Wait count
	_, err = meter.Int64ObservableCounter(
		MetricName(dbSubsystem, "connections_wait_total"),
		metric.WithDescription("Total number of times waited for a connection"),
		metric.WithInt64Callback(m.observeWaitCount),
	)
	if err != nil {
		return err
	}

	// Wait duration
	_, err = meter.Float64ObservableGauge(
		MetricName(dbSubsystem, "connections_wait_duration_seconds"),
		metric.WithDescription("Total time spent waiting for connections"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(m.observeWaitDuration),
	)
	if err != nil {
		return err
	}

	// Max open connections
	_, err = meter.Int64ObservableGauge(
		MetricName(dbSubsystem, "connections_max_open"),
		metric.WithDescription("Maximum number of open connections allowed"),
		metric.WithInt64Callback(m.observeMaxOpenConnections),
	)
	if err != nil {
		return err
	}

	return nil
}

// updateStats updates cached stats periodically
func (m *DatabaseConnectionMonitor) updateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastStats = m.db.Stats()
	m.statsUpdated = time.Now()
}

// getStats returns cached stats if fresh, otherwise updates
func (m *DatabaseConnectionMonitor) getStats() sql.DBStats {
	m.mu.RLock()
	if time.Since(m.statsUpdated) < m.config.UpdateInterval {
		defer m.mu.RUnlock()
		return m.lastStats
	}
	m.mu.RUnlock()

	m.updateStats()

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastStats
}

// Callback functions for metrics
func (m *DatabaseConnectionMonitor) observeOpenConnections(
	_ context.Context,
	o metric.Int64Observer,
) error {
	stats := m.getStats()
	o.Observe(int64(stats.OpenConnections))
	return nil
}

func (m *DatabaseConnectionMonitor) observeInUseConnections(
	_ context.Context,
	o metric.Int64Observer,
) error {
	stats := m.getStats()
	o.Observe(int64(stats.InUse))
	return nil
}

func (m *DatabaseConnectionMonitor) observeIdleConnections(
	_ context.Context,
	o metric.Int64Observer,
) error {
	stats := m.getStats()
	o.Observe(int64(stats.Idle))
	return nil
}

func (m *DatabaseConnectionMonitor) observeWaitCount(
	_ context.Context,
	o metric.Int64Observer,
) error {
	stats := m.getStats()
	o.Observe(stats.WaitCount)
	return nil
}

func (m *DatabaseConnectionMonitor) observeWaitDuration(
	_ context.Context,
	o metric.Float64Observer,
) error {
	stats := m.getStats()
	o.Observe(stats.WaitDuration.Seconds())
	return nil
}

func (m *DatabaseConnectionMonitor) observeMaxOpenConnections(
	_ context.Context,
	o metric.Int64Observer,
) error {
	stats := m.getStats()
	o.Observe(int64(stats.MaxOpenConnections))
	return nil
}

// DatabaseInstrumentation holds all database instrumentation components
type DatabaseInstrumentation struct {
	hook    *CombinedDatabaseHook
	monitor *DatabaseConnectionMonitor
}

// InstrumentDatabase adds comprehensive instrumentation to a Bun database
func InstrumentDatabase(
	db *bun.DB,
	serviceName string,
	metrics *Metrics,
) (*DatabaseInstrumentation, error) {
	// Create hook configuration
	hookConfig := DefaultDatabaseHookConfig(serviceName, metrics)

	// Add combined hook for tracing and metrics
	hook := NewCombinedDatabaseHook(hookConfig)
	db.AddQueryHook(hook)

	// Create connection monitor
	monitorConfig := DefaultDatabaseMonitorConfig(serviceName)
	monitor, err := NewDatabaseConnectionMonitor(db, metrics, monitorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection monitor: %w", err)
	}

	return &DatabaseInstrumentation{
		hook:    hook,
		monitor: monitor,
	}, nil
}

// TransactionOptions provides options for database transactions
type TransactionOptions struct {
	ReadOnly bool
	Timeout  time.Duration
}

// RunInTransaction executes a function within a database transaction with tracing
func RunInTransaction(
	ctx context.Context,
	db *bun.DB,
	opts *TransactionOptions,
	fn func(*bun.Tx) error,
) error {
	tracer := otel.Tracer("database")

	// Create transaction options
	txOpts := &sql.TxOptions{}
	if opts != nil {
		txOpts.ReadOnly = opts.ReadOnly
	}

	// Apply timeout if specified
	if opts != nil && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Start transaction span
	ctx, span := tracer.Start(ctx, "db.transaction",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.Bool("db.transaction.read_only", txOpts.ReadOnly),
		),
	)
	defer span.End()

	// Begin transaction
	tx, err := db.BeginTx(ctx, txOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to begin transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute function
	err = fn(&tx)
	if err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			span.RecordError(rbErr)
			span.AddEvent("rollback_failed", trace.WithAttributes(
				attribute.String("error", rbErr.Error()),
			))
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "Transaction failed")
		return err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	span.SetStatus(codes.Ok, "Transaction completed successfully")
	return nil
}

// WrapDatabaseOperation wraps any database operation with a span
func WrapDatabaseOperation(
	ctx context.Context,
	operationName string,
	attrs []attribute.KeyValue,
	fn func(context.Context) error,
) error {
	tracer := otel.Tracer("database")

	// Build attributes
	baseAttrs := []attribute.KeyValue{
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", operationName),
	}
	baseAttrs = append(baseAttrs, attrs...)

	ctx, span := tracer.Start(ctx, fmt.Sprintf("db.%s", operationName),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(baseAttrs...),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "Operation completed successfully")
	}

	return err
}
