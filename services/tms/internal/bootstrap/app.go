package bootstrap

import (
	"context"

	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules"
	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/core/services/analyticsservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/fiscaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/smsjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

type App struct {
	*fx.App
}

func Options() fx.Option {
	return fx.Options(
		config.Module,
		config.Hooks(),
		infrastructure.ObservabilityModule,
		infrastructure.RedisModule,
		infrastructure.RedisRepositoriesModule,
		infrastructure.DatabaseModule,
		modules.ValidatorModule,
		modules.PostgresRepositoryModule,
		modules.QueryCacheModule,
		fx.Provide(encryptionservice.New),
		fx.Provide(integrationservice.New),
		formula.Module,
		formulatemplateservice.Module,
		temporaljobs.Module,
		schedule.Module,
		auditjobs.Module,
		documentintelligencejobs.Module,
		documentuploadjobs.Module,
		thumbnailjobs.Module,
		smsjobs.Module,
		samsarajobs.Module,
		shipmentjobs.Module,
		fiscaljobs.Module,
		analyticsservice.Module,
	)
}

func NewApp(opts ...fx.Option) *App {
	baseOpts := Options()
	allOpts := append([]fx.Option{baseOpts}, opts...)

	return &App{
		App: fx.New(allOpts...),
	}
}

func APIOptions() fx.Option {
	return fx.Options(
		api.HelpersModule,
		api.HandlersModule,
		api.MiddlewareModule,
		api.ServerModule,
		api.MonitoringServerModule,
		api.RouterModule,
		api.ServiceModule,
		modulesinfra.StorageModule,
		modulesinfra.SMSModule,

		modulesinfra.AblyClientModule,
		modulesinfra.MeilisearchClientModule,
		fx.Invoke(startTCAConsumer),
	)
}

func startTCAConsumer(lc fx.Lifecycle, consumer *tablechangealertservice.Consumer) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start(context.Background())
			return nil
		},
		OnStop: func(context.Context) error {
			consumer.Stop()
			return nil
		},
	})
}

func WorkerOptions() fx.Option {
	return fx.Options(
		modulesinfra.StorageModule,
		modulesinfra.AblyClientModule,

		modulesinfra.MeilisearchClientModule,
		modulesinfra.SMSModule,
		api.ServiceModule,
		temporaljobs.WorkerModule,
	)
}
