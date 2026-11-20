package edi

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func SetConfig(c *config.Config) {
	cfg = c
}

var EDICmd = &cobra.Command{
	Use:   "edi",
	Short: "EDI development commands",
}

var flowsCmd = &cobra.Command{
	Use:   "flows",
	Short: "Work with Lading flow instances in development",
}

func init() {
	EDICmd.AddCommand(flowsCmd)
}
