package apiapp

import (
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/schemaversion"
	"github.com/emoss08/trenova/internal/modules/accounting"
	"github.com/emoss08/trenova/internal/modules/agent"
	"github.com/emoss08/trenova/internal/modules/billing"
	"github.com/emoss08/trenova/internal/modules/documents"
	"github.com/emoss08/trenova/internal/modules/edi"
	"github.com/emoss08/trenova/internal/modules/equipment"
	"github.com/emoss08/trenova/internal/modules/identity"
	"github.com/emoss08/trenova/internal/modules/kernel"
	"github.com/emoss08/trenova/internal/modules/operations"
	"github.com/emoss08/trenova/internal/modules/parties"
	"github.com/emoss08/trenova/internal/modules/places"
	"github.com/emoss08/trenova/internal/modules/rating"
	"github.com/emoss08/trenova/internal/modules/reference"
	"github.com/emoss08/trenova/internal/modules/settlement"
	"github.com/emoss08/trenova/internal/modules/workforce"
	"github.com/emoss08/trenova/internal/modules/workspace"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return OptionsFor(edition.Current())
}

func OptionsFor(e *edition.Edition) fx.Option {
	return fx.Options(
		api.HelpersModule,
		api.GraphQLModule,
		api.HandlersModule,
		api.MiddlewareModule,
		api.ServerModule,
		api.PprofServerModule,
		api.MonitoringServerModule,
		api.RouterModule,
		modulesinfra.RealtimeGatewayModule,
		kernel.APIOptions(),
		identity.APIOptions(),
		workspace.APIOptions(),
		reference.APIOptions(),
		places.APIOptions(),
		equipment.APIOptions(),
		parties.APIOptions(),
		workforce.APIOptions(),
		documents.APIOptions(),
		rating.APIOptions(),
		operations.APIOptions(),
		edi.APIOptions(),
		accounting.APIOptions(),
		billing.APIOptions(),
		settlement.APIOptions(),
		agent.APIOptions(),
		e.APIOption(),
	)
}

func Run(base fx.Option) {
	fx.New(schemaversion.Module, base, Options()).Run()
}
