package workerapp

import (
	"context"

	partiesworker "github.com/emoss08/trenova/internal/modules/parties/jobs"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentqualityjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/conversationschedulejobs"

	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	accountingworker "github.com/emoss08/trenova/internal/modules/accounting/jobs"
	agentworker "github.com/emoss08/trenova/internal/modules/agent/jobs"
	billingworker "github.com/emoss08/trenova/internal/modules/billing/jobs"
	documentsworker "github.com/emoss08/trenova/internal/modules/documents/jobs"
	ediworker "github.com/emoss08/trenova/internal/modules/edi/jobs"
	identityworker "github.com/emoss08/trenova/internal/modules/identity/jobs"
	operationsworker "github.com/emoss08/trenova/internal/modules/operations/jobs"
	placesworker "github.com/emoss08/trenova/internal/modules/places/jobs"
	ratingworker "github.com/emoss08/trenova/internal/modules/rating/jobs"
	referenceworker "github.com/emoss08/trenova/internal/modules/reference/jobs"
	settlementworker "github.com/emoss08/trenova/internal/modules/settlement/jobs"
	workforceworker "github.com/emoss08/trenova/internal/modules/workforce/jobs"
	workspaceworker "github.com/emoss08/trenova/internal/modules/workspace/jobs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//nolint:funlen // the module list is the worker-only application graph
func Modules() fx.Option {
	return fx.Options(
		partiesworker.WorkerOptions(),
		identityworker.WorkerOptions(),
		workspaceworker.WorkerOptions(),
		referenceworker.WorkerOptions(),
		placesworker.WorkerOptions(),
		workforceworker.WorkerOptions(),
		documentsworker.WorkerOptions(),
		ratingworker.WorkerOptions(),
		operationsworker.WorkerOptions(),
		ediworker.WorkerOptions(),
		accountingworker.WorkerOptions(),
		billingworker.WorkerOptions(),
		settlementworker.WorkerOptions(),
		agentworker.WorkerOptions(),
		schedule.Module,
	)
}

func Options() fx.Option {
	return OptionsFor(edition.Current())
}

func OptionsFor(e *edition.Edition) fx.Option {
	return fx.Options(
		temporaljobs.WorkerModule,
		agentjobs.WorkerModule,
		conversationschedulejobs.WorkerModule,
		agentqualityjobs.WorkerModule,
		e.WorkerOption(),
	)
}

func UnscheduledOptions() fx.Option {
	return temporaljobs.UnscheduledWorkerModule
}

type RunParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func Run(base fx.Option, queues []string) {
	opts := []fx.Option{base, Options(), fx.Invoke(logLifecycle)}
	if len(queues) > 0 {
		opts = append(opts, fx.Supply(&registry.QueueFilter{Queues: queues}))
	}

	fx.New(opts...).Run()
}

func logLifecycle(lc fx.Lifecycle, params RunParams) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			params.Logger.Info("Starting worker service",
				zap.String("environment", params.Config.App.Env),
				zap.String("version", params.Config.App.Version),
			)

			params.Logger.Info("Temporal workers initialized and running")

			return nil
		},
		OnStop: func(_ context.Context) error {
			params.Logger.Info("Stopping worker service")
			params.Logger.Info("All workers stopped gracefully")

			return nil
		},
	})
}
