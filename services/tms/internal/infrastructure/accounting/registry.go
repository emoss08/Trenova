package accounting

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
)

type registry struct {
	providers map[integration.Type]accountingconnectorports.AccountingProvider
}

func NewRegistry(
	providers ...accountingconnectorports.AccountingProvider,
) accountingconnectorports.AccountingConnectorRegistry {
	byType := make(map[integration.Type]accountingconnectorports.AccountingProvider, len(providers))
	for _, provider := range providers {
		if provider != nil {
			byType[provider.IntegrationType()] = provider
		}
	}
	return &registry{providers: byType}
}

func (r *registry) For(typ integration.Type) (accountingconnectorports.AccountingProvider, bool) {
	provider, ok := r.providers[typ]
	return provider, ok
}
