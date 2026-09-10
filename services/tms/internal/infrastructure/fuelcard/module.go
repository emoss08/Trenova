package fuelcard

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/fuelcard/fileconnector"
	"github.com/emoss08/trenova/internal/infrastructure/fuelcard/rampconnector"
	"go.uber.org/fx"
)

// Module wires the fuel card connectors into the group the integration service
// resolves feeds from. A network with no connector here simply has no feed, which
// is what keeps the marketplace honest about what can actually be turned on.
var Module = fx.Module("fuelcard",
	fx.Provide(
		fx.Annotate(newWEXConnector, fx.ResultTags(`group:"fuelCardConnectors"`)),
		fx.Annotate(newComdataConnector, fx.ResultTags(`group:"fuelCardConnectors"`)),
		fx.Annotate(newRampConnector, fx.ResultTags(`group:"fuelCardConnectors"`)),
	),
)

// WEX and Comdata are read from the transaction export a customer already
// receives, because neither network grants production API access without its own
// partner agreement.
func newWEXConnector() services.FuelCardProvider {
	return fileconnector.New(integration.TypeWEXFuel)
}

func newComdataConnector() services.FuelCardProvider {
	return fileconnector.New(integration.TypeComdataFuel)
}

func newRampConnector() services.FuelCardProvider {
	return rampconnector.New()
}
