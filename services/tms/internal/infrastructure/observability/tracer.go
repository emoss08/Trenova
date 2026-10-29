package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
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

	Lifecycle        fx.Lifecycle
	AppConfig        *appconfig.Config
	MonitoringConfig *monitoringconfig.Config
	Logger           *zap.Logger
}

type TracerProvider struct {
	provider         *sdktrace.TracerProvider
	tracer           trace.Tracer
	appConfig        *appconfig.Config
	monitoringConfig *monitoringconfig.Config
	l                *zap.Logger
}

func NewTracerProvider(p TracerProviderParams) (*TracerProvider, error) {
	log := p.Logger.With(zap.String("component", "tracer"))
	aitrace.SetSamplingRate(p.MonitoringConfig.Tracing.GetAISamplingRate())

	if !p.MonitoringConfig.Tracing.Enabled {
		log.Warn("🟡 Tracing is disabled")
		noopProvider := noop.NewTracerProvider()
		return &TracerProvider{
			provider:         nil,
			tracer:           noopProvider.Tracer("noop"),
			appConfig:        p.AppConfig,
			monitoringConfig: p.MonitoringConfig,
			l:                log,
		}, nil
	}

	tp := &TracerProvider{
		appConfig:        p.AppConfig,
		monitoringConfig: p.MonitoringConfig,
		l:                log,
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
		semconv.ServiceName(tp.monitoringConfig.Tracing.ServiceName),
		semconv.ServiceVersion(tp.appConfig.Version),
		attribute.String("deployment.environment", tp.appConfig.Env),
		attribute.String("service.namespace", "trenova"),
		attribute.String("service.instance.id", generateInstanceID()),
	)

	exporter, err := createExporter(ctx, &tp.monitoringConfig.Tracing, tp.l)
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
		sdktrace.WithIDGenerator(aitrace.NewIDGenerator()),
		sdktrace.WithSampler(createSampler(&tp.monitoringConfig.Tracing)),
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
		zap.String("provider", tp.monitoringConfig.Tracing.Provider),
		zap.String("endpoint", tp.monitoringConfig.Tracing.Endpoint),
		zap.Float64("sampling_rate", tp.monitoringConfig.Tracing.SamplingRate),
		zap.Float64("ai_sampling_rate", tp.monitoringConfig.Tracing.GetAISamplingRate()),
	)

	return nil
}

func createExporter(
	ctx context.Context,
	cfg *monitoringconfig.TracingConfig,
	logger *zap.Logger,
) (sdktrace.SpanExporter, error) {
	switch cfg.Provider {
	case "jaeger":
		logger.Info("Using OTLP exporter for Jaeger (recommended approach)")
		return createOTLPExporter(ctx, cfg)
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
	cfg *monitoringconfig.TracingConfig,
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

func createOTLPExporter(
	ctx context.Context,
	cfg *monitoringconfig.TracingConfig,
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

func createSampler(cfg *monitoringconfig.TracingConfig) sdktrace.Sampler {
	return aitrace.NewSampler(cfg.SamplingRate)
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

	// The clock alone is not enough. Windows resolves time.Now to a coarse
	// tick, so two ids made in the same tick come out identical — and an
	// instance id that can collide is not an instance id. The timestamp stays
	// because it is what makes the value readable in a trace; the entropy is
	// what makes it unique.
	var entropy [4]byte
	if _, rErr := rand.Read(entropy[:]); rErr != nil {
		return fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
	}

	return fmt.Sprintf(
		"%s-%d-%d-%s",
		hostname,
		os.Getpid(),
		time.Now().UnixNano(),
		hex.EncodeToString(entropy[:]),
	)
}
