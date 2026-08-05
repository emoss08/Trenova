package cli

import (
	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/runner"
)

func newRunCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "run [-- go test flags]",
		Short: "Run only the affected tests",
		Long: "run resolves the affected packages, narrows them to individual tests where that is\n" +
			"provably safe, and hands the result to go test. Everything after -- is forwarded.",
		Args: cobra.ArbitraryArgs,
		Example: "  assay run --since \"$BASE\" -- -count=1 -race\n" +
			"  assay run --tags integration --since HEAD~1",
		RunE: func(cmd *cobra.Command, args []string) error {
			passthrough, err := goTestArgs(cmd, args)
			if err != nil {
				return err
			}

			summary, p, err := resolvePlan(cmd.Context(), opts)
			if err != nil {
				return err
			}

			printer := report.NewPrinter(cmd.ErrOrStderr(), opts.useColor())
			if err := printer.Summary(summary, opts.verbose); err != nil {
				return err
			}

			full, narrowed := p.groups()
			if len(full) == 0 && len(narrowed) == 0 {
				cmd.PrintErrln("no affected tests")

				return nil
			}

			root, err := resolveRoot(cmd.Context(), opts.root)
			if err != nil {
				return err
			}

			code, err := runner.Run(cmd.Context(), runner.Options{
				Root:      root,
				Groups:    runner.GroupByRun(full, narrowed),
				Tags:      opts.tags,
				ExtraArgs: passthrough,
				Stdout:    cmd.OutOrStdout(),
				Stderr:    cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			if code != 0 {
				return &ExitError{Code: code}
			}

			return nil
		},
	}
}
