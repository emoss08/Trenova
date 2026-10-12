package accounting

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
)

type registry struct {
	providers map[integration.Type]accountingsyncports.AccountingProvider
}

func NewRegistry(
	providers ...accountingsyncports.AccountingProvider,
) accountingsyncports.AccountingConnectorRegistry {
	byType := make(map[integration.Type]accountingsyncports.AccountingProvider, len(providers))
	for _, provider := range providers {
		if provider != nil {
			byType[provider.IntegrationType()] = provider
		}
	}
	return &registry{providers: byType}
}

func (r *registry) For(typ integration.Type) (accountingsyncports.AccountingProvider, bool) {
	provider, ok := r.providers[typ]
	return provider, ok
}
