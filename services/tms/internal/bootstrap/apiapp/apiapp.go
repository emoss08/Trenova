package apiapp

import (
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
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
		e.APIOption(),
	)
}

func Run(base fx.Option) {
	fx.New(base, Options()).Run()
}
