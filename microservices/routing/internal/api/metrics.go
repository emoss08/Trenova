package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the routing service
type Metrics struct {
	// _ HTTP metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec

	// _ Route calculation metrics
	RouteCalculations    *prometheus.CounterVec
	RouteCalculationTime *prometheus.HistogramVec
	RouteDistance        *prometheus.HistogramVec
	RouteNodesSearched   *prometheus.HistogramVec

	// _ Cache metrics
	CacheHits    *prometheus.CounterVec
	CacheMisses  *prometheus.CounterVec
	CacheLatency *prometheus.HistogramVec

	// _ Algorithm metrics
	AlgorithmUsage        *prometheus.CounterVec
	OptimizationTypeUsage *prometheus.CounterVec

	// _ Error metrics
	ErrorsTotal *prometheus.CounterVec

	// _ Graph metrics
	GraphNodes       prometheus.Gauge
	GraphEdges       prometheus.Gauge
	GraphMemoryBytes prometheus.Gauge

	// _ Import metrics
	LastImportTime    prometheus.Gauge
	ImportedNodes     prometheus.Gauge
	ImportedEdges     prometheus.Gauge
	TruckRestrictions *prometheus.GaugeVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// _ HTTP metrics
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		ResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),

		// _ Route calculation metrics
		RouteCalculations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_calculations_total",
				Help: "Total number of route calculations",
			},
			[]string{"vehicle_type", "cache_hit"},
		),
		RouteCalculationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_calculation_duration_seconds",
				Help:    "Route calculation duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"algorithm", "optimization_type"},
		),
		RouteDistance: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_distance_miles",
				Help:    "Calculated route distance in miles",
				Buckets: []float64{10, 50, 100, 250, 500, 1000, 2000, 3000},
			},
			[]string{"vehicle_type"},
		),
		RouteNodesSearched: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_nodes_searched",
				Help:    "Number of nodes searched during route calculation",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 200000},
			},
			[]string{"algorithm"},
		),

		// _ Cache metrics
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"}, // redis, postgres
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		CacheLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_cache_latency_seconds",
				Help:    "Cache operation latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"cache_type", "operation"}, // get, set
		),

		// _ Algorithm metrics
		AlgorithmUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_algorithm_usage_total",
				Help: "Total usage count per algorithm",
			},
			[]string{"algorithm"},
		),
		OptimizationTypeUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_optimization_type_usage_total",
				Help: "Total usage count per optimization type",
			},
			[]string{"optimization_type"},
		),

		// _ Error metrics
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_errors_total",
				Help: "Total number of errors",
			},
			[]string{"error_type"},
		),

		// _ Graph metrics
		GraphNodes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_graph_nodes",
				Help: "Total number of nodes in the graph",
			},
		),
		GraphEdges: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_graph_edges",
				Help: "Total number of edges in the graph",
			},
		),
		GraphMemoryBytes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_graph_memory_bytes",
				Help: "Memory usage of the graph in bytes",
			},
		),

		// _ Import metrics
		LastImportTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_last_import_timestamp",
				Help: "Timestamp of the last import",
			},
		),
		ImportedNodes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_imported_nodes_total",
				Help: "Total number of imported nodes",
			},
		),
		ImportedEdges: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_imported_edges_total",
				Help: "Total number of imported edges",
			},
		),
		TruckRestrictions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "routing_truck_restrictions_total",
				Help: "Total number of truck restrictions by type",
			},
			[]string{"restriction_type"}, // height, weight, length, hazmat, etc.
		),
	}

	// _ Register all metrics
	prometheus.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.ResponseSize,
		m.RouteCalculations,
		m.RouteCalculationTime,
		m.RouteDistance,
		m.RouteNodesSearched,
		m.CacheHits,
		m.CacheMisses,
		m.CacheLatency,
		m.AlgorithmUsage,
		m.OptimizationTypeUsage,
		m.ErrorsTotal,
		m.GraphNodes,
		m.GraphEdges,
		m.GraphMemoryBytes,
		m.LastImportTime,
		m.ImportedNodes,
		m.ImportedEdges,
		m.TruckRestrictions,
	)

	return m
}

// PrometheusHandler returns a Fiber handler for Prometheus metrics
func PrometheusHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

// RecordHTTPMetrics is a middleware that records HTTP metrics
func (m *Metrics) RecordHTTPMetrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		timer := prometheus.NewTimer(m.RequestDuration.WithLabelValues(
			c.Method(),
			c.Path(),
		))

		// _ Process request
		err := c.Next()

		// _ Record metrics
		timer.ObserveDuration()

		m.RequestsTotal.WithLabelValues(
			c.Method(),
			c.Path(),
			string(rune(c.Response().StatusCode())),
		).Inc()

		m.ResponseSize.WithLabelValues(
			c.Method(),
			c.Path(),
		).Observe(float64(len(c.Response().Body())))

		return err
	}
}

// RecordRouteCalculation records metrics for a route calculation
func (m *Metrics) RecordRouteCalculation(
	vehicleType string,
	cacheHit bool,
	distance float64,
	duration float64,
	algorithm string,
	optimizationType string,
	nodesSearched int,
) {
	// _ Count the calculation
	m.RouteCalculations.WithLabelValues(vehicleType, formatBool(cacheHit)).Inc()

	if !cacheHit {
		// _ Record calculation time
		m.RouteCalculationTime.WithLabelValues(algorithm, optimizationType).Observe(duration)

		// _ Record distance
		m.RouteDistance.WithLabelValues(vehicleType).Observe(distance)

		// _ Record nodes searched
		m.RouteNodesSearched.WithLabelValues(algorithm).Observe(float64(nodesSearched))

		// _ Record algorithm usage
		m.AlgorithmUsage.WithLabelValues(algorithm).Inc()
		m.OptimizationTypeUsage.WithLabelValues(optimizationType).Inc()
	}
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.CacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordError records an error
func (m *Metrics) RecordError(errorType string) {
	m.ErrorsTotal.WithLabelValues(errorType).Inc()
}

// UpdateGraphStats updates graph statistics
func (m *Metrics) UpdateGraphStats(nodes, edges int, memoryBytes int64) {
	m.GraphNodes.Set(float64(nodes))
	m.GraphEdges.Set(float64(edges))
	m.GraphMemoryBytes.Set(float64(memoryBytes))
}

// formatBool converts bool to string for labels
func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
