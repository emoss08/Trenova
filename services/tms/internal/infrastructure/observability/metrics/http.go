package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type HTTP struct {
	Base
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	activeRequests  prometheus.Gauge
}

func NewHTTP(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *HTTP {
	m := &HTTP{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latencies in seconds",
			Buckets:   HTTPDurationBuckets,
		},
		[]string{"method", "path", "status"},
	)

	m.responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response sizes in bytes",
			Buckets:   HTTPResponseSizeBuckets,
		},
		[]string{"method", "path"},
	)

	m.activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "active_requests",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	m.mustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.responseSize,
		m.activeRequests,
	)

	return m
}

func (m *HTTP) RecordHTTPRequest(
	method, path string,
	status int,
	duration float64,
	responseSize int,
) {
	if !m.IsEnabled() {
		return
	}
	statusStr := strconv.Itoa(status)
	m.requestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.requestDuration.WithLabelValues(method, path, statusStr).Observe(duration)
	m.responseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

func (m *HTTP) IncrementActiveRequests() {
	m.ifEnabled(func() { m.activeRequests.Inc() })
}

func (m *HTTP) DecrementActiveRequests() {
	m.ifEnabled(func() { m.activeRequests.Dec() })
}

func (m *HTTP) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.IsEnabled() {
			c.Next()
			return
		}

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = UnknownValue
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
