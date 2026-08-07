package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/risk"
	"github.com/emoss08/assay/internal/ui"
	"github.com/emoss08/assay/internal/vcs"
)

func newRiskCommand(opts *options) *cobra.Command {
	var (
		windowDays int
		top        int
		packages   []string
		asJSON     bool
	)

	cmd := &cobra.Command{
		Use:   "risk",
		Short: "Rank functions by churn times untested code",
		Long: "risk joins what the coverage index knows with what git knows: a function is\n" +
			"exposed when its code changes often and few tests execute it. Untested-but-frozen\n" +
			"code is a dormant liability and heavily-tested hot code is fine — the product of\n" +
			"churn and uncovered executable lines is where the next regression is most likely\n" +
			"to land unseen.\n\n" +
			"Coverage comes from the index (assay index), unioned across every test package\n" +
			"that can reach each analyzed one. Churn is per file over the trailing window.",
		Args: cobra.NoArgs,
		Example: "  assay risk\n" +
			"  assay risk --window-days 30 --top 10\n" +
			"  assay risk --packages ./internal/... --json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			session, err := opts.open(ctx)
			if err != nil {
				return err
			}

			head, err := vcs.HeadCommit(ctx, session.root)
			if err != nil {
				return err
			}
			loader, err := session.loaderFor(ctx, head, opts.tags)
			if err != nil {
				return err
			}
			if loader == nil {
				return fmt.Errorf("risk needs a coverage index; run assay index first")
			}

			window := time.Duration(windowDays) * 24 * time.Hour
			churn, err := vcs.Churn(ctx, session.root, window)
			if err != nil {
				return err
			}

			var targets []string
			if len(packages) > 0 {
				targets, err = resolveRiskTargets(session, packages)
				if err != nil {
					return err
				}
			}

			analysis, err := risk.Analyze(risk.Options{
				Root:     session.root,
				Commit:   head,
				Graph:    session.graph,
				Loader:   loader,
				Churn:    churn,
				Packages: targets,
			})
			if err != nil {
				return err
			}
			if analysis.Analyzed == 0 {
				return fmt.Errorf("no package has a usable index record at %s; run assay index first",
					vcs.ShortSHA(head))
			}

			if asJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")

				return encoder.Encode(struct {
					WindowDays int `json:"windowDays"`
					risk.Report
				}{windowDays, analysis})
			}

			printRiskReport(cmd, opts, analysis, windowDays, top)

			return nil
		},
	}

	cmd.Flags().IntVar(&windowDays, "window-days", 90, "churn window in days")
	cmd.Flags().IntVar(&top, "top", 20, "entries to show")
	cmd.Flags().StringSliceVar(&packages, "packages", nil, "limit analysis to these import paths")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable output")

	return cmd
}

// resolveRiskTargets expands --packages patterns against every package in the
// graph, not only the testable ones — the untested packages are the ones a
// risk report exists to surface.
func resolveRiskTargets(s *session, requested []string) ([]string, error) {
	return resolvePackagePatterns(s, requested, s.graph.Packages())
}

func printRiskReport(cmd *cobra.Command, opts *options, analysis risk.Report, windowDays, top int) {
	out := report.NewLines(cmd.OutOrStdout(), opts.useColor())

	if len(analysis.Functions) == 0 {
		out.Good("✓ no exposed functions: everything churning in the last %d days is covered", windowDays)
		printRiskFooter(out, analysis)

		return
	}

	paint := out.Painter()
	shown := min(top, len(analysis.Functions))
	out.Line("%s %s", paint.Badge(ui.ToneWarn, "RISK"),
		paint.Warn(fmt.Sprintf("%d exposed functions — churn × uncovered lines, worst first (top %d):",
			len(analysis.Functions), shown)))

	rows := make([][]string, 0, shown)
	for _, fn := range analysis.Functions[:shown] {
		rows = append(rows, []string{
			fmt.Sprintf("%d", fn.Score),
			shortenTail(fn.Package, 32) + "." + fn.Function,
			fmt.Sprintf("%d/%d", fn.UncoveredLines, fn.ExecutableLines),
			fmt.Sprintf("%d", fn.Commits),
			fmt.Sprintf("%s:%d", fn.File, fn.StartLine),
		})
	}
	out.Line("%s", paint.Table(
		[]string{"score", "function", "uncovered", "commits", "location"},
		rows,
	))
	if len(analysis.Functions) > shown {
		out.Muted("… and %d more (--top %d to widen, --json for all)",
			len(analysis.Functions)-shown, len(analysis.Functions))
	}
	printRiskFooter(out, analysis)
}

func printRiskFooter(out report.Lines, analysis risk.Report) {
	if analysis.CoveredChurning > 0 {
		out.Muted("%d churning functions are fully covered — change landing where tests are watching",
			analysis.CoveredChurning)
	}
	if analysis.UncoveredStable > 0 {
		out.Muted("%d functions with uncovered lines sit in files no commit touched — dormant, lower priority",
			analysis.UncoveredStable)
	}
	if analysis.Partial > 0 {
		out.Muted("%d packages analyzed with some covering records missing; their risk may be overstated",
			analysis.Partial)
	}
	if analysis.Unindexed > 0 {
		out.Muted("%d packages have no usable index record and are invisible to this report",
			analysis.Unindexed)
	}
}
