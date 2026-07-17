package worker

import (
	"context"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type WorkerParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

var WorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker management commands",
	Long: `Worker management commands.

Examples:
  trenova worker run          # Run the worker service`,
}

var workerQueues []string

var workerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the worker service",
	Long: `Run the worker service.

Examples:
  trenova worker run                        # Run all task queues
  trenova worker run --queues=report-queue  # Run only the listed task queues`,
	RunE: runWorker,
}

func startWorker(lc fx.Lifecycle, params WorkerParams) {
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

func runWorker(_ *cobra.Command, _ []string) error {
	opts := []fx.Option{
		bootstrap.WorkerOptions(),
		fx.Invoke(startWorker),
	}
	if len(workerQueues) > 0 {
		opts = append(opts, fx.Supply(&registry.QueueFilter{Queues: workerQueues}))
	}

	app := bootstrap.NewApp(opts...)

	app.Run()
	return nil
}

func init() {
	workerRunCmd.Flags().StringSliceVar(
		&workerQueues,
		"queues",
		nil,
		"task queues to run (default: all); overrides temporal.worker.queues config",
	)
	WorkerCmd.AddCommand(workerRunCmd)
}
