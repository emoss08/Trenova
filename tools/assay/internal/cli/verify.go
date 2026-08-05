package cli

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/runner"
	"github.com/emoss08/assay/internal/selection"
)

func newVerifyCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "verify [-- go test flags]",
		Short: "Prove the narrowing was sound by running what it excluded",
		Long: "verify runs the complement of the narrowed selection: every test that package-level\n" +
			"selection would have run but line-level narrowing dropped. Those tests must all\n" +
			"pass. A failure is proof the narrowing was unsound, and is reported as such.\n\n" +
			"This is a nightly check, not a per-PR one — the complement is the expensive half.",
		Args:    cobra.ArbitraryArgs,
		Example: "  assay verify --since \"$BASE\"",
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

			if !p.narrowed.Enabled {
				cmd.PrintErrln("narrowing was not applied; there is nothing to verify")

				return nil
			}

			groups, packages, tests := complement(p)
			if len(groups) == 0 {
				cmd.PrintErrln("narrowing excluded no tests; nothing to verify")

				return nil
			}

			cmd.PrintErrf("verifying %d excluded tests across %d packages\n", tests, packages)

			root, err := resolveRoot(cmd.Context(), opts.root)
			if err != nil {
				return err
			}

			code, err := runner.Run(cmd.Context(), runner.Options{
				Root:      root,
				Groups:    groups,
				Tags:      opts.tags,
				ExtraArgs: passthrough,
				Stdout:    cmd.OutOrStdout(),
				Stderr:    cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			if code != 0 {
				cmd.PrintErrln()
				cmd.PrintErrln("UNSOUND: a test that narrowing excluded has failed.")
				cmd.PrintErrln("The selection would have skipped a real regression. Do not trust")
				cmd.PrintErrln("narrowing for this change; investigate before relying on it again.")

				return &ExitError{Code: code}
			}

			cmd.PrintErrln("sound: every excluded test passes")

			return nil
		},
	}
}

func complement(p plan) ([]runner.Group, int, int) {
	byPattern := make(map[string][]string)
	packages := 0
	tests := 0

	for _, pkg := range p.narrowed.Plans {
		excluded := pkg.ExcludedTests()
		if len(excluded) == 0 {
			continue
		}
		packages++
		tests += len(excluded)
		pattern := selection.RunPattern(excluded)
		byPattern[pattern] = append(byPattern[pattern], pkg.ImportPath)
	}

	patterns := make([]string, 0, len(byPattern))
	for pattern := range byPattern {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)

	groups := make([]runner.Group, 0, len(patterns))
	for _, pattern := range patterns {
		group := byPattern[pattern]
		sort.Strings(group)
		groups = append(groups, runner.Group{Packages: group, Run: pattern})
	}

	return groups, packages, tests
}
