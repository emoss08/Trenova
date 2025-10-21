package api

import (
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/spf13/cobra"
)

var APICmd = &cobra.Command{
	Use:   "api",
	Short: "API management commands",
	Long: `API management commands.

Examples:
  trenova api run          # Run the API service`,
}

var apiRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the API service",
	Long: `Run the API service.

Examples:
  trenova api run          # Run the API service`,
	RunE: runAPI,
}

func runAPI(cmd *cobra.Command, args []string) error {
	app := bootstrap.NewApp(
		bootstrap.APIOptions(),
	)
	app.Run()
	return nil
}

func init() {
	APICmd.AddCommand(apiRunCmd)
}
