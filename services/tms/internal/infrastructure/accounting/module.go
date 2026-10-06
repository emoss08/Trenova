package accounting

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcconnector"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/qboconnector"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/xeroconnector"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/outboundlimit"
	"go.uber.org/fx"
)

var Module = fx.Module("accounting-connectors",
	fx.Provide(
		fx.Annotate(outboundlimit.New, fx.ResultTags(`name:"accountingLimiter"`)),
		qboconnector.New,
		xeroconnector.New,
		bcconnector.New,
		func(
			qbo *qboconnector.Provider,
			xero *xeroconnector.Provider,
			bc *bcconnector.Provider,
		) services.AccountingConnectorRegistry {
			return NewRegistry(qbo, xero, bc)
		},
	),
)
