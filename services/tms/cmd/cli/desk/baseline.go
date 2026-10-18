package desk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/spf13/cobra"
)

const (
	defaultBaselineSpec = "deskbench/baseline.yaml"
	baselinesDir        = "baselines"
	sweepsDir           = "sweeps"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline [scenario file or directory ...]",
	Short: "Run the Phase 0 baseline matrix and write a report later runs compare against",
	Long: `Runs every scenario (default: deskbench/scenarios, chat and unattended) on every cell
of deskbench/baseline.yaml, each the configured number of trials, every cell pinned to its own
provider so no row mixes models. Writes the full run (report.json, summary.md, transcript.md,
calls/) and, beside it, baseline.md and baseline.json: by model and by scenario, means per
trial with 95% intervals, pass^k, tokens split into uncached input, cache read, cache write and
output, cost per task and per completed task, tool selection and retrieval.

The later rebuild compares against it with:
  trenova desk run --compare .deskbench/baselines/<time>

Examples:
  trenova desk baseline
  trenova desk baseline --spec deskbench/baseline.yaml --parallel 4
  trenova desk baseline --tag unattended`,
	RunE: runBaseline,
}

var sweepCmd = &cobra.Command{
	Use:   "sweep [scenario file or directory ...]",
	Short: "Measure how many held tools a turn can take before disclosure pays",
	Long: `For each held tool count N (default 8, 12, 20 and 30) the scenario's agent is held to N
tools: the ones the scenario needs and the agent's others up to N. Every scenario then runs with
every held tool sent ("all", the disclosure threshold lifted) and disclosed ("disclosed", 8
ranked tools and find_tools), on every cell. Writes one run per point and sweep.md and
sweep.json, from which ai.toolDisclosureThreshold is set.

Examples:
  trenova desk sweep
  trenova desk sweep --held 12,20 --mode all,disclosed --provider claude-haiku-5-5@medium`,
	RunE: runSweep,
}

func initBaseline() {
	baselineCmd.Flags().StringVar(&flags.spec, "spec", defaultBaselineSpec, "baseline spec")
	baselineCmd.Flags().IntVar(&flags.parallel, "parallel", 1, "cases run at once (at most 8)")
	baselineCmd.Flags().StringSliceVar(&flags.tags, "tag", nil,
		"only scenarios carrying one of these tags")
	baselineCmd.Flags().StringSliceVar(&flags.splits, "split", nil,
		"only scenarios of these splits: development, held-out, adversarial")

	sweepCmd.Flags().StringVar(&flags.spec, "spec", defaultBaselineSpec, "baseline spec")
	sweepCmd.Flags().IntSliceVar(&flags.held, "held", nil, "held tool counts (default: the spec's)")
	sweepCmd.Flags().StringSliceVar(&flags.modes, "mode", nil,
		"all, disclosed or both (default: the spec's)")
	sweepCmd.Flags().StringSliceVar(&flags.providers, "provider", nil,
		"cells to sweep (default: the spec's sweep cells)")
	sweepCmd.Flags().IntVar(&flags.parallel, "parallel", 1, "cases run at once (at most 8)")
	sweepCmd.Flags().StringSliceVar(&flags.tags, "tag", nil,
		"only scenarios carrying one of these tags")
	sweepCmd.Flags().StringSliceVar(&flags.splits, "split", nil,
		"only scenarios of these splits: development, held-out, adversarial")
}

func loadScenarioArgs(args []string) ([]*deskbench.Scenario, error) {
	paths := args
	if len(paths) == 0 {
		paths = []string{defaultScenario}
	}

	scenarios, err := deskbench.LoadScenarios(paths)
	if err != nil {
		return nil, err
	}
	scenarios = filterScenarios(scenarios)
	if len(scenarios) == 0 {
		return nil, errors.New("no scenario matches the filters")
	}

	return scenarios, nil
}

func stampedDirectory(kind string) (string, error) {
	dir := flags.out
	if dir == "" {
		dir = filepath.Join(benchRoot, kind, time.Now().Format(runStamp))
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	return dir, nil
}

func runBaseline(cmd *cobra.Command, args []string) error {
	spec, err := deskbench.LoadBaselineSpec(flags.spec)
	if err != nil {
		return err
	}
	scenarios, err := loadScenarioArgs(args)
	if err != nil {
		return err
	}
	outDir, err := stampedDirectory(baselinesDir)
	if err != nil {
		return err
	}

	report, _, err := execute(cmd.Context(), executeOptions{
		kind:   "baseline",
		passK:  spec.PassK,
		label:  "baseline",
		outDir: outDir,
	}, deskbench.RunOptions{
		Scenarios: scenarios,
		User:      flags.user,
		Agent:     flags.agent,
		Providers: spec.Cells,
		Repeat:    spec.Repeat,
		Parallel:  flags.parallel,
		Keep:      flags.keep,
		Delete:    flags.delete,
		Progress:  printProgress,
	})
	if report == nil {
		return err
	}

	baseline := deskbench.BaselineOf(report)
	if writeErr := baseline.Write(outDir); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	fmt.Println()
	fmt.Print(baseline.Render())
	fmt.Fprintf(os.Stderr, "\nbaseline written to %s (baseline.md, baseline.json, report.json)\n",
		outDir)
	fmt.Fprintf(os.Stderr, "compare a later run with: trenova desk run --compare %s\n", outDir)

	return err
}

func sweepModes(spec *deskbench.BaselineSpec) ([]deskbench.SweepMode, error) {
	if len(flags.modes) == 0 {
		return spec.Sweep.Modes, nil
	}

	modes := make([]deskbench.SweepMode, 0, len(flags.modes))
	for _, raw := range flags.modes {
		mode := deskbench.SweepMode(strings.ToLower(strings.TrimSpace(raw)))
		if mode != deskbench.SweepAll && mode != deskbench.SweepDisclosed {
			return nil, fmt.Errorf("--mode is all or disclosed, not %q", raw)
		}
		modes = append(modes, mode)
	}

	return modes, nil
}

func runSweep(cmd *cobra.Command, args []string) error {
	spec, err := deskbench.LoadBaselineSpec(flags.spec)
	if err != nil {
		return err
	}
	scenarios, err := loadScenarioArgs(args)
	if err != nil {
		return err
	}
	modes, err := sweepModes(spec)
	if err != nil {
		return err
	}
	held := flags.held
	if len(held) == 0 {
		held = spec.Sweep.Held
	}
	if len(held) == 0 {
		return errors.New("name the held tool counts with --held or sweep.held in the spec")
	}
	cells := flags.providers
	if len(cells) == 0 {
		cells = spec.Sweep.Cells
	}

	root, err := stampedDirectory(sweepsDir)
	if err != nil {
		return err
	}
	sweep := &deskbench.SweepReport{StartedAt: time.Now(), Commit: commit(), PassK: spec.PassK}

	var runErrs []error
	for _, mode := range modes {
		threshold := mode.Threshold()
		for _, count := range held {
			point := &deskbench.Sweep{Held: count, Mode: mode}
			dir := filepath.Join(root, string(mode)+"-"+strconv.Itoa(count))
			if err = os.MkdirAll(dir, dirMode); err != nil {
				return fmt.Errorf("create %s: %w", dir, err)
			}
			report, _, runErr := execute(cmd.Context(), executeOptions{
				kind:      "sweep",
				passK:     spec.PassK,
				threshold: &threshold,
				label:     "sweep " + point.Label(),
				outDir:    dir,
			}, deskbench.RunOptions{
				Scenarios: scenarios,
				User:      flags.user,
				Agent:     flags.agent,
				Providers: cells,
				Repeat:    spec.Sweep.Repeat,
				Parallel:  flags.parallel,
				Progress:  printProgress,
				Sweep:     point,
			})
			if runErr != nil {
				runErrs = append(runErrs, fmt.Errorf("%s: %w", point.Label(), runErr))
			}
			if report == nil {
				continue
			}
			sweep.Add(&deskbench.SweepPoint{
				Held:      count,
				Mode:      mode,
				Threshold: threshold,
				Run:       dir,
				Cases:     report.Summary.Cases,
				Skipped:   report.Summary.Skipped,
				Cells:     report.Summary.Cells,
			})
			if cmd.Context().Err() != nil {
				break
			}
		}
	}
	sweep.FinishedAt = time.Now()

	if err = sweep.Write(root); err != nil {
		return errors.Join(append(runErrs, err)...)
	}
	fmt.Println()
	fmt.Print(sweep.Render())
	fmt.Fprintf(os.Stderr, "\nsweep written to %s (sweep.md, sweep.json, one run per point)\n", root)

	return errors.Join(runErrs...)
}
