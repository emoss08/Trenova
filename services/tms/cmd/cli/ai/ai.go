package ai

import (
	"github.com/spf13/cobra"
)

var AICmd = &cobra.Command{
	Use:   "ai",
	Short: "AI operations run by Trenova operators",
	Long: `AI operations run by Trenova operators.

Examples:
  trenova ai training-export start --from 2026-01-01 --requested-by "Jordan Lee"
  trenova ai training-export list
  trenova ai training-export status aitx_01J...
  trenova ai training-export withdrawn aitx_01J... --output withdrawn.txt`,
}

func init() {
	AICmd.AddCommand(trainingExportCmd)
}
