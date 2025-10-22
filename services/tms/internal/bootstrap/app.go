package bootstrap

import (
	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	"github.com/emoss08/trenova/internal/bootstrap/modules/cdc"
	"github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules/permission"
	"github.com/emoss08/trenova/internal/bootstrap/modules/querycache"
	"github.com/emoss08/trenova/internal/bootstrap/modules/seqgen"
	"github.com/emoss08/trenova/internal/bootstrap/modules/statemanager"
	"github.com/emoss08/trenova/internal/bootstrap/modules/streaming"
	"github.com/emoss08/trenova/internal/bootstrap/modules/validators"
	"github.com/emoss08/trenova/internal/bootstrap/modules/worker"
	"github.com/emoss08/trenova/internal/core/services"
	"github.com/emoss08/trenova/internal/core/services/email"
	"github.com/emoss08/trenova/internal/infrastructure/ai"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/routing"
	"github.com/emoss08/trenova/pkg/formula"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type App struct {
	*fx.App
}

func Options() fx.Option {
	return fx.Options(
		config.Module,
		config.Hooks(),
		AddLogger(),
		infrastructure.ObservabilityModule,
		infrastructure.DatabaseModule,
		infrastructure.FileStorageModule,
		infrastructure.SearchModule,
		routing.Module,
		ai.Module,
		infrastructure.RedisModule,
		infrastructure.RedisRepositoriesModule,
		infrastructure.PostgresRepositoriesModule,
		infrastructure.TemporalClientModule,
		infrastructure.CalculatorsModule,
		cdc.Module,
		streaming.Module,
		formula.Module,
		statemanager.Module,
		seqgen.Module,
		validators.Module,
		email.Module,
		services.Module,
		querycache.Module, // Warm field caches at startup
		permission.Options(),
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
		api.HandlerModule,
		api.MiddlewareModule,
		api.ServerModule,
		api.RouterModule,
	)
}

func WorkerOptions() fx.Option {
	return fx.Options(
		worker.Module,
	)
}

func CLIOptions() fx.Option {
	return fx.Options(
		infrastructure.RedisModule,
	)
}

func AddLogger() fx.Option {
	return fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: log}
	})
}
