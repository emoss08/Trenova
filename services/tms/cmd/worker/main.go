package main

import (
	"fmt"
	"os"

	"github.com/emoss08/trenova/internal/bootstrap/core"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/bootstrap/workerapp"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var queues []string

var rootCmd = &cobra.Command{
	Use:   "trenova-worker",
	Short: "Run the Trenova worker service",
	RunE: func(_ *cobra.Command, _ []string) error {
		e := edition.Current()
		workerapp.Run(fx.Options(core.OptionsFor(e), workerapp.Modules()), queues)
		return nil
	},
}

func main() {
	rootCmd.Flags().StringSliceVar(
		&queues,
		"queues",
		nil,
		"task queues to run (default: all); overrides temporal.worker.queues config",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
