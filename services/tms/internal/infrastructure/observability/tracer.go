package observability

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TracerProviderParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *config.Config
	Logger    *zap.Logger
}

type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	cfg      *config.Config
	l        *zap.Logger
}

func NewTracerProvider(p TracerProviderParams) (*TracerProvider, error) {
	log := p.Logger.With(zap.String("component", "tracer"))

	if !p.Config.Monitoring.Tracing.Enabled {
		log.Warn("🟡 Tracing is disabled")
		noopProvider := noop.NewTracerProvider()
		return &TracerProvider{
			provider: nil,
			tracer:   noopProvider.Tracer("noop"),
			cfg:      p.Config,
			l:        log,
		}, nil
	}

	tp := &TracerProvider{
		cfg: p.Config,
		l:   log,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return tp.initialize(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	noopProvider := noop.NewTracerProvider()
	tp.tracer = noopProvider.Tracer("noop")

	return tp, nil
}

func (tp *TracerProvider) initialize(ctx context.Context) error {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(tp.cfg.Monitoring.Tracing.ServiceName),
		semconv.ServiceVersion(tp.cfg.App.Version),
		attribute.String("deployment.environment", tp.cfg.App.Env),
		attribute.String("service.namespace", "trenova"),
		attribute.String("service.instance.id", generateInstanceID()),
	)

	exporter, err := createExporter(ctx, &tp.cfg.Monitoring.Tracing, tp.l)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxQueueSize(2048),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(createSampler(&tp.cfg.Monitoring.Tracing)),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tp.provider = provider
	tp.tracer = provider.Tracer("trenova")

	tp.l.Info("Tracer provider initialized",
		zap.String("provider", tp.cfg.Monitoring.Tracing.Provider),
		zap.String("endpoint", tp.cfg.Monitoring.Tracing.Endpoint),
		zap.Float64("sampling_rate", tp.cfg.Monitoring.Tracing.SamplingRate),
	)

	return nil
}

func createExporter(
	ctx context.Context,
	cfg *config.TracingConfig,
	logger *zap.Logger,
) (sdktrace.SpanExporter, error) {
	switch cfg.Provider {
	case "jaeger":
		logger.Info("Using OTLP exporter for Jaeger (recommended approach)")
		return createOTLPExporter(ctx, cfg)
	case "zipkin":
		return createZipkinExporter(cfg)
	case "otlp":
		return createOTLPExporter(ctx, cfg)
	case "otlp-grpc":
		return createOTLPGRPCExporter(ctx, cfg)
	case "stdout":
		return createStdoutExporter()
	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s", cfg.Provider)
	}
}

func createOTLPGRPCExporter(
	ctx context.Context,
	cfg *config.TracingConfig,
) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if os.Getenv("APP_ENV") != "production" {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

func createZipkinExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	return zipkin.New(cfg.Endpoint)
}

func createOTLPExporter(
	ctx context.Context,
	cfg *config.TracingConfig,
) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}

	if os.Getenv("APP_ENV") != "production" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	client := otlptracehttp.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

func createStdoutExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
}

func createSampler(cfg *config.TracingConfig) sdktrace.Sampler {
	if cfg.SamplingRate >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if cfg.SamplingRate <= 0.0 {
		return sdktrace.NeverSample()
	}

	return sdktrace.TraceIDRatioBased(cfg.SamplingRate)
}

func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

func (tp *TracerProvider) IsEnabled() bool {
	return tp.provider != nil
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}

	tp.l.Info("Shutting down tracer provider")
	if err := tp.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}
	return nil
}

func (tp *TracerProvider) StartSpan(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return tp.tracer.Start( //nolint:spancheck // Caller is responsible for ending span
		ctx,
		name,
		opts...)
}

func (tp *TracerProvider) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

func (tp *TracerProvider) SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

func (tp *TracerProvider) RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err, opts...)
	}
}

func generateInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
}
