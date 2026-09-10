package integrationservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (s *Service) ResolveFuelFeed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider fuelpurchase.CardProvider,
) (*services.ResolvedFuelFeed, error) {
	for integrationType, connector := range s.fuelCardConnectors {
		if !carriesProvider(integrationType, provider) {
			continue
		}

		feed, err := s.resolveFuelFeed(ctx, tenantInfo, integrationType, connector)
		if err != nil {
			return nil, err
		}
		if feed != nil && feed.Provider == provider {
			return feed, nil
		}
	}

	return nil, nil
}

func (s *Service) ListFuelFeeds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*services.ResolvedFuelFeed, error) {
	feeds := make([]*services.ResolvedFuelFeed, 0, len(s.fuelCardConnectors))

	for integrationType, connector := range s.fuelCardConnectors {
		feed, err := s.resolveFuelFeed(ctx, tenantInfo, integrationType, connector)
		if err != nil {
			return nil, err
		}
		if feed != nil {
			feeds = append(feeds, feed)
		}
	}

	return feeds, nil
}

// resolveFuelFeed reports a nil feed when the organization has no enabled, fully
// configured connection of this type. That is an ordinary state for most tenants,
// so it is not an error.
func (s *Service) resolveFuelFeed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
	connector services.FuelCardProvider,
) (*services.ResolvedFuelFeed, error) {
	runtimeCfg, err := s.GetRuntimeConfig(ctx, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	if runtimeCfg == nil || !runtimeCfg.Ready {
		return nil, nil
	}

	return &services.ResolvedFuelFeed{
		Connector:       connector,
		IntegrationType: integrationType,
		Provider:        providerFor(integrationType, runtimeCfg.Config),
		Config:          runtimeCfg.Config,
		DiscoverCards:   discoverCards(runtimeCfg.Config),
	}, nil
}

func carriesProvider(integrationType integration.Type, provider fuelpurchase.CardProvider) bool {
	for _, candidate := range integrationType.CardProviders() {
		if candidate == provider {
			return true
		}
	}

	return false
}

// providerFor picks the card brand a connection carries. A WEX connection may
// hold either WEX or EFS cards, so the brand is configured; every other network
// has exactly one.
func providerFor(
	integrationType integration.Type,
	config map[string]string,
) fuelpurchase.CardProvider {
	brands := integrationType.CardProviders()
	if len(brands) == 0 {
		return fuelpurchase.CardProviderOther
	}

	chosen := fuelpurchase.CardProvider(
		strings.TrimSpace(config[integration.ConfigKeyFuelCardProvider]),
	)
	for _, brand := range brands {
		if brand == chosen {
			return brand
		}
	}

	return brands[0]
}

// discoverCards defaults to true, matching the field's declared default, so a
// connection saved before the option existed still creates cards.
func discoverCards(config map[string]string) bool {
	value, ok := stringutils.ParseBool(
		stringutils.WithDefault(config[integration.ConfigKeyFuelDefaultCardEnd], "true"),
	)

	return !ok || value
}
