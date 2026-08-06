package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/flake"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/mutate"
	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/vcs"
)

type mutateFlags struct {
	baselinePath string
	writeBase    bool
	minMSI       float64
	jobs         int
	packages     []string
	whole        bool
	quiet        bool
	asJSON       bool
	allowFailing bool
	noSchemata   bool
	noHarness    bool
}

func newMutateCommand(opts *options) *cobra.Command {
	flags := &mutateFlags{}

	cmd := &cobra.Command{
		Use:   "mutate",
		Short: "Check whether the tests would notice a bug",
		Long: "mutate changes the code in small, deliberate ways and reports whether any test\n" +
			"objected. A surviving mutant is a gap: the line runs under test, but nothing\n" +
			"asserts on what it does.\n\n" +
			"Only lines the coverage index says a test executes are worth mutating, and only\n" +
			"the covering tests are run for each mutant, so this needs `assay index` first.\n\n" +
			"Scope defaults to the diff. Mutating whole packages takes minutes to hours and\n" +
			"has to be asked for with --whole.",
		Args: cobra.NoArgs,
		Example: "  assay mutate --since \"$BASE\"\n" +
			"  assay mutate --whole --packages ./internal/cover\n" +
			"  assay mutate --whole --write-baseline",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMutate(cmd, opts, flags)
		},
	}

	cmd.Flags().StringVar(&flags.baselinePath, "baseline", mutate.DefaultBaselinePath,
		"accepted-survivor baseline, relative to the workspace root")
	cmd.Flags().BoolVar(&flags.writeBase, "write-baseline", false,
		"record the surviving mutants as accepted instead of failing on them")
	cmd.Flags().Float64Var(&flags.minMSI, "min-msi", 0, "fail if the mutation score is below this percentage")
	cmd.Flags().IntVar(&flags.jobs, "jobs", 0, "mutants to evaluate concurrently (default: half of GOMAXPROCS)")
	cmd.Flags().StringSliceVar(&flags.packages, "packages", nil, "limit mutation to these import paths")
	cmd.Flags().BoolVar(&flags.whole, "whole", false, "mutate entire packages rather than only changed lines")
	cmd.Flags().BoolVar(&flags.quiet, "quiet", false, "suppress progress output")
	cmd.Flags().BoolVar(&flags.asJSON, "json", false, "emit machine-readable output")
	cmd.Flags().BoolVar(&flags.allowFailing, "allow-failing-tests", false,
		"proceed when a covering test already fails, excluding it from every mutant plan")
	cmd.Flags().BoolVar(&flags.noSchemata, "no-schemata", false,
		"compile a separate binary per mutant instead of sharing one per package (much slower)")
	cmd.Flags().BoolVar(&flags.noHarness, "no-harness", false,
		"spawn a process per mutant and test package instead of switching mutants inside one (slower)")

	return cmd
}

func runMutate(cmd *cobra.Command, opts *options, flags *mutateFlags) error {
	ctx := cmd.Context()

	if err := opts.validate(); err != nil {
		return err
	}

	session, err := opts.open(ctx)
	if err != nil {
		return err
	}

	scope, base, err := mutationScope(ctx, opts, flags, session)
	if err != nil {
		return err
	}

	targets, err := mutationTargets(scope, flags, session)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		cmd.PrintErrln("no packages in scope; nothing to mutate")

		return nil
	}

	mutants, err := generateMutants(ctx, opts, session.root, targets, scope)
	if err != nil {
		return err
	}
	if len(mutants) == 0 {
		cmd.PrintErrln("no eligible mutation sites; a change confined to declarations has none")

		return nil
	}

	execOpts := mutate.ExecuteOptions{
		Root:         session.root,
		Tags:         opts.tags,
		Jobs:         flags.jobs,
		Budget:       scope.Budget,
		PackageDir:   packageDirResolver(session),
		PackageOrder: scope.PackageOrder,
		NoSchemata:   flags.noSchemata,
		NoHarness:    flags.noHarness,
	}

	quarantine, err := flake.LoadQuarantine(filepath.Join(session.root, flake.DefaultQuarantinePath))
	if err != nil {
		return err
	}
	if dropped := quarantinedInPlans(mutants, quarantine); dropped > 0 {
		cmd.PrintErrf("excluding %d quarantined flaky tests from every mutant plan (%s)\n",
			dropped, flake.DefaultQuarantinePath)
		mutants = mutate.Exclude(mutants, quarantine.Blocked())
	}

	preflight, err := mutate.RunPreflight(ctx, mutants, execOpts)
	if err != nil {
		return err
	}
	recordPreflight(cmd, opts, session, preflight)
	if !preflight.Clean() {
		if !flags.allowFailing {
			return failingTestsError(preflight)
		}
		cmd.PrintErrf("warning: excluding %d already-failing tests from every mutant plan\n",
			len(preflight.Names()))
		mutants = mutate.Exclude(mutants, preflight.Failing)
	}

	execOpts.Progress = newProgress(cmd.ErrOrStderr(), flags.quiet)

	results, err := mutate.Execute(ctx, mutants, execOpts)
	if err != nil {
		return err
	}
	finishProgress(cmd.ErrOrStderr(), flags.quiet)

	baselinePath := filepath.Join(session.root, flags.baselinePath)
	baseline, err := mutate.LoadBaseline(baselinePath)
	if err != nil {
		return err
	}

	if flags.writeBase {
		return writeBaseline(cmd, ctx, session.root, baselinePath, results)
	}

	score := mutate.Evaluate(results, baseline)

	if flags.asJSON {
		if err := emitMutationJSON(cmd, results, score); err != nil {
			return err
		}
	} else {
		printMutationSummary(cmd, opts, results, score, base)
	}

	return mutationExit(cmd, score, flags.minMSI)
}

// quarantinedInPlans counts the distinct quarantined tests that would
// otherwise judge a mutant, so the exclusion note reports what actually
// changed rather than the size of the quarantine file.
func quarantinedInPlans(mutants []mutate.Mutant, quarantine *flake.Quarantine) int {
	if quarantine.Len() == 0 {
		return 0
	}

	seen := make(map[[2]string]struct{})
	for _, m := range mutants {
		for pkg, tests := range m.Tests() {
			for _, test := range tests {
				if quarantine.Contains(pkg, test) {
					seen[[2]string{pkg, test}] = struct{}{}
				}
			}
		}
	}

	return len(seen)
}

// recordPreflight files every preflight verdict as flake evidence. Preflight
// runs the covering tests against unmutated code, so a verdict here that
// conflicts with one from watch or a hunt at the same fingerprint is exactly
// the false-kill precursor the journal exists to catch.
func recordPreflight(cmd *cobra.Command, opts *options, session *session, pre mutate.Preflight) {
	journal := openJournal(cmd, opts, session.root)
	if journal == nil || len(pre.Ran) == 0 {
		return
	}

	fingerprinter := index.NewFingerprinter(session.graph, session.manifest.Digests(), opts.tags)
	now := time.Now().Unix()

	var observations []flake.Observation
	for pkg, tests := range pre.Ran {
		fp := fingerprinter.For(pkg).String()
		failing := make(map[string]struct{}, len(pre.Failing[pkg]))
		for _, test := range pre.Failing[pkg] {
			failing[test] = struct{}{}
		}
		for _, test := range tests {
			verdict := flake.VerdictPass
			if _, failed := failing[test]; failed {
				verdict = flake.VerdictFail
			}
			observations = append(observations, flake.Observation{
				Package:     pkg,
				Test:        test,
				Fingerprint: fp,
				Verdict:     verdict,
				Source:      "preflight",
				Unix:        now,
			})
		}
	}

	if err := journal.Record(observations); err != nil {
		cmd.PrintErrf("flake journal unavailable: %v\n", err)
	}
}

func mutationScope(
	ctx context.Context,
	opts *options,
	flags *mutateFlags,
	session *session,
) (*mutate.Scope, vcs.Result, error) {
	changes, err := opts.changeSet(ctx, session.root)
	if err != nil {
		return nil, vcs.Result{}, err
	}

	commit := changes.BaseCommit
	if commit == "" {
		commit, err = vcs.HeadCommit(ctx, session.root)
		if err != nil {
			return nil, vcs.Result{}, err
		}
	}

	loader, err := session.loaderFor(ctx, commit, opts.tags)
	if err != nil {
		return nil, vcs.Result{}, err
	}

	scopeChanges := changes.Changes
	if flags.whole {
		scopeChanges = nil
	}

	scope, err := mutate.NewScope(mutate.ScopeOptions{
		Graph:      session.graph,
		Loader:     loader,
		BaseCommit: commit,
		Changes:    scopeChanges,
	})
	if err != nil {
		return nil, vcs.Result{}, err
	}

	return scope, changes, nil
}

func mutationTargets(scope *mutate.Scope, flags *mutateFlags, session *session) ([]string, error) {
	targets := scope.Packages()

	if len(flags.packages) == 0 {
		return targets, nil
	}

	requested, err := resolveIndexTargets(session, flags.packages)
	if err != nil {
		return nil, err
	}

	wanted := make(map[string]struct{}, len(requested))
	for _, pkg := range requested {
		wanted[pkg] = struct{}{}
	}

	var filtered []string
	for _, pkg := range targets {
		if _, ok := wanted[pkg]; ok {
			filtered = append(filtered, pkg)
		}
	}

	return filtered, nil
}

func generateMutants(
	ctx context.Context,
	opts *options,
	root string,
	targets []string,
	scope *mutate.Scope,
) ([]mutate.Mutant, error) {
	var all []mutate.Mutant
	for _, pkg := range targets {
		mutants, err := mutate.Generate(ctx, mutate.GenerateOptions{
			Root:     root,
			Package:  pkg,
			Tags:     opts.tags,
			Eligible: scope.Eligible,
			Tests:    scope.Tests,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, mutants...)
	}

	return all, nil
}

func writeBaseline(
	cmd *cobra.Command,
	ctx context.Context,
	root, path string,
	results []mutate.Result,
) error {
	baseline := mutate.FromResults(results)
	if err := baseline.Save(path); err != nil {
		return err
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}

	if vcs.IsIgnored(ctx, root, path) {
		cmd.PrintErrf(
			"warning: %s is gitignored, so this baseline will not be committed and CI will\n"+
				"         see every survivor as new. Add an exception or pick another path.\n", rel)
	}

	cmd.Printf("wrote %d accepted survivors to %s\n", baseline.Len(), rel)

	return nil
}

func printMutationSummary(
	cmd *cobra.Command,
	opts *options,
	results []mutate.Result,
	score mutate.Score,
	changes vcs.Result,
) {
	out := report.NewLines(cmd.OutOrStdout(), opts.useColor())

	if changes.Note != "" {
		out.Warn("warning: %s", changes.Note)
	}

	out.Line("%d mutants · %d killed · %d survived · %d no-coverage · MSI %.1f%%",
		len(results), score.Killed+score.Timeout, score.Survived, score.NoCoverage, score.MSI())

	// A run dominated by the overlay path is minutes slower than one that shares
	// binaries, so the split is worth saying out loud rather than leaving to guesswork.
	if modes := mutate.Modes(results); modes[mutate.ModeOverlay] > 0 || modes[mutate.ModeSchemata] > 0 {
		out.Muted("%d mutants judged in shared processes, %d in processes of their own, %d on binaries of their own",
			modes[mutate.ModeHarness], modes[mutate.ModeSchemata], modes[mutate.ModeOverlay])
	}

	for _, reason := range fallbackReasons(results, 3) {
		out.Warn("schemata fallback: %s", reason)
	}

	if score.NotBuilt > 0 {
		out.Warn("%d mutants failed to compile; that is an assay defect, not a gap in your tests",
			score.NotBuilt)
	}

	if score.Accepted > 0 {
		out.Muted("%d survivors accepted by the baseline", score.Accepted)
	}

	if len(score.NewSurvivors) == 0 {
		out.Good("no new surviving mutants")

		return
	}

	out.Warn("%d new surviving mutants:", len(score.NewSurvivors))
	for _, r := range score.NewSurvivors {
		out.Line("  %s  %s in %s  %s -> %s  (%d tests ran)",
			r.Mutant.ID, r.Mutant.Location(), functionOrPackage(r), r.Mutant.Original,
			r.Mutant.Replacement, r.Tests)
	}
	out.Muted("a survivor means no test with known coverage of that line objected")
}

// fallbackReasons collects the distinct reasons schemata-capable mutants were
// judged on binaries of their own.
func fallbackReasons(results []mutate.Result, limit int) []string {
	reasons := make([]string, 0, len(results))
	for _, r := range results {
		reasons = append(reasons, r.Fallback)
	}

	return summariseReasons(reasons, limit)
}

func functionOrPackage(r mutate.Result) string {
	if r.Mutant.Function != "" {
		return r.Mutant.Function
	}

	return r.Mutant.Package
}

func emitMutationJSON(cmd *cobra.Command, results []mutate.Result, score mutate.Score) error {
	payload := struct {
		Score   mutate.Score    `json:"score"`
		MSI     float64         `json:"msi"`
		Mutants []mutate.Result `json:"mutants"`
	}{Score: score, MSI: score.MSI(), Mutants: results}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")

	return enc.Encode(payload)
}

func mutationExit(cmd *cobra.Command, score mutate.Score, minMSI float64) error {
	if len(score.NewSurvivors) > 0 {
		return &ExitError{Code: 1}
	}
	if minMSI > 0 && score.Scored() > 0 && score.MSI() < minMSI {
		cmd.PrintErrf("mutation score %.1f%% is below the required %.1f%%\n", score.MSI(), minMSI)

		return &ExitError{Code: 1}
	}

	return nil
}

// failingTestsError refuses to score a red suite. A covering test that already
// fails marks every mutant it judges as killed, so the score would come out
// flattering and meaningless.
func failingTestsError(p mutate.Preflight) error {
	names := p.Names()
	shown := names
	if len(shown) > 8 {
		shown = shown[:8]
	}

	return fmt.Errorf(
		"%d covering tests already fail before any mutation, so every mutant they judge\n"+
			"would be recorded as killed and the score would be inflated. Fix them, or pass\n"+
			"--allow-failing-tests to exclude them from mutant plans.\n  %s",
		len(names), strings.Join(shown, "\n  "))
}
