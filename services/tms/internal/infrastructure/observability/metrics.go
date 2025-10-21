package observability

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type MetricsRegistry struct {
	registry *prometheus.Registry
	cfg      *config.Config
	logger   *zap.Logger

	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpActiveRequests  prometheus.Gauge

	shipmentsProcessed  *prometheus.CounterVec
	shipmentDuration    *prometheus.HistogramVec
	documentsUploaded   *prometheus.CounterVec
	documentProcessTime *prometheus.HistogramVec
	complianceChecks    *prometheus.CounterVec

	dbQueriesTotal      *prometheus.CounterVec
	dbQueryDuration     *prometheus.HistogramVec
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge

	cacheHits      *prometheus.CounterVec
	cacheMisses    *prometheus.CounterVec
	cacheEvictions prometheus.Counter

	errorsTotal     *prometheus.CounterVec
	panicRecoveries prometheus.Counter

	activeUsers prometheus.Gauge
	revenue     *prometheus.CounterVec
	apiKeyUsage *prometheus.CounterVec

	// WebSocket metrics
	wsConnectionsActive  prometheus.Gauge
	wsConnectionsTotal   *prometheus.CounterVec
	wsMessagesSent       *prometheus.CounterVec
	wsMessagesReceived   *prometheus.CounterVec
	wsMessageSize        *prometheus.HistogramVec
	wsBroadcastsSent     *prometheus.CounterVec
	wsConnectionDuration *prometheus.HistogramVec
	wsRoomsActive        prometheus.Gauge
	wsUsersPerRoom       *prometheus.HistogramVec
	wsConnectionErrors   *prometheus.CounterVec
	wsPingPongLatency    *prometheus.HistogramVec

	// Streaming metrics
	streamingConnectionsActive prometheus.Gauge
	streamingConnectionsTotal  *prometheus.CounterVec
	streamingMessagesSent      *prometheus.CounterVec
	streamingBroadcastsSent    *prometheus.CounterVec

	// CDC metrics
	cdcMessagesTotal       *prometheus.CounterVec
	cdcMessageDuration     *prometheus.HistogramVec
	cdcHandlerErrors       *prometheus.CounterVec
	cdcSchemaCache         *prometheus.CounterVec
	cdcConsumerLag         prometheus.Gauge
	cdcBatchSize           *prometheus.HistogramVec
	cdcRebalances          prometheus.Counter
	cdcProcessingWorkers   prometheus.Gauge
}

func NewMetricsRegistry(cfg *config.Config, logger *zap.Logger) (*MetricsRegistry, error) {
	if !cfg.Monitoring.Metrics.Enabled {
		logger.Warn("🟡 Metrics collection is disabled")
		// Return a no-op metrics registry instead of an error
		return &MetricsRegistry{
			registry: nil,
			cfg:      cfg,
			logger:   logger,
		}, nil
	}

	if cfg.IsDevelopment() && !cfg.Monitoring.Metrics.Enabled {
		logger.Debug("🟡 Metrics disabled in development environment")
		// Return a no-op metrics registry instead of an error
		return &MetricsRegistry{
			registry: nil,
			cfg:      cfg,
			logger:   logger,
		}, nil
	}

	registry := prometheus.NewRegistry()

	m := &MetricsRegistry{
		registry: registry,
		cfg:      cfg,
		logger:   logger,
	}

	if err := m.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	logger.Info("Metrics registry initialized",
		zap.String("provider", cfg.Monitoring.Metrics.Provider),
		zap.Int("port", cfg.Monitoring.Metrics.Port),
		zap.String("path", cfg.Monitoring.Metrics.Path),
	)

	return m, nil
}

func (m *MetricsRegistry) initializeMetrics() error { //nolint:funlen // This is a long function
	namespace := "trenova"

	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latencies in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response sizes in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "path"},
	)

	m.httpActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "active_requests",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	m.shipmentsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "shipments_processed_total",
			Help:      "Total number of shipments processed",
		},
		[]string{"status", "type", "customer"},
	)

	m.shipmentDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "shipment_processing_duration_seconds",
			Help:      "Time taken to process shipments in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"type"},
	)

	m.documentsUploaded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "documents_uploaded_total",
			Help:      "Total number of documents uploaded",
		},
		[]string{"type", "status"},
	)

	m.documentProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "document_processing_duration_seconds",
			Help:      "Time taken to process documents in seconds",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"type"},
	)

	m.complianceChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "compliance_checks_total",
			Help:      "Total number of compliance checks performed",
		},
		[]string{"type", "result"},
	)

	m.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "queries_total",
			Help:      "Total number of database queries executed",
		},
		[]string{"operation", "table", "status"},
	)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "query_duration_seconds",
			Help:      "Database query execution time in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"operation", "table"},
	)

	m.dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "connections_active",
			Help:      "Number of active database connections",
		},
	)

	m.dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "database",
			Name:      "connections_idle",
			Help:      "Number of idle database connections",
		},
	)

	m.cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache"},
	)

	m.cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache"},
	)

	m.cacheEvictions = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "evictions_total",
			Help:      "Total number of items evicted from cache",
		},
	)

	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "errors",
			Name:      "total",
			Help:      "Total number of errors by type and source",
		},
		[]string{"type", "source"},
	)

	m.panicRecoveries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "errors",
			Name:      "panic_recoveries_total",
			Help:      "Total number of recovered panics",
		},
	)

	m.activeUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "users",
			Name:      "active_total",
			Help:      "Number of currently active users",
		},
	)

	m.revenue = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "business",
			Name:      "revenue_total",
			Help:      "Total revenue processed in cents",
		},
		[]string{"currency", "type"},
	)

	m.apiKeyUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "api",
			Name:      "key_usage_total",
			Help:      "API key usage by key ID and endpoint",
		},
		[]string{"key_id", "endpoint"},
	)

	// Initialize WebSocket metrics
	m.wsConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "connections_active",
			Help:      "Number of active WebSocket connections",
		},
	)

	m.wsConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "connections_total",
			Help:      "Total number of WebSocket connections by organization and user",
		},
		[]string{"org_id", "user_id"},
	)

	m.wsMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "messages_sent_total",
			Help:      "Total number of WebSocket messages sent",
		},
		[]string{"type"}, // type: "notification", "pong", "broadcast", etc.
	)

	m.wsMessagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "messages_received_total",
			Help:      "Total number of WebSocket messages received",
		},
		[]string{"type"}, // type: "ping", "message", etc.
	)

	m.wsMessageSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "message_size_bytes",
			Help:      "Size of WebSocket messages in bytes",
			Buckets:   prometheus.ExponentialBuckets(64, 2, 10), // 64B to 32KB
		},
		[]string{"direction", "type"}, // direction: "sent", "received", type: message type
	)

	m.wsBroadcastsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "broadcasts_sent_total",
			Help:      "Total number of WebSocket broadcasts sent",
		},
		[]string{"target", "source"}, // target: "user", "org", "room", source: "local", "redis"
	)

	m.wsConnectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "connection_duration_seconds",
			Help:      "Duration of WebSocket connections in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}, // 1s to 1h
		},
		[]string{"org_id"},
	)

	m.wsRoomsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "rooms_active",
			Help:      "Number of active WebSocket rooms",
		},
	)

	m.wsUsersPerRoom = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "users_per_room",
			Help:      "Number of users per WebSocket room",
			Buckets:   []float64{1, 2, 5, 10, 20, 50, 100, 200, 500},
		},
		[]string{"room_id"},
	)

	m.wsConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "connection_errors_total",
			Help:      "Total number of WebSocket connection errors",
		},
		[]string{"error_type"}, // "upgrade_failed", "auth_failed", "read_error", "write_error"
	)

	m.wsPingPongLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "websocket",
			Name:      "ping_pong_latency_seconds",
			Help:      "Latency of ping-pong messages in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"user_id"},
	)

	m.streamingConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "streaming",
			Name:      "connections_active",
			Help:      "Number of active streaming connections",
		},
	)
	m.streamingConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "streaming",
			Name:      "connections_total",
			Help:      "Total number of streaming connections",
		},
		[]string{"org_id", "user_id", "entity"},
	)

	m.streamingMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "streaming",
			Name:      "messages_sent_total",
			Help:      "Total number of streaming messages sent by entity",
		},
		[]string{"entity"},
	)
	m.streamingBroadcastsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "streaming",
			Name:      "broadcasts_sent_total",
			Help:      "Total number of streaming broadcasts sent by entity",
		},
		[]string{"entity"},
	)

	// Initialize CDC metrics
	m.cdcMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "messages_total",
			Help:      "Total number of CDC messages processed",
		},
		[]string{"table", "operation", "status"}, // status: "success", "error"
	)

	m.cdcMessageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "message_duration_seconds",
			Help:      "Time taken to process CDC messages in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"table", "operation"},
	)

	m.cdcHandlerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "handler_errors_total",
			Help:      "Total number of CDC handler errors",
		},
		[]string{"table", "error_type"},
	)

	m.cdcSchemaCache = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "schema_cache_operations_total",
			Help:      "Total number of schema cache operations",
		},
		[]string{"operation"}, // operation: "hit", "miss", "eviction"
	)

	m.cdcConsumerLag = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "consumer_lag",
			Help:      "Current consumer lag in messages",
		},
	)

	m.cdcBatchSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "batch_size",
			Help:      "Number of messages processed per batch",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"table"},
	)

	m.cdcRebalances = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "rebalances_total",
			Help:      "Total number of consumer rebalances",
		},
	)

	m.cdcProcessingWorkers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "cdc",
			Name:      "processing_workers",
			Help:      "Number of active CDC processing workers",
		},
	)

	collectList := []prometheus.Collector{
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpResponseSize,
		m.httpActiveRequests,
		m.shipmentsProcessed,
		m.shipmentDuration,
		m.documentsUploaded,
		m.documentProcessTime,
		m.complianceChecks,
		m.dbQueriesTotal,
		m.dbQueryDuration,
		m.dbConnectionsActive,
		m.dbConnectionsIdle,
		m.cacheHits,
		m.cacheMisses,
		m.cacheEvictions,
		m.errorsTotal,
		m.panicRecoveries,
		m.activeUsers,
		m.revenue,
		m.apiKeyUsage,
		m.wsConnectionsActive,
		m.wsConnectionsTotal,
		m.wsMessagesSent,
		m.wsMessagesReceived,
		m.wsMessageSize,
		m.wsBroadcastsSent,
		m.wsConnectionDuration,
		m.wsRoomsActive,
		m.wsUsersPerRoom,
		m.wsConnectionErrors,
		m.wsPingPongLatency,
		m.streamingConnectionsActive,
		m.streamingConnectionsTotal,
		m.streamingMessagesSent,
		m.streamingBroadcastsSent,
		m.cdcMessagesTotal,
		m.cdcMessageDuration,
		m.cdcHandlerErrors,
		m.cdcSchemaCache,
		m.cdcConsumerLag,
		m.cdcBatchSize,
		m.cdcRebalances,
		m.cdcProcessingWorkers,
	}

	for _, collector := range collectList {
		if err := m.registry.Register(collector); err != nil {
			return fmt.Errorf("failed to register metric collector: %w", err)
		}
	}

	return nil
}

func (m *MetricsRegistry) IsEnabled() bool {
	return m.registry != nil
}

func (m *MetricsRegistry) Registry() *prometheus.Registry {
	return m.registry
}

func (m *MetricsRegistry) Handler() http.Handler {
	if !m.IsEnabled() {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Metrics collection is disabled"))
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

func (m *MetricsRegistry) RecordHTTPRequest(
	method, path string,
	status int,
	duration float64,
	responseSize int,
) {
	if !m.IsEnabled() {
		return
	}

	statusStr := strconv.Itoa(status)
	m.httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.httpRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration)
	m.httpResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

func (m *MetricsRegistry) IncrementActiveRequests() {
	if m.IsEnabled() {
		m.httpActiveRequests.Inc()
	}
}

func (m *MetricsRegistry) DecrementActiveRequests() {
	if m.IsEnabled() {
		m.httpActiveRequests.Dec()
	}
}

func (m *MetricsRegistry) RecordShipment(status, shipmentType, customer string, duration float64) {
	if !m.IsEnabled() {
		return
	}
	m.shipmentsProcessed.WithLabelValues(status, shipmentType, customer).Inc()
	m.shipmentDuration.WithLabelValues(shipmentType).Observe(duration)
}

func (m *MetricsRegistry) RecordDocument(docType, status string, duration float64) {
	if !m.IsEnabled() {
		return
	}
	m.documentsUploaded.WithLabelValues(docType, status).Inc()
	m.documentProcessTime.WithLabelValues(docType).Observe(duration)
}

func (m *MetricsRegistry) RecordComplianceCheck(checkType, result string) {
	if !m.IsEnabled() {
		return
	}
	m.complianceChecks.WithLabelValues(checkType, result).Inc()
}

func (m *MetricsRegistry) RecordDBQuery(operation, table, status string, duration float64) {
	if !m.IsEnabled() {
		return
	}
	m.dbQueriesTotal.WithLabelValues(operation, table, status).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

func (m *MetricsRegistry) UpdateDBConnections(active, idle int) {
	if !m.IsEnabled() {
		return
	}
	m.dbConnectionsActive.Set(float64(active))
	m.dbConnectionsIdle.Set(float64(idle))
}

func (m *MetricsRegistry) RecordCacheHit(cacheName string) {
	if m.IsEnabled() {
		m.cacheHits.WithLabelValues(cacheName).Inc()
	}
}

func (m *MetricsRegistry) RecordCacheMiss(cacheName string) {
	if m.IsEnabled() {
		m.cacheMisses.WithLabelValues(cacheName).Inc()
	}
}

func (m *MetricsRegistry) RecordCacheEviction() {
	if m.IsEnabled() {
		m.cacheEvictions.Inc()
	}
}

func (m *MetricsRegistry) RecordError(errorType, source string) {
	if m.IsEnabled() {
		m.errorsTotal.WithLabelValues(errorType, source).Inc()
	}
}

func (m *MetricsRegistry) RecordPanicRecovery() {
	if m.IsEnabled() {
		m.panicRecoveries.Inc()
	}
}

func (m *MetricsRegistry) UpdateActiveUsers(count int) {
	if m.IsEnabled() {
		m.activeUsers.Set(float64(count))
	}
}

func (m *MetricsRegistry) RecordRevenue(amount float64, currency, revenueType string) {
	if m.IsEnabled() {
		m.revenue.WithLabelValues(currency, revenueType).Add(amount)
	}
}

func (m *MetricsRegistry) RecordAPIKeyUsage(keyID, endpoint string) {
	if m.IsEnabled() {
		m.apiKeyUsage.WithLabelValues(keyID, endpoint).Inc()
	}
}

func (m *MetricsRegistry) StartResourceMonitor(ctx context.Context) {
	if !m.IsEnabled() {
		return
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.collectResourceMetrics()
			}
		}
	}()
}

func (m *MetricsRegistry) collectResourceMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.logger.Debug("Resource metrics collected",
		zap.Uint64("alloc_mb", memStats.Alloc/1024/1024),
		zap.Uint64("sys_mb", memStats.Sys/1024/1024),
		zap.Uint32("num_gc", memStats.NumGC),
		zap.Int("goroutines", runtime.NumGoroutine()),
	)
}

func (m *MetricsRegistry) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.IsEnabled() {
			c.Next()
			return
		}

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		m.IncrementActiveRequests()
		defer m.DecrementActiveRequests()

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		size := c.Writer.Size()

		m.RecordHTTPRequest(c.Request.Method, path, status, duration, size)
	}
}

// WebSocket metrics helpers

func (m *MetricsRegistry) RecordWSConnection(orgID, userID string) {
	if !m.IsEnabled() {
		return
	}
	m.wsConnectionsActive.Inc()
	m.wsConnectionsTotal.WithLabelValues(orgID, userID).Inc()
}

func (m *MetricsRegistry) RecordWSDisconnection(orgID string, connectionStart time.Time) {
	if !m.IsEnabled() {
		return
	}
	m.wsConnectionsActive.Dec()
	duration := time.Since(connectionStart).Seconds()
	m.wsConnectionDuration.WithLabelValues(orgID).Observe(duration)
}

func (m *MetricsRegistry) RecordWSMessage(direction, msgType string, size int) {
	if !m.IsEnabled() {
		return
	}
	if direction == "sent" {
		m.wsMessagesSent.WithLabelValues(msgType).Inc()
	} else {
		m.wsMessagesReceived.WithLabelValues(msgType).Inc()
	}
	m.wsMessageSize.WithLabelValues(direction, msgType).Observe(float64(size))
}

func (m *MetricsRegistry) RecordWSBroadcast(broadcastType string, recipientCount int) {
	if !m.IsEnabled() {
		return
	}
	m.wsBroadcastsSent.WithLabelValues(broadcastType).Add(float64(recipientCount))
}

func (m *MetricsRegistry) UpdateWSRooms(activeRooms int) {
	if !m.IsEnabled() {
		return
	}
	m.wsRoomsActive.Set(float64(activeRooms))
}

func (m *MetricsRegistry) RecordWSRoomSize(roomID string, userCount int) {
	if !m.IsEnabled() {
		return
	}
	m.wsUsersPerRoom.WithLabelValues(roomID).Observe(float64(userCount))
}

func (m *MetricsRegistry) RecordWSError(errorType string) {
	if !m.IsEnabled() {
		return
	}
	m.wsConnectionErrors.WithLabelValues(errorType).Inc()
}

func (m *MetricsRegistry) RecordWSPingLatency(userID string, latency float64) {
	if !m.IsEnabled() {
		return
	}
	m.wsPingPongLatency.WithLabelValues(userID).Observe(latency)
}

func (m *MetricsRegistry) RecordStreamingConnection(orgID, userID, entity string) {
	if !m.IsEnabled() {
		return
	}
	m.streamingConnectionsActive.Inc()
	m.streamingConnectionsTotal.WithLabelValues(orgID, userID, entity).Inc()
}

// CDC metrics helpers

func (m *MetricsRegistry) RecordCDCMessage(table, operation, status string, duration float64) {
	if !m.IsEnabled() {
		return
	}
	m.cdcMessagesTotal.WithLabelValues(table, operation, status).Inc()
	m.cdcMessageDuration.WithLabelValues(table, operation).Observe(duration)
}

func (m *MetricsRegistry) RecordCDCHandlerError(table, errorType string) {
	if !m.IsEnabled() {
		return
	}
	m.cdcHandlerErrors.WithLabelValues(table, errorType).Inc()
}

func (m *MetricsRegistry) RecordCDCSchemaCache(operation string) {
	if !m.IsEnabled() {
		return
	}
	m.cdcSchemaCache.WithLabelValues(operation).Inc()
}

func (m *MetricsRegistry) UpdateCDCConsumerLag(lag float64) {
	if !m.IsEnabled() {
		return
	}
	m.cdcConsumerLag.Set(lag)
}

func (m *MetricsRegistry) RecordCDCBatchSize(table string, size int) {
	if !m.IsEnabled() {
		return
	}
	m.cdcBatchSize.WithLabelValues(table).Observe(float64(size))
}

func (m *MetricsRegistry) RecordCDCRebalance() {
	if !m.IsEnabled() {
		return
	}
	m.cdcRebalances.Inc()
}

func (m *MetricsRegistry) UpdateCDCProcessingWorkers(count int) {
	if !m.IsEnabled() {
		return
	}
	m.cdcProcessingWorkers.Set(float64(count))
}
