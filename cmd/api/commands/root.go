// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package commands

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trenova",
		Short: "Trenova CLI",
		Long:  `A complete CLI for the Trenova platform with various commands for running services and utilities.`,
	}

	// Add subcommands
	cmd.AddCommand(newServeCmd())

	return cmd
}
