/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

// Module provides telemetry infrastructure for the application
var Module = fx.Module("telemetry",
	fx.Provide(
		NewTelemetry,
		// Provide the Prometheus metrics handler
		fx.Annotate(
			NewPrometheusHandler,
			fx.As(new(http.Handler)),
		),
	),
	fx.Invoke(registerTelemetryHooks),
)

// NewPrometheusHandler creates a new Prometheus metrics handler
func NewPrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// LifecycleParams defines dependencies for lifecycle management
type LifecycleParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *logger.Logger
	Config    *config.Config
	Telemetry *Telemetry `optional:"true"`
	Metrics   *Metrics   `optional:"true" name:"telemetryMetrics"`
}

// registerTelemetryHooks sets up lifecycle hooks for telemetry
func registerTelemetryHooks(p LifecycleParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if !p.Config.Telemetry.Enabled {
				p.Logger.Info().Msg("Telemetry is disabled, skipping metrics server")
				return nil
			}

			p.Logger.Info().
				Bool("metrics_enabled", p.Config.Telemetry.MetricsEnabled).
				Int("metrics_port", p.Config.Telemetry.MetricsPort).
				Str("metrics_path", p.Config.Telemetry.MetricsPath).
				Msg("Checking metrics server configuration")

			// Start Prometheus metrics server if metrics are enabled
			if p.Config.Telemetry.MetricsEnabled && p.Config.Telemetry.MetricsPort != 3001 {
				// Get the custom registry if available
				var handler http.Handler
				if p.Telemetry != nil && p.Telemetry.PrometheusRegistry() != nil {
					// Use custom registry to avoid conflicts
					handler = promhttp.HandlerFor(
						p.Telemetry.PrometheusRegistry(),
						promhttp.HandlerOpts{
							ErrorLog:            nil,
							ErrorHandling:       promhttp.ContinueOnError,
							Registry:            p.Telemetry.PrometheusRegistry(),
							DisableCompression:  false,
							MaxRequestsInFlight: 0,
							Timeout:             0,
							EnableOpenMetrics:   true,
						},
					)
				} else {
					// Fallback to default handler
					handler = promhttp.Handler()
				}

				go func() {
					mux := http.NewServeMux()
					mux.Handle(p.Config.Telemetry.MetricsPath, handler)

					addr := fmt.Sprintf(":%d", p.Config.Telemetry.MetricsPort)

					p.Logger.Info().
						Str("addr", addr).
						Str("path", p.Config.Telemetry.MetricsPath).
						Bool("custom_registry", p.Telemetry != nil && p.Telemetry.PrometheusRegistry() != nil).
						Msg("Starting dedicated Prometheus metrics server")

					if err := http.ListenAndServe(addr, mux); err != nil {
						p.Logger.Error().Err(err).Msg("Failed to start metrics server")
					}
				}()
			} else {
				p.Logger.Info().
					Bool("condition_met", p.Config.Telemetry.MetricsEnabled && p.Config.Telemetry.MetricsPort != 3001).
					Msg("Not starting dedicated metrics server")
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if p.Telemetry != nil {
				return p.Telemetry.Shutdown(ctx)
			}
			return nil
		},
	})
}
