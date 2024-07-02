package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// Define the Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	requestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 5),
		},
		[]string{"method", "path"},
	)
	responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 5),
		},
		[]string{"method", "path"},
	)
	httpRequestsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "Number of HTTP requests in progress.",
		},
		[]string{"method", "path"},
	)
	httpResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_response_status_total",
			Help: "Total number of HTTP response status codes.",
		},
		[]string{"status"},
	)
)

func init() {
	// Register the metrics with Prometheus only once
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(requestSize)
	prometheus.MustRegister(responseSize)
	prometheus.MustRegister(httpRequestsInProgress)
	prometheus.MustRegister(httpResponseStatus)
}

// PrometheusMiddleware is a Fiber middleware that collects Prometheus metrics
func PrometheusMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()
		httpRequestsInProgress.WithLabelValues(method, path).Inc()
		defer httpRequestsInProgress.WithLabelValues(method, path).Dec()

		// Process the request
		err := c.Next()

		// Update the metrics
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(method, path).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
		requestSize.WithLabelValues(method, path).Observe(float64(len(c.Request().Body())))
		responseSize.WithLabelValues(method, path).Observe(float64(len(c.Response().Body())))
		httpResponseStatus.WithLabelValues(strconv.Itoa(c.Response().StatusCode())).Inc()

		return err
	}
}
