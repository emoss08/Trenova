package cli

import (
	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
)

func newSelectCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select",
		Short: "Show which test packages a change affects",
		Long: "select resolves the affected test packages and prints them without running anything.\n" +
			"Use --json to feed the result into another tool.",
		Args: cobra.NoArgs,
		Example: "  assay select --since origin/main -v\n" +
			"  git diff --name-only origin/main | assay select --files - --json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			summary, _, err := opts.summarize(cmd.Context())
			if err != nil {
				return err
			}

			if opts.asJSON {
				return report.WriteJSON(cmd.OutOrStdout(), summary)
			}

			return report.NewPrinter(cmd.OutOrStdout(), opts.useColor()).Summary(summary, opts.verbose)
		},
	}

	cmd.Flags().BoolVar(&opts.asJSON, "json", false, "emit machine-readable output")

	return cmd
}
