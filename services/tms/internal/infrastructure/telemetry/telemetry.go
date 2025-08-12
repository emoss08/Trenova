/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	// * Default timeouts for operations
	defaultShutdownTimeout = 30 * time.Second
	defaultExportTimeout   = 10 * time.Second

	// * gRPC connection parameters
	grpcKeepaliveTime         = 10 * time.Second
	grpcKeepaliveTimeout      = 5 * time.Second
	grpcMaxConnectionAge      = 30 * time.Minute
	grpcMaxConnectionAgeGrace = 5 * time.Minute
)

// Telemetry encapsulates all telemetry components for the application
type Telemetry struct {
	tracerProvider     *sdktrace.TracerProvider
	meterProvider      *sdkmetric.MeterProvider
	prometheusRegistry *prometheus.Registry
	logger             *logger.Logger
	config             *config.Config
	shutdownFns        []func(context.Context) error
}

// Params defines the dependencies for creating a Telemetry instance
type Params struct {
	fx.In

	Logger *logger.Logger
	Config *config.Config
}

// Result defines the output of NewTelemetry for dependency injection
type Result struct {
	fx.Out

	Telemetry *Telemetry
	Metrics   *Metrics `name:"telemetryMetrics" optional:"true"`
}

// NewTelemetry creates and configures a new telemetry instance
func NewTelemetry(p Params) (Result, error) {
	if !p.Config.Telemetry.Enabled {
		p.Logger.Info().Msg("Telemetry is disabled")
		return Result{
			Telemetry: &Telemetry{
				logger:      p.Logger,
				config:      p.Config,
				shutdownFns: make([]func(context.Context) error, 0),
			},
			Metrics: nil,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := buildResource(ctx, p.Config)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create resource: %w", err)
	}

	tracerProvider, shutdownTrace, err := initTraceProvider(ctx, res, p.Config)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create trace provider: %w", err)
	}

	meterProvider, prometheusRegistry, shutdownMetrics, err := initMeterProvider(ctx, res, p.Config)
	if err != nil {
		if shutdownTrace != nil {
			_ = shutdownTrace(ctx)
		}
		return Result{}, fmt.Errorf("failed to create meter provider: %w", err)
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	telemetry := &Telemetry{
		tracerProvider:     tracerProvider,
		meterProvider:      meterProvider,
		prometheusRegistry: prometheusRegistry,
		logger:             p.Logger,
		config:             p.Config,
		shutdownFns: []func(context.Context) error{
			shutdownTrace,
			shutdownMetrics,
		},
	}

	metrics, err := NewMetrics(meterProvider)
	if err != nil {
		_ = telemetry.Shutdown(ctx)
		return Result{}, fmt.Errorf("failed to create metrics: %w", err)
	}

	p.Logger.Info().
		Str("service", p.Config.Telemetry.ServiceName).
		Str("version", p.Config.Telemetry.ServiceVersion).
		Str("environment", p.Config.Telemetry.Environment).
		Str("endpoint", p.Config.Telemetry.OTLP.Endpoint).
		Bool("metrics_enabled", p.Config.Telemetry.MetricsEnabled).
		Bool("tracing_enabled", p.Config.Telemetry.TracingEnabled).
		Msg("Telemetry initialized successfully")

	return Result{
		Telemetry: telemetry,
		Metrics:   metrics,
	}, nil
}

// buildResource creates OpenTelemetry resource with service information
func buildResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.Telemetry.ServiceName),
			semconv.ServiceVersion(cfg.Telemetry.ServiceVersion),
			attribute.String("environment", cfg.Telemetry.Environment),
			attribute.String("deployment.environment", cfg.App.Environment),
		),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithFromEnv(),
	)
}

// initTraceProvider initializes the trace provider with OTLP exporter
func initTraceProvider(
	ctx context.Context,
	res *resource.Resource,
	cfg *config.Config,
) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Telemetry.TracingEnabled {
		tp := sdktrace.NewTracerProvider()
		return tp, tp.Shutdown, nil
	}

	conn, err := createGRPCConnection(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	traceExporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithGRPCConn(conn),
			otlptracegrpc.WithTimeout(defaultExportTimeout),
			otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
				Enabled:         cfg.Telemetry.OTLP.RetryConfig.Enabled,
				InitialInterval: cfg.Telemetry.OTLP.RetryConfig.InitialInterval,
				MaxInterval:     cfg.Telemetry.OTLP.RetryConfig.MaxInterval,
				MaxElapsedTime:  cfg.Telemetry.OTLP.RetryConfig.MaxElapsedTime,
			}),
		),
	)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := configureSampler(cfg)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	shutdown := func(ctx context.Context) error {
		err = tp.Shutdown(ctx)
		connErr := conn.Close()
		if err != nil {
			return err
		}
		return connErr
	}

	return tp, shutdown, nil
}

// initMeterProvider initializes the meter provider with OTLP exporter
func initMeterProvider(
	ctx context.Context,
	res *resource.Resource,
	cfg *config.Config,
) (*sdkmetric.MeterProvider, *prometheus.Registry, func(context.Context) error, error) {
	if !cfg.Telemetry.MetricsEnabled {
		mp := sdkmetric.NewMeterProvider()
		return mp, nil, mp.Shutdown, nil
	}

	var readers []sdkmetric.Reader
	var prometheusRegistry *prometheus.Registry

	conn, err := createGRPCConnection(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(defaultExportTimeout),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         cfg.Telemetry.OTLP.RetryConfig.Enabled,
			InitialInterval: cfg.Telemetry.OTLP.RetryConfig.InitialInterval,
			MaxInterval:     cfg.Telemetry.OTLP.RetryConfig.MaxInterval,
			MaxElapsedTime:  cfg.Telemetry.OTLP.RetryConfig.MaxElapsedTime,
		}),
	)
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	readers = append(readers, sdkmetric.NewPeriodicReader(
		metricExporter,
		sdkmetric.WithInterval(10*time.Second),
	))

	// Prometheus exporter temporarily disabled to avoid dependency mismatch; OTLP exporter remains active.

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithView(
			sdkmetric.NewView(
				sdkmetric.Instrument{
					Name: "http_request_duration_seconds",
				},
				sdkmetric.Stream{
					Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
						Boundaries: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
					},
				},
			),
		),
	}

	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	mp := sdkmetric.NewMeterProvider(opts...)

	shutdown := func(ctx context.Context) error {
		err = mp.Shutdown(ctx)
		connErr := conn.Close()
		if err != nil {
			return err
		}
		return connErr
	}

	return mp, prometheusRegistry, shutdown, nil
}

// createGRPCConnection creates a gRPC connection with proper configuration
func createGRPCConnection(cfg *config.Config) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                grpcKeepaliveTime,
			Timeout:             grpcKeepaliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	if cfg.Telemetry.OTLP.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient(cfg.Telemetry.OTLP.Endpoint, opts...)
}

func configureSampler(cfg *config.Config) sdktrace.Sampler {
	probability := cfg.Telemetry.Sampling.Probability

	if probability < 0 {
		probability = 0
	} else if probability > 1 {
		probability = 1
	}

	if cfg.Telemetry.Sampling.ParentBased {
		return sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(probability),
			sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),
			sdktrace.WithRemoteParentNotSampled(sdktrace.NeverSample()),
			sdktrace.WithLocalParentSampled(sdktrace.AlwaysSample()),
			sdktrace.WithLocalParentNotSampled(sdktrace.TraceIDRatioBased(probability)),
		)
	}

	switch probability {
	case 1:
		return sdktrace.AlwaysSample()
	case 0:
		return sdktrace.NeverSample()
	}

	return sdktrace.TraceIDRatioBased(probability)
}

// Shutdown gracefully shuts down all telemetry components
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil || !t.isEnabled() {
		return nil
	}

	shutdownCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, defaultShutdownTimeout)
		defer cancel()
	}

	t.logger.Info().Msg("Starting telemetry shutdown")

	var errs []error
	for _, fn := range t.shutdownFns {
		if fn != nil {
			if err := fn(shutdownCtx); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"telemetry shutdown encountered %d errors: %w",
			len(errs),
			errors.Join(errs...),
		)
	}

	t.logger.Info().Msg("Telemetry shutdown completed successfully")
	return nil
}

// isEnabled checks if telemetry is enabled
func (t *Telemetry) isEnabled() bool {
	return t != nil && t.config != nil && t.config.Telemetry.Enabled
}

// TracerProvider returns the tracer provider
func (t *Telemetry) TracerProvider() *sdktrace.TracerProvider {
	return t.tracerProvider
}

// MeterProvider returns the meter provider
func (t *Telemetry) MeterProvider() *sdkmetric.MeterProvider {
	return t.meterProvider
}

// PrometheusRegistry returns the prometheus registry if available
func (t *Telemetry) PrometheusRegistry() *prometheus.Registry {
	return t.prometheusRegistry
}
