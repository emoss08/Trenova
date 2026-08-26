package gotenberg

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module wires the renderer and probes it once on start.
//
// A failed probe logs and continues rather than aborting startup: the API and
// the worker both boot without a renderer today, and refusing to start would
// take down shipment dispatch because nobody could print an invoice.
var Module = fx.Module(
	"pdf-renderer",
	fx.Provide(
		fx.Annotate(New, fx.As(new(services.PDFRenderer))),
	),
	fx.Invoke(registerHealthProbe),
)

func registerHealthProbe(lc fx.Lifecycle, renderer services.PDFRenderer, logger *zap.Logger) {
	l := logger.Named("pdfrender.gotenberg")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !renderer.Enabled() {
				l.Error(
					"no PDF renderer configured; document generation will be unavailable",
					zap.String("remedy", "set renderer.gotenbergUrl"),
				)
				return nil
			}

			if err := renderer.Healthy(ctx); err != nil {
				l.Error(
					"PDF renderer is not reachable; document generation will fail until it is",
					zap.Error(err),
				)
				return nil
			}

			l.Info("PDF renderer ready")
			return nil
		},
	})
}
