/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package analytics

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/analytics/providers/billingclientprovider"
	"github.com/emoss08/trenova/internal/core/services/analytics/providers/shipmentprovider"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

// ProvidersParams groups all analytics providers for registration
type ProvidersParams struct {
	fx.In

	Registry services.AnalyticsRegistry

	// List all providers here with the group tag - they'll be automatically injected
	Providers []services.AnalyticsPageProvider `group:"analytics_providers"`

	Logger *logger.Logger
}

// RegisterProviders registers all analytics providers with the registry
func RegisterProviders(p ProvidersParams) {
	log := p.Logger.With().
		Str("module", "analytics").
		Logger()
	// Auto-register all providers from the group
	for _, provider := range p.Providers {
		// Get the provider's page to use in logging
		page := provider.GetPage()
		log.Info().Str("page", string(page)).Msg("Registering analytics provider")

		// Register with the registry
		p.Registry.RegisterProvider(provider)
	}
}

// Module is a fx module that sets up the analytics service with providers
var Module = fx.Module("analytics",
	fx.Provide(
		NewRegistry,
		// Register the service as implementing the interface
		fx.Annotate(
			NewService,
			fx.As(new(services.AnalyticsService)),
		),
		// Provide all analytics providers with a group tag
		fx.Annotate(
			shipmentprovider.NewProvider,
			fx.ResultTags(`group:"analytics_providers"`),
			fx.As(new(services.AnalyticsPageProvider)),
		),
		fx.Annotate(
			billingclientprovider.NewProvider,
			fx.ResultTags(`group:"analytics_providers"`),
			fx.As(new(services.AnalyticsPageProvider)),
		),
	),
	fx.Invoke(
		RegisterProviders,
	),
)
