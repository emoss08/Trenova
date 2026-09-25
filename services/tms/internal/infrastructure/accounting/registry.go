package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

type registry struct {
	providers map[integration.Type]services.AccountingProvider
}

func NewRegistry(providers ...services.AccountingProvider) services.AccountingConnectorRegistry {
	byType := make(map[integration.Type]services.AccountingProvider, len(providers))
	for _, provider := range providers {
		if provider != nil {
			byType[provider.IntegrationType()] = provider
		}
	}
	return &registry{providers: byType}
}

func (r *registry) For(typ integration.Type) (services.AccountingProvider, bool) {
	provider, ok := r.providers[typ]
	return provider, ok
}
