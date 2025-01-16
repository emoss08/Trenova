package commands

import (
	"github.com/spf13/cobra"
	"github.com/trenova-app/transport/internal/bootstrap"
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
