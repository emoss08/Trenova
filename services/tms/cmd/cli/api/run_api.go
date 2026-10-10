package api

import (
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
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

func runAPI(_ *cobra.Command, _ []string) error {
	apiapp.Run(bootstrap.Options())
	return nil
}

func init() {
	APICmd.AddCommand(apiRunCmd)
}
