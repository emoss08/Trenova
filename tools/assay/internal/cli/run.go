package cli

import (
	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/runner"
)

func newRunCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "run [-- go test flags]",
		Short: "Run only the affected test packages",
		Long: "run resolves the affected test packages and hands them to go test.\n" +
			"Everything after -- is forwarded to go test unchanged.",
		Args: cobra.ArbitraryArgs,
		Example: "  assay run --since origin/main -- -count=1 -race\n" +
			"  assay run --tags integration --since HEAD~1",
		RunE: func(cmd *cobra.Command, args []string) error {
			passthrough, err := goTestArgs(cmd, args)
			if err != nil {
				return err
			}

			summary, result, err := opts.summarize(cmd.Context())
			if err != nil {
				return err
			}

			printer := report.NewPrinter(cmd.ErrOrStderr(), opts.useColor())
			if err := printer.Summary(summary, opts.verbose); err != nil {
				return err
			}

			if len(result.Packages) == 0 {
				cmd.PrintErrln("no affected test packages")

				return nil
			}

			root, err := resolveRoot(cmd.Context(), opts.root)
			if err != nil {
				return err
			}

			code, err := runner.Run(cmd.Context(), runner.Options{
				Root:      root,
				Packages:  result.Packages,
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
