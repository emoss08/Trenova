package worker

import (
	"context"

	"github.com/emoss08/trenova/internal/bootstrap"
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

var workerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the worker service",
	Long: `Run the worker service.

Examples:
  trenova worker run          # Run the worker service`,
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
	app := bootstrap.NewApp(
		bootstrap.WorkerOptions(),
		fx.Invoke(startWorker),
	)

	app.Run()
	return nil
}

func init() {
	WorkerCmd.AddCommand(workerRunCmd)
}
