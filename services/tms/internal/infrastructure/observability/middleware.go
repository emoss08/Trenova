package observability

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Middleware provides observability middleware for Gin
type Middleware struct {
	tracer  *TracerProvider
	metrics *MetricsRegistry
	logger  *zap.Logger
}

// NewMiddleware creates a new observability middleware
func NewMiddleware(
	tracer *TracerProvider,
	metrics *MetricsRegistry,
	logger *zap.Logger,
) *Middleware {
	return &Middleware{
		tracer:  tracer,
		metrics: metrics,
		logger:  logger,
	}
}

// CombinedMiddleware returns a combined tracing and metrics middleware
func (m *Middleware) TracingMiddleware() gin.HandlerFunc { //nolint:gocognit,cyclop,funlen // This is a long function
	return func(c *gin.Context) {
		start := time.Now()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		var span trace.Span
		if m.tracer.IsEnabled() {
			ctx := propagation.TraceContext{}.Extract(
				c.Request.Context(),
				propagation.HeaderCarrier(c.Request.Header),
			)

			spanName := fmt.Sprintf("%s %s", c.Request.Method, path)

			ctx, span = m.tracer.Tracer().Start(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", c.Request.Method),
					attribute.String("http.target", c.Request.URL.String()),
					attribute.String("http.route", path),
					attribute.String("net.peer.ip", c.ClientIP()),
					attribute.String("http.user_agent", c.Request.UserAgent()),
				),
			)
			defer span.End()

			c.Request = c.Request.WithContext(ctx)

			if span.SpanContext().HasTraceID() {
				c.Set("trace_id", span.SpanContext().TraceID().String())
				c.Set("span_id", span.SpanContext().SpanID().String())
			}
		}

		if m.metrics.IsEnabled() {
			m.metrics.IncrementActiveRequests()
			defer m.metrics.DecrementActiveRequests()
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		responseSize := c.Writer.Size()

		if span != nil && span.IsRecording() {
			span.SetAttributes(
				attribute.Int("http.status_code", status),
				attribute.Int("http.response_content_length", responseSize),
				attribute.Float64("http.request.duration", duration),
			)

			if status >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))

				for _, err := range c.Errors {
					span.RecordError(err.Err)
				}
			} else {
				span.SetStatus(codes.Ok, "")
			}

			if userID, exists := c.Get("user_id"); exists {
				span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
			}
			if orgID, exists := c.Get("organization_id"); exists {
				span.SetAttributes(attribute.String("organization.id", fmt.Sprintf("%v", orgID)))
			}
		}

		if m.metrics.IsEnabled() {
			m.metrics.RecordHTTPRequest(
				c.Request.Method,
				path,
				status,
				duration,
				responseSize,
			)

			if apiKeyID, exists := c.Get("api_key_id"); exists {
				m.metrics.RecordAPIKeyUsage(fmt.Sprintf("%v", apiKeyID), path)
			}

			if len(c.Errors) > 0 {
				for _, err := range c.Errors {
					errorType := "handler"
					switch err.Type { //nolint:exhaustive // This is a valid use case
					case gin.ErrorTypePublic:
						errorType = "public"
					case gin.ErrorTypePrivate:
						errorType = "private"
					}
					m.metrics.RecordError(errorType, path)
				}
			}
		}

		// Skip slow request warning for WebSocket connections (they're long-lived by design)
		isWebSocket := c.GetHeader("Upgrade") == "websocket" ||
			(path != "" && (strings.Contains(path, "/ws/") || strings.Contains(path, "/websocket")))
		isLiveMode := path != "" &&
			(strings.Contains(path, "/live") || strings.Contains(path, "/live-mode"))

		if duration > 1.0 && !isWebSocket && !isLiveMode {
			m.logger.Warn("Slow request detected",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Float64("duration_seconds", duration),
				zap.String("trace_id", c.GetString("trace_id")),
			)
		}
	}
}
