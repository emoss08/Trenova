package cloudcli

import (
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountcli"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportcli"
	"github.com/spf13/cobra"
)

var CloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Trenova Cloud operations run by Trenova operators",
	Long: `Trenova Cloud operations run by Trenova operators.

Examples:
  trenova cloud retire-seed-accounts
  trenova cloud retire-seed-accounts --apply
  trenova cloud staff add jordan@trenova.app --role support --by "Jordan Lee"
  trenova cloud staff remove jordan@trenova.app --by "Jordan Lee"
  trenova cloud staff list`,
}

func init() {
	CloudCmd.AddCommand(seedaccountcli.RetireCmd, supportcli.StaffCmd)
}
