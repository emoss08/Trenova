package analyticsservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/analyticsservice/providers/apikeyprovider"
	"github.com/emoss08/trenova/internal/core/services/analyticsservice/providers/shipmentprovider"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ProvidersParams struct {
	fx.In

	Registry services.AnalyticsRegistry

	// List all providers here with the group tag - they'll be automatically injected
	Providers []services.AnalyticsPageProvider `group:"analytics_providers"`
	Logger    *zap.Logger
}

func RegisterProviders(p ProvidersParams) {
	log := p.Logger.With(zap.String("module", "analytics"))
	for _, provider := range p.Providers {
		page := provider.GetPage()
		log.Info("Registering analytics provider", zap.String("page", string(page)))

		p.Registry.RegisterProvider(provider)
	}
}

var Module = fx.Module("analytics",
	fx.Provide(
		NewRegistry,
		fx.Annotate(
			NewService,
		),
		fx.Annotate(
			shipmentprovider.NewProvider,
			fx.ResultTags(`group:"analytics_providers"`),
			fx.As(new(services.AnalyticsPageProvider)),
		),
		fx.Annotate(
			apikeyprovider.NewProvider,
			fx.ResultTags(`group:"analytics_providers"`),
			fx.As(new(services.AnalyticsPageProvider)),
		),
	),
	fx.Invoke(
		RegisterProviders,
	),
)
