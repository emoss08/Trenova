package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play/playtext"
)

const (
	runSummary     = "play every partner's scenarios against a running Trenova and judge both sides"
	defaultRunFile = ".sim-run"
)

type tradingFlags struct {
	options play.Options
	partner string
	run     uint64
	runFile string
	report  string
}

func (f *tradingFlags) register(set *flag.FlagSet) {
	set.StringVar(&f.options.API, "api", "http://localhost:8080/api/v1", "Trenova's API")
	set.StringVar(&f.options.Sim, "sim", "http://localhost:9210", "the simulator (edi-sim serve)")
	set.StringVar(&f.options.PublicURL, "trenova-public-url", "",
		"the base URL the simulator reaches Trenova's AS2 inbound at (default: --api's host)")
	set.StringVar(&f.options.Email, "email", "admin@trenova.app", "the account to sign in as")
	set.StringVar(&f.options.Password, "password", "admin123!", "its password")
	set.StringVar(&f.options.Root, "partners", "partners", "the folder holding the partners")
	set.StringVar(&f.partner, "partner", "", "only these partners (comma separated, or all)")
	set.Uint64Var(&f.run, "run", 0, fmt.Sprintf(
		"the run (1..%d) that keys every load number; default: the one after the last, "+
			"kept in --run-file", play.MaxRun))
	set.StringVar(&f.runFile, "run-file", defaultRunFile, "where the last run is kept")
	set.StringVar(&f.options.IdentityDir, "identity-dir", ".sim-identity",
		"where Trenova's AS2 keypair for the simulator persists")
	set.StringVar(&f.options.CLI.Binary, "trenova-cli", "",
		"a built Trenova CLI binary, for advance steps "+
			"(go build -o ./build/trenova ./cmd/cli in services/tms)")
	set.StringVar(&f.options.CLI.Dir, "trenova-dir", "../../services/tms",
		"the folder the Trenova CLI runs in (where its config is found)")
	set.BoolVar(&f.options.Verbose, "v", false, "print every step as it passes")
}

func (f *tradingFlags) prepare(stdout io.Writer) error {
	if f.partner != "" {
		f.options.Partners = strings.Split(f.partner, ",")
	}
	if f.options.CLI.Binary != "" {
		binary, err := filepath.Abs(f.options.CLI.Binary)
		if err != nil {
			return err
		}
		f.options.CLI.Binary = binary
	}
	run, err := nextRun(f.run, f.runFile)
	if err != nil {
		return err
	}
	f.options.Run, f.options.Out = run, stdout
	return nil
}

func nextRun(given uint64, path string) (uint64, error) {
	if given > 0 {
		if given > play.MaxRun {
			return 0, fmt.Errorf("--run is 1..%d", play.MaxRun)
		}
		return given, writeRun(path, given)
	}
	last := uint64(0)
	if raw, err := os.ReadFile(path); err == nil {
		last, _ = strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	next := last%play.MaxRun + 1
	return next, writeRun(path, next)
}

func writeRun(path string, run uint64) error {
	if path == "" {
		return nil
	}
	if err := os.WriteFile(path, []byte(strconv.FormatUint(run, 10)+"\n"), 0o600); err != nil {
		return fmt.Errorf("keep the run in %s: %w", path, err)
	}
	return nil
}

func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet(programName+" run", flag.ContinueOnError)
	set.SetOutput(stderr)
	var f tradingFlags
	var scenarios string
	f.register(set)
	set.StringVar(&scenarios, "scenario", "", "only these scenarios (file names, comma separated)")
	set.BoolVar(&f.options.KeepGoing, "keep-going", false,
		"run every scenario and partner after a failure")
	set.Float64Var(&f.options.Scale, "deadline-scale", 1,
		"multiply every wait a scenario states, such as 0.3 for a quick pass")
	set.StringVar(&f.report, "report", "run-report.json", "where to write the JSON report")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if set.NArg() > 0 {
		fmt.Fprintf(stderr, "run: unexpected argument %q\n", set.Arg(0))
		return exitUsage
	}
	if scenarios != "" {
		f.options.Scenarios = strings.Split(scenarios, ",")
	}
	if err := f.prepare(stdout); err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return exitUsage
	}
	f.options.OnScenario = func(_ string, s *play.ScenarioReport) {
		if f.options.Verbose {
			playtext.Scenario(stdout, s)
		}
		playtext.Line(stdout, s)
	}
	report, err := play.Run(ctx, &f.options)
	if report == nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return exitFailed
	}
	fmt.Fprintln(stdout)
	playtext.Summary(stdout, report)
	if f.report != "" {
		if writeErr := report.WriteJSON(f.report); writeErr != nil {
			fmt.Fprintf(stderr, "run: %v\n", writeErr)
			return exitFailed
		}
		fmt.Fprintf(stdout, "report: %s\n", f.report)
	}
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return exitFailed
	}
	if !report.Passed() {
		return exitFailed
	}
	return exitOK
}
