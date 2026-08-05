package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
)

func newSelectCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select",
		Short: "Show which tests a change affects",
		Long: "select resolves the affected packages, narrows them to individual tests where the\n" +
			"coverage index makes that provably safe, and prints the plan without running anything.",
		Args: cobra.NoArgs,
		Example: "  assay select --since \"$BASE\" -v\n" +
			"  git diff --name-only \"$BASE\" | assay select --files - --json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			summary, _, err := resolvePlan(cmd.Context(), opts)
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

func resolvePlan(ctx context.Context, opts *options) (report.Summary, plan, error) {
	session, err := opts.open(ctx)
	if err != nil {
		return report.Summary{}, plan{}, err
	}

	p, err := opts.buildPlan(ctx, session)
	if err != nil {
		return report.Summary{}, plan{}, err
	}

	summary := report.NewSummary(report.Inputs{
		Narrowed:         p.narrowed,
		TotalPackages:    p.summary.totalPackages,
		TestablePackages: p.summary.testablePackages,
		GraphFromCache:   p.summary.graphFromCache,
		Note:             p.summary.note,
		BaseCommit:       p.summary.baseCommit,
	})

	return summary, p, nil
}
