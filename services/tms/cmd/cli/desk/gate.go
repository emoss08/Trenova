package desk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/spf13/cobra"
)

const (
	defaultGateSpec = "deskbench/gate.yaml"
	gatesDir        = "gates"
	groupSlugLength = 40
)

var gateFlags struct {
	probe      bool
	budget     string
	overBudget bool
	confirm    string
}

var gateCmd = &cobra.Command{
	Use:   "gate [scenario file or directory ...]",
	Short: "Run the gate that decides effort per route and the advisor",
	Long: `Runs the matrix in deskbench/gate.yaml (docs/engineering/desk-bench-gate.md) as one run:
each group's cells on the group's scenarios, 4 trials each, then evaluates the rules written in
the spec before the run. The run directory holds:

  gate.md, gate.json   the decisions and the figures each read
  summary.md           cell figures by split, with no scenario detail
  development/         the development split's report, transcripts and call files
  sealed/              everything else: the full report, the held-out and adversarial
                       splits, and the logs; never handed over

Only development/ and gate.md are handed over.

Three steps, in order:
  trenova desk gate --probe                       # project cost and time; refuses over budget
  trenova desk gate --budget .deskbench/gates/<time>-probe
  trenova desk gate --confirm .deskbench/gates/<time>   # run 2, held-out only

The run stops once it has spent the probe's projected cost times the spec's overrun.`,
	RunE: runGate,
}

func initGate() {
	gateCmd.Flags().StringVar(&flags.spec, "spec", defaultGateSpec, "gate spec")
	gateCmd.Flags().IntVar(&flags.parallel, "parallel", 1, "cases run at once (at most 8)")
	gateCmd.Flags().StringSliceVar(&flags.tags, "tag", nil,
		"only scenarios carrying one of these tags")
	gateCmd.Flags().StringSliceVar(&flags.splits, "split", nil,
		"only scenarios of these splits: development, held-out, adversarial")
	gateCmd.Flags().BoolVar(&gateFlags.probe, "probe", false,
		"run each cell once on a sample of development scenarios and project the run")
	gateCmd.Flags().StringVar(&gateFlags.budget, "budget", "",
		"the probe directory whose projection bounds this run")
	gateCmd.Flags().BoolVar(&gateFlags.overBudget, "over-budget", false,
		"run although the probe projected more than the spec's budget (the owner's call)")
	gateCmd.Flags().StringVar(&gateFlags.confirm, "confirm", "",
		"the gate directory to confirm on the held-out split")
}

func runGate(cmd *cobra.Command, args []string) error {
	scenarios, err := loadScenarioArgs(args)
	if err != nil {
		return err
	}

	switch {
	case gateFlags.probe:
		spec, specErr := deskbench.LoadGateSpec(flags.spec)
		if specErr != nil {
			return specErr
		}
		return runGateProbe(cmd.Context(), spec, scenarios)
	case gateFlags.confirm != "":
		return runGateConfirm(cmd.Context(), scenarios)
	default:
		spec, specErr := deskbench.LoadGateSpec(flags.spec)
		if specErr != nil {
			return specErr
		}
		return runGateMatrix(cmd.Context(), spec, scenarios)
	}
}

type spendCap struct {
	mu     sync.Mutex
	limit  float64
	spent  float64
	cancel context.CancelFunc
	hit    bool
}

func (c *spendCap) progress(result *deskbench.CaseResult) {
	printProgress(result)
	if c == nil || c.limit <= 0 || result.Usage.CostUSD == nil {
		return
	}
	cost, _ := result.Usage.CostUSD.Float64()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.spent += cost
	if c.spent > c.limit && !c.hit {
		c.hit = true
		fmt.Fprintf(os.Stderr, "\nspent $%.2f, over the cap of $%.2f: stopping the run\n",
			c.spent, c.limit)
		c.cancel()
	}
}

type groupRun struct {
	group     *deskbench.GateGroup
	scenarios []*deskbench.Scenario
}

func gateRuns(groups []deskbench.GateGroup, scenarios []*deskbench.Scenario) []groupRun {
	runs := make([]groupRun, 0, len(groups))
	for idx := range groups {
		group := &groups[idx]
		picked := deskbench.PlannedScenarios(group, scenarios)
		if len(picked) == 0 {
			fmt.Fprintf(os.Stderr, "group %s: no scenario matches, skipped\n", group.Name)
			continue
		}
		runs = append(runs, groupRun{group: group, scenarios: picked})
	}

	return runs
}

func runGroups(
	ctx context.Context,
	root, label string,
	repeat, passK int,
	runs []groupRun,
	limit *spendCap,
) ([]*deskbench.CaseResult, error) {
	cases := make([]*deskbench.CaseResult, 0, 256)
	var runErrs []error
	for _, run := range runs {
		dir := deskbench.GroupDir(root, stringutils.SlugifyASCII(run.group.Name, groupSlugLength))
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return cases, fmt.Errorf("create %s: %w", dir, err)
		}
		progress := printProgress
		if limit != nil {
			progress = limit.progress
		}
		report, _, err := execute(ctx, executeOptions{
			kind:    "gate",
			passK:   passK,
			label:   label + " " + run.group.Name,
			outDir:  dir,
			noWrite: true,
		}, deskbench.RunOptions{
			Scenarios: run.scenarios,
			User:      flags.user,
			Agent:     flags.agent,
			Providers: run.group.Cells,
			Repeat:    repeat,
			Parallel:  flags.parallel,
			Keep:      flags.keep,
			Delete:    flags.delete,
			Progress:  progress,
		})
		if report != nil {
			cases = append(cases, report.Cases...)
		}
		if err != nil {
			runErrs = append(runErrs, fmt.Errorf("%s: %w", run.group.Name, err))
			break
		}
		if ctx.Err() != nil {
			break
		}
	}

	return cases, errors.Join(runErrs...)
}

func writeGateRun(root string, meta deskbench.ReportMeta, cases []*deskbench.CaseResult) error {
	return deskbench.WriteGateRun(root, &meta, cases)
}

func gateMeta(label string, passK int, started time.Time) deskbench.ReportMeta {
	return deskbench.ReportMeta{
		StartedAt:  started,
		FinishedAt: time.Now(),
		Namespace:  flags.namespace,
		Version:    cfg.App.Version,
		Commit:     commit(),
		Command:    strings.Join(os.Args[1:], " "),
		PassK:      passK,
		Label:      label,
	}
}

func runGateProbe(
	ctx context.Context,
	spec *deskbench.GateSpec,
	scenarios []*deskbench.Scenario,
) error {
	root, err := gateDirectory("-probe")
	if err != nil {
		return err
	}
	runs := make([]groupRun, 0, len(spec.Groups))
	for idx := range spec.Groups {
		group := &spec.Groups[idx]
		sampled := deskbench.ProbeScenarios(spec, group, scenarios)
		if len(sampled) == 0 {
			fmt.Fprintf(os.Stderr, "group %s: no development scenario to probe\n", group.Name)
			continue
		}
		runs = append(runs, groupRun{group: group, scenarios: sampled})
	}

	started := time.Now()
	cases, runErr := runGroups(ctx, root, "probe", 1, 1, runs, nil)
	if err = writeGateRun(root, gateMeta("gate probe", 1, started), cases); err != nil {
		return errors.Join(runErr, err)
	}
	projection := deskbench.Project(spec, cases, scenarios, flags.parallel)
	if err = projection.Write(root); err != nil {
		return errors.Join(runErr, err)
	}
	fmt.Println()
	fmt.Print(projection.Render())
	fmt.Fprintf(os.Stderr, "\nprobe written to %s\n", root)
	if runErr != nil {
		return runErr
	}
	if !projection.Within {
		return errors.New("the projected run is over budget or unpriced: report it before going on")
	}
	fmt.Fprintf(os.Stderr, "run it with: trenova desk gate --budget %s --parallel %d\n",
		root, flags.parallel)

	return nil
}

func runGateMatrix(
	parent context.Context,
	spec *deskbench.GateSpec,
	scenarios []*deskbench.Scenario,
) error {
	if gateFlags.budget == "" {
		return errors.New("run --probe first and pass its directory with --budget")
	}
	projection, err := deskbench.LoadProjection(gateFlags.budget)
	if err != nil {
		return err
	}
	if !projection.Within && !gateFlags.overBudget {
		return fmt.Errorf("the probe projected $%.2f and %.1f hours, over the budget of $%.2f "+
			"and %.1f hours; the owner decides whether to pass --over-budget",
			projection.Cost, projection.Hours, projection.MaxUSD, projection.MaxHours)
	}

	root, err := gateDirectory("")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	limit := &spendCap{limit: projection.Cap, cancel: cancel}

	started := time.Now()
	cases, runErr := runMatrix(ctx, root, spec, scenarios, limit)
	if err = writeGateRun(root, gateMeta("gate", spec.PassK, started), cases); err != nil {
		return errors.Join(runErr, err)
	}
	if limit.hit || runErr != nil {
		return errors.Join(runErr, fmt.Errorf(
			"the run was cut short ($%.2f of a $%.2f cap spent); no decision is drawn (%s)",
			limit.spent, limit.limit, root))
	}

	result := deskbench.EvaluateGate(spec, cases)
	result.Run = root
	result.Commit = commit()
	result.StartedAt = started
	result.FinishedAt = time.Now()
	if err = result.Write(root); err != nil {
		return errors.Join(runErr, err)
	}
	fmt.Println()
	fmt.Print(result.Render())
	fmt.Fprintf(os.Stderr, "\ngate written to %s; hand over only development/ and gate.md, "+
		"never sealed/\n", root)
	fmt.Fprintf(os.Stderr, "confirm it with: trenova desk gate --confirm %s\n", root)

	return runErr
}

func runMatrix(
	ctx context.Context,
	root string,
	spec *deskbench.GateSpec,
	scenarios []*deskbench.Scenario,
	limit *spendCap,
) ([]*deskbench.CaseResult, error) {
	first := make([]deskbench.GateGroup, 0, len(spec.Groups))
	deferred := make([]deskbench.GateGroup, 0, 1)
	for idx := range spec.Groups {
		if spec.Groups[idx].AdvisorRoutes {
			deferred = append(deferred, spec.Groups[idx])
			continue
		}
		first = append(first, spec.Groups[idx])
	}

	cases, runErr := runGroups(ctx, root, "gate", spec.Repeat, spec.PassK,
		gateRuns(first, scenarios), limit)
	if len(deferred) == 0 || runErr != nil || ctx.Err() != nil {
		return cases, runErr
	}

	routes := deskbench.AdvisorRoutes(spec, cases)
	if len(routes) == 0 {
		fmt.Fprintln(os.Stderr, "the advisor is tried on no route: Haiku is within the gap everywhere")
		return cases, runErr
	}
	fmt.Fprintf(os.Stderr, "trying the advisor on: %v\n", routes)
	for idx := range deferred {
		deferred[idx].Routes = routes
	}
	more, moreErr := runGroups(ctx, root, "gate", spec.Repeat, spec.PassK,
		gateRuns(deferred, scenarios), limit)

	return append(cases, more...), errors.Join(runErr, moreErr)
}

func runGateConfirm(parent context.Context, scenarios []*deskbench.Scenario) error {
	gate, err := deskbench.LoadGateResult(gateFlags.confirm)
	if err != nil {
		return err
	}
	first, err := deskbench.LoadGateRunReport(gate.Run)
	if err != nil {
		return fmt.Errorf("read run 1's report: %w", err)
	}

	root, err := gateDirectory("-confirm")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var limit *spendCap
	if gateFlags.budget != "" {
		projection, loadErr := deskbench.LoadProjection(gateFlags.budget)
		if loadErr != nil {
			return loadErr
		}
		limit = &spendCap{limit: projection.Cap, cancel: cancel}
	}

	started := time.Now()
	cases, runErr := runGroups(ctx, root, "confirm", gate.Spec.Repeat, gate.Spec.PassK,
		gateRuns(gate.ConfirmGroups(), scenarios), limit)
	if err = writeGateRun(root, gateMeta("gate confirm", gate.Spec.PassK, started),
		cases); err != nil {
		return errors.Join(runErr, err)
	}
	if (limit != nil && limit.hit) || runErr != nil {
		return errors.Join(runErr, fmt.Errorf(
			"the confirming run was cut short; nothing is confirmed (%s)", root))
	}

	confirmation := deskbench.ConfirmGate(gate, first.Cases, cases)
	confirmation.Gate = gateFlags.confirm
	confirmation.Run = root
	confirmation.Commit = commit()
	confirmation.StartedAt = started
	confirmation.FinishedAt = time.Now()
	if err = confirmation.Write(root); err != nil {
		return errors.Join(runErr, err)
	}
	fmt.Println()
	fmt.Print(confirmation.Render())
	fmt.Fprintf(os.Stderr, "\nconfirmation written to %s (confirm.md, confirm.json)\n", root)

	return runErr
}

func gateDirectory(suffix string) (string, error) {
	dir := flags.out
	if dir == "" {
		dir = filepath.Join(benchRoot, gatesDir, time.Now().Format(runStamp)+suffix)
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	return dir, nil
}
