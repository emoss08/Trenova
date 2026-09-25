package temporaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs/connection"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// claimCheckThreshold is the payload size above which a payload leaves workflow
// history for object storage. An agent's growing transcript crosses it within a
// few turns while an ordinary activity result stays inline and readable in the
// UI. The server warns at 256 KiB, so the SDK default would let every long
// conversation sit right at that warning.
const claimCheckThreshold = 64 * 1024

type TemporalClientParams struct {
	fx.In

	Config   *config.Config
	Logger   *zap.Logger
	LC       fx.Lifecycle
	Payloads converter.StorageDriver
	Metrics  *metrics.Registry `optional:"true"`
}

type TemporalClientResult struct {
	fx.Out

	Client client.Client
}

// NewTemporalClient builds the one client the process uses, without dialling.
//
// Connecting lazily is what keeps a Temporal outage at boot from turning
// background work off until the next restart. The client used to be nil when
// the first dial failed, every caller read that as "workflows are disabled",
// and nothing ever checked again. Now the first call connects, a call made
// while Temporal is down fails loudly, and the next one succeeds once it is
// back.
func NewTemporalClient(p TemporalClientParams) (TemporalClientResult, error) {
	log := p.Logger.Named("temporal-client")
	cfg := p.Config.GetTemporalConfig()

	clientOptions, summary, err := connection.BuildOptions(cfg)
	if err != nil {
		return TemporalClientResult{}, fmt.Errorf("configure temporal client: %w", err)
	}

	if cfg.Security.EnableEncryption || cfg.Security.EnableCompression {
		clientOptions.DataConverter = temporaltype.NewEncryptionDataConverter(
			temporaltype.DataConverterOptions{
				EnableEncryption:     cfg.Security.EnableEncryption,
				EncryptionKeyID:      cfg.Security.EncryptionKeyID,
				EnableCompression:    cfg.Security.EnableCompression,
				CompressionThreshold: cfg.Security.CompressionThreshold,
			},
		)

		if cfg.Security.EnableEncryption {
			log.Warn(
				"encryption enabled - ensure TEMPORAL_ENCRYPTION_KEY environment variable is set",
				zap.String("keyID", cfg.Security.EncryptionKeyID),
			)
		}
	}

	// Offloading runs after the data converter, so with encryption on a
	// payload reaches object storage already encrypted.
	clientOptions.ExternalStorage = converter.ExternalStorage{
		Drivers:              []converter.StorageDriver{p.Payloads},
		PayloadSizeThreshold: claimCheckThreshold,
	}

	tracing, err := opentelemetry.NewTracingInterceptor(tracingOptions())
	if err != nil {
		return TemporalClientResult{}, fmt.Errorf("configure temporal tracing: %w", err)
	}
	// The tracing interceptor is also a worker interceptor, and the SDK applies
	// a client's worker interceptors to every worker built from it, so
	// workflows and activities are traced as well as the calls that start them.
	clientOptions.Interceptors = append(
		[]interceptor.ClientInterceptor{tracing},
		clientOptions.Interceptors...,
	)

	if p.Metrics != nil {
		handler, handlerErr := sdkMetricsHandler(p.Metrics, log)
		if handlerErr != nil {
			return TemporalClientResult{}, handlerErr
		}
		clientOptions.MetricsHandler = handler
	}

	c, err := client.NewLazyClient(clientOptions)
	if err != nil {
		return TemporalClientResult{}, fmt.Errorf("create temporal client: %w", err)
	}

	log.Info(
		"temporal client ready; it connects on first use",
		append(
			summary.Fields(),
			zap.String("identity", clientOptions.Identity),
			zap.Bool("encryptionEnabled", cfg.Security.EnableEncryption),
			zap.Bool("compressionEnabled", cfg.Security.EnableCompression),
			zap.Int("claimCheckThresholdBytes", claimCheckThreshold),
		)...,
	)

	p.LC.Append(fx.Hook{
		OnStop: func(context.Context) error {
			log.Info("closing temporal client")
			c.Close()
			return nil
		},
	})

	return TemporalClientResult{Client: c}, nil
}

// tracingOptions keeps the interceptor to starts, workflows and activities.
// Workflow Streams signals the workflow on every publish and polls it with an
// update on every read, so tracing signals, updates and queries turned one
// streamed reply into hundreds of spans that say nothing about the work.
func tracingOptions() opentelemetry.TracerOptions {
	return opentelemetry.TracerOptions{
		DisableSignalTracing: true,
		DisableUpdateTracing: true,
		DisableQueryTracing:  true,
	}
}

// sdkMetricsHandler reports the SDK's own metrics, such as schedule-to-start
// latency, slot usage and sticky cache hits, on the service's /metrics. Those
// are the numbers that say whether a queue has enough workers, which nothing
// the service records about itself can tell.
func sdkMetricsHandler(registry *metrics.Registry, log *zap.Logger) (client.MetricsHandler, error) {
	provider, err := registry.MeterProvider()
	if err != nil {
		return nil, fmt.Errorf("configure temporal metrics: %w", err)
	}

	return opentelemetry.NewMetricsHandler(opentelemetry.MetricsHandlerOptions{
		Meter: provider.Meter("temporal-sdk-go"),
		// The handler panics on a meter error by default. A metric that cannot
		// be recorded is never a reason to take the process down.
		OnError: func(err error) {
			log.Warn("temporal sdk metric could not be recorded", zap.Error(err))
		},
	}), nil
}
