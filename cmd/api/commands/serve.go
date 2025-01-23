package commands

import (
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the API server",
		Long:  `Start the main API server for the Transport Management System.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bootstrap.Bootstrap()
		},
	}
}
