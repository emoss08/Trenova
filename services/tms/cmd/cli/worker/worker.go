package worker

import (
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/bootstrap/workerapp"
	"github.com/spf13/cobra"
)

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
  trenova worker run                                   # Run all task queues
  trenova worker run --queues=report-queue             # Run only the listed task queues
  trenova worker run --queues=agent-chat-queue         # Interactive assistant turns
  trenova worker run --queues=agent-background-queue   # Scheduled and event-driven agent runs
  trenova worker run --queues=agent-heavy-queue        # Agent run replays

The agent's work is split across queues so one class cannot starve another:
a research run that takes ten minutes must not hold the slot a person's
question is waiting in. Running each on its own deployment is what makes the
split mean anything; a worker given no --queues polls them all.`,
	RunE: runWorker,
}

func runWorker(_ *cobra.Command, _ []string) error {
	workerapp.Run(bootstrap.Options(), workerQueues)
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
