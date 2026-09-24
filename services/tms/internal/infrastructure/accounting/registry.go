package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

type registry struct {
	connectors map[integration.Type]services.AccountingConnector
}

func NewRegistry(connectors ...services.AccountingConnector) services.AccountingConnectorRegistry {
	byType := make(map[integration.Type]services.AccountingConnector, len(connectors))
	for _, connector := range connectors {
		if connector != nil {
			byType[connector.IntegrationType()] = connector
		}
	}
	return &registry{connectors: byType}
}

func (r *registry) For(typ integration.Type) (services.AccountingConnector, bool) {
	connector, ok := r.connectors[typ]
	return connector, ok
}
