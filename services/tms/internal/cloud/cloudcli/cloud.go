package cloudcli

import (
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountcli"
	"github.com/spf13/cobra"
)

var CloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Trenova Cloud operations run by Trenova operators",
	Long: `Trenova Cloud operations run by Trenova operators.

Examples:
  trenova cloud retire-seed-accounts
  trenova cloud retire-seed-accounts --apply`,
}

func init() {
	CloudCmd.AddCommand(seedaccountcli.RetireCmd)
}
