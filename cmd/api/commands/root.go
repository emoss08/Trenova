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
