package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/flake"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/overlay"
	"github.com/emoss08/assay/internal/proc"
	"github.com/emoss08/assay/internal/report"
)

func newFlakesCommand(opts *options) *cobra.Command {
	var (
		hunt          bool
		runs          int
		packages      []string
		writeQuar     bool
		perRunTimeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "flakes",
		Short: "Report tests whose verdicts changed while the code did not",
		Long: "flakes reads the evidence assay's own runs produce — watch cycles, mutation\n" +
			"preflights, hunts — and reports every test observed both passing and failing at\n" +
			"the same package fingerprint. A verdict that changed with no code change is the\n" +
			"definition of a flake; a verdict that changed across code changes is just the\n" +
			"code changing, and is never reported.\n\n" +
			"--hunt generates evidence deliberately: it builds each target package's test\n" +
			"binary once and runs it repeatedly at the current tree state.\n\n" +
			"--quarantine records the detected flakes in " + flake.DefaultQuarantinePath + ",\n" +
			"which mutation then refuses to judge mutants by — a flaky covering test marks\n" +
			"mutants killed for reasons that have nothing to do with the mutation.",
		Args: cobra.NoArgs,
		Example: "  assay flakes\n" +
			"  assay flakes --hunt --packages ./internal/... --runs 20\n" +
			"  assay flakes --quarantine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlakes(cmd, opts, flakesRequest{
				hunt:          hunt,
				runs:          runs,
				packages:      packages,
				writeQuar:     writeQuar,
				perRunTimeout: perRunTimeout,
			})
		},
	}

	cmd.Flags().BoolVar(&hunt, "hunt", false,
		"run the target packages' tests repeatedly at the current tree state to generate evidence")
	cmd.Flags().IntVar(&runs, "runs", 10, "repetitions per package when hunting")
	cmd.Flags().StringSliceVar(&packages, "packages", nil, "packages to hunt (required with --hunt)")
	cmd.Flags().BoolVar(&writeQuar, "quarantine", false,
		"record every detected flake in "+flake.DefaultQuarantinePath)
	cmd.Flags().DurationVar(&perRunTimeout, "timeout", 0, "per-run timeout when hunting (default: 2m)")

	return cmd
}

type flakesRequest struct {
	hunt          bool
	runs          int
	packages      []string
	writeQuar     bool
	perRunTimeout time.Duration
}

func runFlakes(cmd *cobra.Command, opts *options, req flakesRequest) error {
	ctx := cmd.Context()

	session, err := opts.open(ctx)
	if err != nil {
		return err
	}

	journal := openJournal(cmd, opts, session.root)
	if journal == nil {
		return fmt.Errorf("flake evidence needs a cache directory; --no-cache is incompatible with flakes")
	}

	if req.hunt {
		if len(req.packages) == 0 {
			return fmt.Errorf("--hunt needs --packages; hunting every package would run the whole suite %d times", req.runs)
		}
		if req.runs < 2 {
			return fmt.Errorf("--runs must be at least 2; one run cannot produce a conflicting verdict")
		}
		if err := huntFlakes(ctx, cmd, opts, session, journal, req); err != nil {
			return err
		}
	}

	observations, err := journal.Load()
	if err != nil {
		return err
	}

	quarantinePath := filepath.Join(session.root, flake.DefaultQuarantinePath)
	quarantine, err := flake.LoadQuarantine(quarantinePath)
	if err != nil {
		return err
	}

	flakes := flake.Detect(observations)
	out := report.NewLines(cmd.OutOrStdout(), opts.useColor())

	if len(flakes) == 0 {
		out.Good("no flaky tests detected across %d recorded verdicts", len(observations))
		if len(observations) == 0 {
			out.Muted("evidence accumulates from watch sessions and mutation preflights; " +
				"or generate it now: assay flakes --hunt --packages <pkg>")
		}

		return nil
	}

	out.Warn("%d flaky tests:", len(flakes))
	for _, f := range flakes {
		marker := ""
		if quarantine.Contains(f.Package, f.Test) {
			marker = "  [quarantined]"
		}
		out.Line("  %s.%s%s", f.Package, f.Test, marker)
		out.Muted("      %d fails / %d passes at identical code · seen via %s · last %s",
			f.Fails(), f.Passes(), strings.Join(flakeSources(f), ", "),
			f.LastSeen().Format("2006-01-02 15:04"))
		if detail := firstDetail(f); detail != "" {
			out.Muted("      %s", detail)
		}
	}

	if req.writeQuar {
		added := 0
		now := time.Now().Unix()
		for _, f := range flakes {
			added += quarantine.Add(flake.QuarantineEntry{
				Package:   f.Package,
				Test:      f.Test,
				Note:      fmt.Sprintf("%d fails / %d passes at identical code", f.Fails(), f.Passes()),
				AddedUnix: now,
			})
		}
		if added > 0 {
			if err := quarantine.Save(quarantinePath); err != nil {
				return err
			}
			out.Line("%d tests quarantined in %s; mutation will no longer judge mutants by them", added, quarantinePath)
		} else {
			out.Muted("every detected flake is already quarantined")
		}

		return nil
	}

	out.Muted("a flaky test that judges mutants inflates the mutation score; " +
		"quarantine with: assay flakes --quarantine")

	return &ExitError{Code: 1}
}

// huntFlakes builds each target package's test binary once and runs it
// repeatedly at the current tree state, recording every top-level verdict.
// Repetition at one fingerprint is what turns suspicion into evidence.
func huntFlakes(
	ctx context.Context,
	cmd *cobra.Command,
	opts *options,
	session *session,
	journal *flake.Journal,
	req flakesRequest,
) error {
	targets, err := resolveIndexTargets(session, req.packages)
	if err != nil {
		return err
	}

	timeout := req.perRunTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	workdir, err := os.MkdirTemp("", "assay-hunt-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workdir)

	fingerprinter := index.NewFingerprinter(session.graph, session.manifest.Digests(), opts.tags)

	cores := runtime.GOMAXPROCS(0)
	jobs := max(1, min(cores/2, len(targets)))
	buildParallelism := max(1, cores/jobs)

	cmd.PrintErrf("hunting %d packages × %d runs at the current tree state\n", len(targets), req.runs)

	hunting := pool.New().
		WithMaxGoroutines(jobs).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for _, target := range targets {
		hunting.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			pkg, ok := session.graph.Package(target)
			if !ok {
				return fmt.Errorf("package %s vanished from the graph", target)
			}

			observations, huntErr := huntPackage(ctx, huntRun{
				root:             session.root,
				importPath:       target,
				dir:              pkg.Dir,
				workdir:          workdir,
				tags:             opts.tags,
				runs:             req.runs,
				timeout:          timeout,
				buildParallelism: buildParallelism,
				fingerprint:      fingerprinter.For(target).String(),
			})
			if huntErr != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				cmd.PrintErrf("  %s: %v\n", target, huntErr)

				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}

			return journal.Record(observations)
		})
	}

	return hunting.Wait()
}

type huntRun struct {
	root             string
	importPath       string
	dir              string
	workdir          string
	tags             []string
	runs             int
	timeout          time.Duration
	buildParallelism int
	fingerprint      string
}

func huntPackage(ctx context.Context, req huntRun) ([]flake.Observation, error) {
	binary := filepath.Join(req.workdir, overlay.UniqueName(req.importPath)+".test")

	args := []string{"test", "-c", "-p", strconv.Itoa(req.buildParallelism), "-o", binary}
	if len(req.tags) > 0 {
		args = append(args, "-tags", strings.Join(req.tags, ","))
	}
	args = append(args, req.importPath)

	if out, buildErr := commandOutput(ctx, req.root, 0, "go", args...); buildErr != nil {
		return nil, fmt.Errorf("compile test binary: %v: %s", buildErr, firstLine(out))
	}
	if _, statErr := os.Stat(binary); statErr != nil {
		// No test files under the current build tags; nothing to observe.
		return nil, nil
	}

	var observations []flake.Observation
	for run := 0; run < req.runs; run++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		out, runErr := commandOutput(ctx, req.dir, req.timeout, binary,
			"-test.v", "-test.count=1", "-test.timeout", (req.timeout * 9 / 10).String())
		if runErr != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}

		passes, fails := flake.ParseVerboseOutput(out)
		if len(passes) == 0 && len(fails) == 0 {
			// The run produced no verdicts — a crash outside any test, or a
			// timeout before the first result. Unattributable, so unrecorded.
			continue
		}

		now := time.Now().Unix()
		for _, name := range passes {
			observations = append(observations, flake.Observation{
				Package: req.importPath, Test: name, Fingerprint: req.fingerprint,
				Verdict: flake.VerdictPass, Source: "hunt", Unix: now,
			})
		}
		for _, name := range fails {
			observations = append(observations, flake.Observation{
				Package: req.importPath, Test: name, Fingerprint: req.fingerprint,
				Verdict: flake.VerdictFail, Source: "hunt", Unix: now,
			})
		}
	}

	return observations, nil
}

func commandOutput(
	ctx context.Context,
	dir string,
	timeout time.Duration,
	name string,
	args ...string,
) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	proc.Isolate(cmd)

	return cmd.CombinedOutput()
}

// openJournal resolves the workspace's flake journal, or nil when caching is
// off or the journal cannot be created — recording is never worth failing the
// primary command over, except where the command *is* the journal.
func openJournal(cmd *cobra.Command, opts *options, root string) *flake.Journal {
	store := opts.store()
	if store == nil {
		return nil
	}

	journal, err := flake.OpenJournal(store.Dir(), root)
	if err != nil {
		cmd.PrintErrf("flake journal unavailable: %v\n", err)

		return nil
	}

	return journal
}

func flakeSources(f flake.Flake) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, e := range f.Evidence {
		for _, source := range e.Sources {
			if _, dup := seen[source]; dup {
				continue
			}
			seen[source] = struct{}{}
			out = append(out, source)
		}
	}

	return out
}

func firstDetail(f flake.Flake) string {
	for _, e := range f.Evidence {
		if e.Detail != "" {
			return e.Detail
		}
	}

	return ""
}

func firstLine(out []byte) string {
	text := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(text, '\n'); idx > 0 {
		return text[:idx]
	}

	return text
}
