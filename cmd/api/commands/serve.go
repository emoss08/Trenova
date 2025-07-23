// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
