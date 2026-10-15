package desk

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/spf13/cobra"
)

const (
	defaultRouting    = "deskbench/routing"
	latestRoutingFile = "latest-route"
)

var routeCmd = &cobra.Command{
	Use:   "route [routing file or directory ...]",
	Short: "Score how well questions are matched to the agent with the tools for them",
	Long: `Asks the Desk's agent matcher (myAgentMatch) each labelled question in deskbench/routing,
from the agent it names, and scores where the question would go: to another agent, kept
where it was, to no agent that can help, or nowhere. No turn runs and nothing is written;
it needs Postgres and Redis, the organization's embedding provider for the full ranking, and
Temporal like every bench command.

Each run writes routing.md and routing.json to .deskbench/runs/<time>-route/ and compares
with the previous routing run. Exits non-zero when any question goes the wrong way.

Examples:
  trenova desk route
  trenova desk route deskbench/routing/billing.yaml --name past-due
  trenova desk route --compare ''`,
	RunE: runRouting,
}

func init() {
	routeCmd.Flags().
		StringSliceVar(&flags.names, "name", nil, "only cases whose name contains one of these")
	routeCmd.Flags().StringVar(&flags.compare, "compare", comparePrevious,
		"routing report to compare with: 'previous', a run directory, a routing.json, or '' for none")
}

func runRouting(cmd *cobra.Command, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{defaultRouting}
	}
	cases, err := deskbench.LoadRoutingCases(paths)
	if err != nil {
		return err
	}
	if len(flags.names) > 0 {
		kept := cases[:0]
		for _, routing := range cases {
			if nameMatches(routing.Name, flags.names) {
				kept = append(kept, routing)
			}
		}
		cases = kept
	}
	if len(cases) == 0 {
		return errors.New("no routing case matches the filters")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	outDir, err := runDirectory("route")
	if err != nil {
		return err
	}
	baseline := loadRoutingBaseline(flags.compare)

	bench, err := openBench(ctx, outDir)
	if err != nil {
		return err
	}
	if err = syncBenchAgents(ctx, bench); err != nil {
		_ = bench.Close()
		return err
	}

	fmt.Fprintf(os.Stderr, "desk bench: %d routing case(s), writing to %s\n", len(cases), outDir)
	startedAt := time.Now()
	results, routeErr := bench.Route(ctx, cases, deskbench.RouteOptions{
		User:     flags.user,
		Progress: printRouting,
	})
	if closeErr := bench.Close(); closeErr != nil {
		fmt.Fprintf(os.Stderr, "stopping the bench: %v\n", closeErr)
	}
	if routeErr != nil {
		return fmt.Errorf("the routing run was cut short: %w", routeErr)
	}

	report := deskbench.BuildRoutingReport(commit(), flags.user, startedAt, results)
	if err = report.Write(outDir, baseline); err != nil {
		return err
	}
	if err = os.WriteFile(
		filepath.Join(benchRoot, latestRoutingFile),
		[]byte(outDir+"\n"),
		fileMode,
	); err != nil {
		fmt.Fprintf(os.Stderr, "could not record the latest routing run: %v\n", err)
	}

	fmt.Println()
	fmt.Print(deskbench.RenderRouting(report, baseline))
	fmt.Printf("\n%s\n", filepath.Join(outDir, "routing.md"))

	if failed := report.Total - report.Passed; failed > 0 {
		return fmt.Errorf("%d of %d questions went the wrong way", failed, report.Total)
	}

	return nil
}

func loadRoutingBaseline(compare string) *deskbench.RoutingReport {
	if compare == "" {
		return nil
	}
	path := compare
	if compare == comparePrevious {
		raw, err := os.ReadFile(filepath.Join(benchRoot, latestRoutingFile))
		if err != nil {
			fmt.Fprintln(os.Stderr, "not comparing: no earlier routing run recorded")
			return nil
		}
		path = strings.TrimSpace(string(raw))
	}

	baseline, err := deskbench.LoadRoutingReport(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "not comparing: %v\n", err)
		return nil
	}

	return baseline
}

func printRouting(result *deskbench.RoutingResult) {
	mark := "PASS"
	if !result.Passed {
		mark = "FAIL"
	}
	fmt.Fprintf(os.Stderr, "%s %-40s expected %-28s got %s\n", mark, result.Case, result.Expect,
		result.Got())
}
