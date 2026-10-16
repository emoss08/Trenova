package docling

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"document-converter",
	fx.Provide(
		fx.Annotate(New, fx.As(new(kernelports.DocumentConverter))),
	),
	fx.Invoke(registerHealthProbe),
)

func registerHealthProbe(
	lc fx.Lifecycle,
	converter kernelports.DocumentConverter,
	logger *zap.Logger,
) {
	l := logger.Named("docling")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !converter.Enabled() {
				l.Warn(
					"no document converter configured; organizations on the Docling extraction engine get no extraction",
					zap.String(
						"remedy",
						"set documentConversion.doclingUrl and documentConversion.apiKey",
					),
				)
				return nil
			}

			if err := converter.Healthy(ctx); err != nil {
				l.Error(
					"document converter is not reachable; Docling extraction will retry until it is",
					zap.Error(err),
				)
				return nil
			}

			l.Info("document converter ready")
			return nil
		},
	})
}
