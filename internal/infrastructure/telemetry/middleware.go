package telemetry

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerCache caches tracers to avoid repeated lookups
var (
	tracerCache = make(map[string]trace.Tracer)
	tracerMu    sync.RWMutex
)

// getTracer returns a cached tracer or creates a new one
func getTracer(serviceName string) trace.Tracer {
	tracerMu.RLock()
	tracer, exists := tracerCache[serviceName]
	tracerMu.RUnlock()

	if exists {
		return tracer
	}

	tracerMu.Lock()
	defer tracerMu.Unlock()

	if tr, ok := tracerCache[serviceName]; ok {
		return tr
	}

	tracer = otel.Tracer(serviceName)
	tracerCache[serviceName] = tracer
	return tracer
}

// TracingConfig provides configuration for the tracing middleware
type TracingConfig struct {
	ServiceName    string
	Tracer         trace.Tracer
	SkipPaths      []string
	SensitivePaths []string
	RecordBody     bool
	MaxBodySize    int
}

// DefaultTracingConfig returns a default configuration
func DefaultTracingConfig(serviceName string) TracingConfig {
	return TracingConfig{
		ServiceName: serviceName,
		SkipPaths:   []string{"/health", "/metrics", "/favicon.ico"},
		MaxBodySize: 65536,
		RecordBody:  false,
	}
}

// NewTracingMiddleware creates a new tracing middleware with configuration
//
//nolint:gocritic // this is dependnecy injection
func NewTracingMiddleware(cfg TracingConfig) fiber.Handler {
	tracer := cfg.Tracer
	if tracer == nil {
		tracer = getTracer(cfg.ServiceName)
	}

	skipMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	sensitiveMap := make(map[string]bool, len(cfg.SensitivePaths))
	for _, path := range cfg.SensitivePaths {
		sensitiveMap[path] = true
	}

	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		if skipMap[c.Path()] {
			return c.Next()
		}

		ctx := c.UserContext()

		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(c.GetReqHeaders()))

		spanName := buildSpanName(c)

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(buildRequestAttributes(c, &cfg, sensitiveMap)...),
		)
		defer span.End()

		c.SetUserContext(ctx)

		propagator.Inject(ctx, propagation.HeaderCarrier(c.GetRespHeaders()))

		err := c.Next()

		recordResponseAttributes(span, c, &cfg)

		setSpanStatus(span, c.Response().StatusCode(), err)

		return err
	}
}

// buildSpanName constructs a meaningful span name
func buildSpanName(c *fiber.Ctx) string {
	if route := c.Route(); route != nil {
		return fmt.Sprintf("%s %s", c.Method(), route.Path)
	}

	return fmt.Sprintf("%s %s", c.Method(), c.Path())
}

// buildRequestAttributes builds request attributes with security considerations
func buildRequestAttributes(
	c *fiber.Ctx,
	cfg *TracingConfig,
	sensitiveMap map[string]bool,
) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(c.Method()),
		semconv.URLPath(c.Path()),
		semconv.UserAgentOriginal(c.Get(fiber.HeaderUserAgent)),
		semconv.ClientAddress(c.IP()),
	}

	if route := c.Route(); route != nil {
		attrs = append(attrs, semconv.HTTPRouteKey.String(route.Path))
	}

	if !sensitiveMap[c.Path()] {
		attrs = append(attrs, attribute.String("http.url", c.OriginalURL()))
	}

	if cfg.RecordBody {
		bodySize := len(c.Body())
		attrs = append(attrs, semconv.HTTPRequestBodySizeKey.Int(bodySize))

		if bodySize > 0 && bodySize <= cfg.MaxBodySize && !sensitiveMap[c.Path()] {
			attrs = append(attrs, attribute.String("http.request.body", string(c.Body())))
		}
	}

	return attrs
}

// recordResponseAttributes records response attributes
func recordResponseAttributes(span trace.Span, c *fiber.Ctx, cfg *TracingConfig) {
	statusCode := c.Response().StatusCode()
	respBodySize := len(c.Response().Body())

	attrs := []attribute.KeyValue{
		semconv.HTTPResponseStatusCode(statusCode),
		semconv.HTTPResponseBodySizeKey.Int(respBodySize),
	}

	if cfg.RecordBody && respBodySize > 0 && respBodySize <= cfg.MaxBodySize {
		if statusCode >= 400 {
			attrs = append(
				attrs,
				attribute.String("http.response.body", string(c.Response().Body())),
			)
		}
	}

	span.SetAttributes(attrs...)
}

// setSpanStatus sets the span status based on HTTP status code and error
func setSpanStatus(span trace.Span, statusCode int, err error) {
	switch {
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	case statusCode >= http.StatusBadRequest:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	case statusCode >= http.StatusInternalServerError:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	case statusCode >= http.StatusBadGateway:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	case statusCode >= http.StatusGatewayTimeout:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	case statusCode >= http.StatusTooManyRequests:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	case statusCode >= http.StatusServiceUnavailable:
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	default:
		span.SetStatus(codes.Ok, http.StatusText(statusCode))
	}
}

// MetricsConfig provides configuration for the metrics middleware
type MetricsConfig struct {
	Metrics      *Metrics
	SkipPaths    []string
	RecordSize   bool
	GroupedPaths map[string]string
}

// DefaultMetricsConfig returns a default configuration
func DefaultMetricsConfig(metrics *Metrics) MetricsConfig {
	return MetricsConfig{
		Metrics:      metrics,
		SkipPaths:    []string{"/health", "/metrics", "/favicon.ico"},
		RecordSize:   true,
		GroupedPaths: map[string]string{
			// ! Group similar paths to avoid high cardinality
			// ! Example: "/api/v1/users/123" -> "/api/v1/users/:id"
		},
	}
}

// NewMetricsMiddleware creates a new metrics middleware with configuration
func NewMetricsMiddleware(cfg MetricsConfig) fiber.Handler {
	if cfg.Metrics == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// ! Pre-compile skip paths for performance
	skipMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		if skipMap[c.Path()] {
			return c.Next()
		}

		start := time.Now()
		ctx := c.UserContext()

		cfg.Metrics.RecordActiveHTTPRequest(ctx, 1)
		defer cfg.Metrics.RecordActiveHTTPRequest(ctx, -1)

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		routePath := getRoutePathForMetrics(c, cfg.GroupedPaths)

		var requestSize, responseSize int64
		if cfg.RecordSize {
			requestSize = int64(len(c.Body()))
			responseSize = int64(len(c.Response().Body()))
		}

		cfg.Metrics.RecordHTTPRequest(
			ctx,
			c.Method(),
			routePath,
			statusCode,
			duration,
			requestSize,
			responseSize,
		)

		return err
	}
}

// getRoutePathForMetrics returns the appropriate route path for metrics
func getRoutePathForMetrics(c *fiber.Ctx, groupedPaths map[string]string) string {
	if grouped, exists := groupedPaths[c.Path()]; exists {
		return grouped
	}

	if route := c.Route(); route != nil {
		return route.Path
	}

	return c.Path()
}

// LoggingConfig provides configuration for the logging middleware
type LoggingConfig struct {
	Logger         *logger.Logger
	SkipPaths      []string
	LogBody        bool
	MaxBodySize    int
	SensitivePaths []string
	LogHeaders     bool
	SlowThreshold  time.Duration
}

// DefaultLoggingConfig returns a default configuration
func DefaultLoggingConfig(log *logger.Logger) *LoggingConfig {
	return &LoggingConfig{
		Logger:        log,
		SkipPaths:     []string{"/health", "/metrics", "/favicon.ico"},
		LogBody:       false,
		MaxBodySize:   1024,
		LogHeaders:    false,
		SlowThreshold: 5 * time.Second,
	}
}

// NewLoggingMiddleware creates a new logging middleware with configuration
func NewLoggingMiddleware(cfg *LoggingConfig) fiber.Handler {
	if cfg.Logger == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	skipMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	sensitiveMap := make(map[string]bool, len(cfg.SensitivePaths))
	for _, path := range cfg.SensitivePaths {
		sensitiveMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		if skipMap[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		logEvent := determineLogLevel(cfg.Logger, statusCode, duration, cfg.SlowThreshold)

		ctx := c.UserContext()
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			logEvent = logEvent.
				Str("trace_id", span.SpanContext().TraceID().String()).
				Str("span_id", span.SpanContext().SpanID().String())
		}

		logEvent = buildLogEvent(logEvent, c, statusCode, duration, err, cfg, sensitiveMap)
		logEvent.Msg("HTTP request")

		return err
	}
}

// determineLogLevel returns the appropriate log level based on response
func determineLogLevel(
	log *logger.Logger,
	statusCode int,
	duration time.Duration,
	slowThreshold time.Duration,
) *zerolog.Event {
	switch {
	case statusCode >= 500:
		return log.Error()
	case statusCode >= 400:
		return log.Warn()
	case duration > slowThreshold:
		return log.Warn()
	default:
		return log.Info()
	}
}

// buildLogEvent builds the log event with appropriate fields
func buildLogEvent(
	event *zerolog.Event,
	c *fiber.Ctx,
	statusCode int,
	duration time.Duration,
	err error,
	cfg *LoggingConfig,
	sensitiveMap map[string]bool,
) *zerolog.Event {
	event = event.
		Str("method", c.Method()).
		Str("path", c.Path()).
		Int("status", statusCode).
		Dur("duration", duration).
		Str("ip", c.IP()).
		Str("user_agent", c.Get(fiber.HeaderUserAgent))

	if route := c.Route(); route != nil {
		event = event.Str("route", route.Path)
	}

	if err != nil {
		event = event.Err(err)
	}

	event = event.
		Int("bytes_sent", len(c.Response().Body())).
		Int("bytes_received", len(c.Body()))

	if cfg.LogHeaders && !sensitiveMap[c.Path()] {
		event = event.Interface("request_headers", c.GetReqHeaders())
	}

	if reqID := c.Get(fiber.HeaderXRequestID); reqID != "" {
		event = event.Str("request_id", reqID)
	}

	if cfg.LogBody && !sensitiveMap[c.Path()] {
		if reqBody := c.Body(); len(reqBody) > 0 && len(reqBody) <= cfg.MaxBodySize {
			event = event.Str("request_body", string(reqBody))
		}

		if statusCode >= 400 {
			if respBody := c.Response().Body(); len(respBody) > 0 &&
				len(respBody) <= cfg.MaxBodySize {
				event = event.Str("response_body", string(respBody))
			}
		}
	}

	return event
}

// RecoveryConfig provides configuration for the recovery middleware
type RecoveryConfig struct {
	Logger              *logger.Logger
	EnableStackTrace    bool
	PrintStack          bool
	CustomErrorResponse func(*fiber.Ctx)
}

// DefaultRecoveryConfig returns a default configuration
func DefaultRecoveryConfig(log *logger.Logger) RecoveryConfig {
	return RecoveryConfig{
		Logger:           log,
		EnableStackTrace: true,
		PrintStack:       false,
	}
}

// NewRecoveryMiddleware creates a new recovery middleware with configuration
func NewRecoveryMiddleware(cfg RecoveryConfig) fiber.Handler {
	if cfg.Logger == nil {
		return func(c *fiber.Ctx) error {
			defer func() {
				if r := recover(); r != nil {
					_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Internal server error",
					})
				}
			}()
			return c.Next()
		}
	}

	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				handlePanic(c, r, cfg)
			}
		}()

		return c.Next()
	}
}

// handlePanic handles a panic recovery with proper logging and response
func handlePanic(c *fiber.Ctx, recovered any, cfg RecoveryConfig) {
	ctx := c.UserContext()

	var err error
	switch v := recovered.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf("%s", v)
	default:
		err = fmt.Errorf("panic: %v", v)
	}

	logEvent := cfg.Logger.Error().
		Err(err).
		Interface("panic", recovered).
		Str("method", c.Method()).
		Str("path", c.Path()).
		Str("ip", c.IP())

	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		logEvent = logEvent.
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String())

		span.RecordError(err)
		span.SetStatus(codes.Error, "Panic recovered")
	}

	if cfg.EnableStackTrace {
		stackTrace := getStackTrace(3) // ! Skip runtime.gopanic, this func, and defer
		logEvent = logEvent.Str("stack_trace", stackTrace)

		if cfg.PrintStack {
			fmt.Fprintf(c.Response().BodyWriter(), "PANIC: %v\n%s", recovered, stackTrace)
		}
	}

	if route := c.Route(); route != nil {
		logEvent = logEvent.Str("route", route.Path)
	}

	logEvent.Msg("Panic recovered")

	if cfg.CustomErrorResponse != nil {
		cfg.CustomErrorResponse(c)
	} else {
		_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal server error",
			"message": "An unexpected error occurred",
		})
	}
}

// getStackTrace returns a formatted stack trace
func getStackTrace(skip int) string {
	const maxStackSize = 64 * 1024 // 64KB
	buf := make([]byte, maxStackSize)
	n := runtime.Stack(buf, false)

	stack := string(buf[:n])

	lines := strings.Split(stack, "\n")
	if skip*2 < len(lines) {
		lines = lines[skip*2:]
	}

	return strings.Join(lines, "\n")
}

// CombinedMiddleware combines all telemetry middleware with proper configuration
type CombinedMiddleware struct {
	Tracing  fiber.Handler
	Metrics  fiber.Handler
	Logging  fiber.Handler
	Recovery fiber.Handler
}

// NewCombinedMiddleware creates all telemetry middleware with sensible defaults
func NewCombinedMiddleware(
	serviceName string,
	metrics *Metrics,
	log *logger.Logger,
) CombinedMiddleware {
	return CombinedMiddleware{
		Tracing:  NewTracingMiddleware(DefaultTracingConfig(serviceName)),
		Metrics:  NewMetricsMiddleware(DefaultMetricsConfig(metrics)),
		Logging:  NewLoggingMiddleware(DefaultLoggingConfig(log)),
		Recovery: NewRecoveryMiddleware(DefaultRecoveryConfig(log)),
	}
}

// Apply applies all middleware in the correct order
func (m CombinedMiddleware) Apply(app fiber.Router) {
	// ! Order matters: Recovery -> Tracing -> Metrics -> Logging
	app.Use(m.Recovery)
	app.Use(m.Tracing)
	app.Use(m.Metrics)
	app.Use(m.Logging)
}
