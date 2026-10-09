package desk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/spf13/cobra"
)

const (
	benchRoot       = ".deskbench"
	runsDir         = "runs"
	latestFile      = "latest"
	liveFile        = "live.log"
	defaultScenario = "deskbench/scenarios"
	comparePrevious = "previous"
	dirMode         = 0o750
	fileMode        = 0o640
	runStamp        = "20060102-150405"
	askNameLength   = 60
	defaultFindings = "deskbench/findings.yaml"
	defaultAgents   = "deskbench/agents.yaml"
	formatMarkdown  = "md"
	formatCSV       = "csv"
	formatJSON      = "json"
	statusAll       = "all"
)

var cfg *config.Config

func SetConfig(c *config.Config) {
	cfg = c
}

var flags struct {
	namespace string
	user      string
	agent     string
	providers []string
	out       string
	keep      bool
	delete    bool
	repeat    int
	parallel  int
	tags      []string
	names     []string
	compare   string
	thread    string
	page      string
	approve   bool
	reject    string
	quiet     bool
	statuses  []string
	severity  []string
	area      string
	format    string
	findings  string
}

var DeskCmd = &cobra.Command{
	Use:   "desk",
	Short: "Drive the Desk's agents through real turns and record everything they did",
	Long: `The desk bench sends messages to the Desk's agents exactly as the Desk does, on this
checkout's code, and records every model request and reply, every tool call with its
arguments, verdict and result, every refusal, retry and regrounding, every proposal with
its preview, and every decision and follow-up.

It hosts its own worker on a Temporal namespace of its own (deskbench), so a dev worker
running older code never answers its turns. It needs Postgres, Redis and Temporal from
docker-compose-local.yml; the API and worker do not need to be running.

Each run writes to .deskbench/runs/<time>/: summary.md, transcript.md, report.json, the
exact request of every model call under calls/, each distinct system prompt under prompts/,
each distinct tool list under tools/, and the worker's own log (worker.log).`,
}

var askCmd = &cobra.Command{
	Use:   "ask <message> [next message ...]",
	Short: "Send one conversation's messages and print everything that happened",
	Long: `Each argument is one message, sent in order on one conversation, the next only after
every turn the last one started has finished.

Examples:
  trenova desk ask "whats running late today"
  trenova desk ask "find the walmart load going to dallas" "push its delivery to friday"
  trenova desk ask --approve "mark shipment S-1042 delivered"
  trenova desk ask --provider "Haiku" --agent "Dispatch Assistant" "who is free tomorrow"
  trenova desk ask --thread athr_... "and the one after that?"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

var runCmd = &cobra.Command{
	Use:   "run [scenario file or directory ...]",
	Short: "Run scenarios and score them against their expectations",
	Long: `Runs every scenario (default: deskbench/scenarios) once per --provider and --repeat,
checks each step against its expectations, and compares the pass rates with the previous
run. Exits non-zero when any case fails.

Examples:
  trenova desk run
  trenova desk run --repeat 3 --provider default --provider "Haiku 5.5"
  trenova desk run --tag dispatch --parallel 2
  trenova desk run deskbench/scenarios/billing.yaml --name invoice
  trenova desk run --compare .deskbench/runs/20261009-101500`,
	RunE: runScenarios,
}

var findingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "List or export the problems the bench has found (deskbench/findings.yaml)",
	Long: `Findings are kept in deskbench/findings.yaml: every problem a bench run turned up, with
its status, severity, area, evidence, cause and fix, so one that is not fixed straight away is
not lost. This command lists them, filtered, as markdown (default), CSV or JSON.

Examples:
  trenova desk findings                                  # every open finding
  trenova desk findings --status all --format md --out findings.md
  trenova desk findings --severity critical,high --format csv
  trenova desk findings --area anthropic --format json`,
	RunE: listFindings,
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete every [bench] conversation the bench left in the Desk",
	RunE:  cleanThreads,
}

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List the chat agents the bench user may talk to",
	RunE:  listAgents,
}

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List the chat models the organization has, by the names --provider takes",
	RunE:  listProviders,
}

func init() {
	persistent := DeskCmd.PersistentFlags()
	persistent.StringVar(&flags.namespace, "namespace", deskbench.DefaultNamespace,
		"Temporal namespace the bench's own worker uses")
	persistent.StringVar(&flags.user, "user", deskbench.DefaultUserEmail, "email of the person the bench acts as")
	persistent.StringVar(&flags.agent, "agent", "",
		"agent name, id or template (default: the general assistant)")
	persistent.StringVar(&flags.out, "out", "", "directory for this run's files (default: .deskbench/runs/<time>)")
	persistent.BoolVar(&flags.keep, "keep", false,
		"leave what the agents proposed waiting for a decision instead of rejecting it at the end")
	persistent.BoolVar(&flags.delete, "delete", false,
		"delete each conversation when its case ends instead of keeping it in the Desk")

	askCmd.Flags().StringSliceVar(&flags.providers, "provider", nil,
		"chat model to pin, by name, model id or provider id (default: the organization's order)")
	askCmd.Flags().StringVar(&flags.thread, "thread", "", "continue an existing conversation")
	askCmd.Flags().StringVar(&flags.page, "page", "",
		`page context as JSON, e.g. '{"path":"/shipments","title":"Shipments"}'`)
	askCmd.Flags().BoolVar(&flags.approve, "approve", false,
		"approve every proposal the last message raised and follow what the agent says about it")
	askCmd.Flags().StringVar(&flags.reject, "reject", "",
		"reject every proposal the last message raised, telling the agent this note")
	askCmd.Flags().BoolVar(&flags.quiet, "quiet", false, "print the summary only, not the transcript")

	runCmd.Flags().StringSliceVar(&flags.providers, "provider", nil,
		"chat models to run against, repeatable; 'default' is the organization's order")
	runCmd.Flags().IntVar(&flags.repeat, "repeat", 1, "runs of each scenario per model")
	runCmd.Flags().IntVar(&flags.parallel, "parallel", 1, "cases run at once (at most 8)")
	runCmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "only scenarios carrying one of these tags")
	runCmd.Flags().StringSliceVar(&flags.names, "name", nil, "only scenarios whose name contains one of these")
	runCmd.Flags().StringVar(&flags.compare, "compare", comparePrevious,
		"report to compare with: 'previous', a run directory, a report.json, or '' for none")

	findingsCmd.Flags().StringVar(&flags.findings, "file", defaultFindings, "findings file")
	findingsCmd.Flags().StringSliceVar(&flags.statuses, "status", []string{string(deskbench.FindingOpen)},
		"open, fixed, wontfix, or all")
	findingsCmd.Flags().StringSliceVar(&flags.severity, "severity", nil, "critical, high, medium, low")
	findingsCmd.Flags().StringVar(&flags.area, "area", "", "only findings whose area contains this")
	findingsCmd.Flags().StringVar(&flags.format, "format", formatMarkdown, "md, csv or json")

	DeskCmd.AddCommand(askCmd, runCmd, findingsCmd, cleanCmd, agentsCmd, providersCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	if flags.approve && flags.reject != "" {
		return errors.New("choose --approve or --reject, not both")
	}
	if len(flags.providers) > 1 {
		return errors.New("ask pins one model; use `desk run` to compare several")
	}

	scenario, err := askScenario(args)
	if err != nil {
		return err
	}

	var threadID pulid.ID
	if flags.thread != "" {
		if threadID, err = pulid.MustParse(flags.thread); err != nil {
			return fmt.Errorf("--thread: %w", err)
		}
	}

	report, outDir, err := execute(cmd.Context(), "ask", deskbench.RunOptions{
		Scenarios: []*deskbench.Scenario{scenario},
		User:      flags.user,
		Agent:     flags.agent,
		Providers: flags.providers,
		Keep:      flags.keep,
		Delete:    flags.delete,
		ThreadID:  threadID,
	}, "")
	if err != nil {
		return err
	}

	if !flags.quiet {
		transcript, readErr := os.ReadFile(filepath.Join(outDir, "transcript.md"))
		if readErr == nil {
			fmt.Println(string(transcript))
		}
	}
	fmt.Print(deskbench.RenderSummary(report))
	printFiles(outDir)

	return nil
}

func askScenario(messages []string) (*deskbench.Scenario, error) {
	var page map[string]any
	if flags.page != "" {
		if err := sonic.UnmarshalString(flags.page, &page); err != nil {
			return nil, fmt.Errorf("--page is not JSON: %w", err)
		}
	}

	steps := make([]deskbench.Step, 0, len(messages)+1)
	for idx, message := range messages {
		step := deskbench.Step{Say: message}
		if idx == 0 {
			step.Page = page
		}
		steps = append(steps, step)
	}
	switch {
	case flags.approve:
		steps = append(steps, deskbench.Step{Decide: deskbench.DecisionApprove})
	case flags.reject != "":
		steps = append(steps, deskbench.Step{Decide: deskbench.DecisionReject, Note: flags.reject})
	}

	scenario := &deskbench.Scenario{
		Name:  "ask " + stringutils.Ellipsize(stringutils.CollapseWhitespace(messages[0]), askNameLength),
		Steps: steps,
	}
	if err := scenario.Validate(); err != nil {
		return nil, err
	}

	return scenario, nil
}

func runScenarios(cmd *cobra.Command, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{defaultScenario}
	}

	scenarios, err := deskbench.LoadScenarios(paths)
	if err != nil {
		return err
	}
	scenarios = filterScenarios(scenarios)
	if len(scenarios) == 0 {
		return errors.New("no scenario matches the filters")
	}

	report, outDir, err := execute(cmd.Context(), "run", deskbench.RunOptions{
		Scenarios: scenarios,
		User:      flags.user,
		Agent:     flags.agent,
		Providers: flags.providers,
		Repeat:    flags.repeat,
		Parallel:  flags.parallel,
		Keep:      flags.keep,
		Delete:    flags.delete,
		Progress:  printProgress,
	}, flags.compare)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Print(deskbench.RenderSummary(report))
	printFiles(outDir)

	if failed := report.Summary.Cases - report.Summary.Passed; failed > 0 {
		return fmt.Errorf("%d of %d cases failed", failed, report.Summary.Cases)
	}

	return nil
}

func filterScenarios(scenarios []*deskbench.Scenario) []*deskbench.Scenario {
	kept := make([]*deskbench.Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if !scenario.HasTag(flags.tags) {
			continue
		}
		if len(flags.names) > 0 && !nameMatches(scenario.Name, flags.names) {
			continue
		}
		kept = append(kept, scenario)
	}

	return kept
}

func nameMatches(name string, wanted []string) bool {
	lowered := strings.ToLower(name)
	for _, want := range wanted {
		if strings.Contains(lowered, strings.ToLower(want)) {
			return true
		}
	}

	return false
}

func execute(
	parent context.Context,
	kind string,
	opts deskbench.RunOptions,
	compare string,
) (*deskbench.Report, string, error) {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	outDir, err := runDirectory(kind)
	if err != nil {
		return nil, "", err
	}

	var baseline *deskbench.Report
	if compare != "" {
		baseline, err = loadBaseline(compare)
		if err != nil {
			fmt.Fprintf(os.Stderr, "not comparing: %v\n", err)
		}
	}

	livePath := filepath.Join(benchRoot, liveFile)
	bench, err := openBench(ctx, outDir, filepath.Join(outDir, liveFile), livePath)
	if err != nil {
		return nil, "", err
	}

	if err = syncBenchAgents(ctx, bench); err != nil {
		_ = bench.Close()
		return nil, "", err
	}

	meta := deskbench.ReportMeta{
		StartedAt: time.Now(),
		Namespace: flags.namespace,
		Version:   cfg.App.Version,
		Commit:    commit(),
		Command:   strings.Join(os.Args[1:], " "),
		WorkerLog: bench.LogPath(),
	}
	fmt.Fprintf(os.Stderr, "desk bench: %d scenario(s), writing to %s\n", len(opts.Scenarios), outDir)
	fmt.Fprintf(os.Stderr, "watch it live: tail -f %s\n", livePath)

	cases, runErr := bench.Run(ctx, opts)
	meta.FinishedAt = time.Now()
	if closeErr := bench.Close(); closeErr != nil {
		fmt.Fprintf(os.Stderr, "stopping the bench: %v\n", closeErr)
	}

	report := deskbench.BuildReport(meta, cases)
	if baseline != nil {
		report.Comparison = deskbench.Compare(baseline, report)
	}
	if err = report.Write(outDir); err != nil {
		return nil, outDir, err
	}
	if kind == "run" {
		if err = markLatest(outDir); err != nil {
			fmt.Fprintf(os.Stderr, "could not record the latest run: %v\n", err)
		}
	}
	if runErr != nil {
		return report, outDir, fmt.Errorf("the run was cut short: %w", runErr)
	}

	return report, outDir, nil
}

func syncBenchAgents(ctx context.Context, bench *deskbench.Bench) error {
	specs, err := deskbench.LoadAgentSpecs(defaultAgents)
	if err != nil || len(specs) == 0 {
		return err
	}

	session, err := bench.SessionFor(ctx, flags.user)
	if err != nil {
		return err
	}

	synced, err := session.SyncAgents(ctx, specs)
	if err != nil {
		return fmt.Errorf("sync the bench agents in %s: %w", defaultAgents, err)
	}
	for _, name := range synced.Created {
		fmt.Fprintf(os.Stderr, "created agent %s\n", name)
	}
	for _, name := range synced.Updated {
		fmt.Fprintf(os.Stderr, "updated agent %s\n", name)
	}

	return nil
}

func openBench(ctx context.Context, outDir string, livePaths ...string) (*deskbench.Bench, error) {
	if cfg == nil {
		return nil, errors.New("configuration was not loaded")
	}
	if err := deskbench.AssertSafe(cfg, flags.namespace); err != nil {
		return nil, err
	}
	if err := deskbench.EnsureNamespace(ctx, &cfg.Temporal, flags.namespace); err != nil {
		return nil, err
	}

	return deskbench.Open(ctx, deskbench.OpenOptions{
		Namespace: flags.namespace,
		OutDir:    outDir,
		LivePaths: livePaths,
	})
}

func runDirectory(kind string) (string, error) {
	dir := flags.out
	if dir == "" {
		dir = filepath.Join(benchRoot, runsDir, time.Now().Format(runStamp)+"-"+kind)
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	return dir, nil
}

func loadBaseline(compare string) (*deskbench.Report, error) {
	path := compare
	if compare == comparePrevious {
		raw, err := os.ReadFile(filepath.Join(benchRoot, latestFile))
		if err != nil {
			return nil, errors.New("no earlier run recorded")
		}
		path = strings.TrimSpace(string(raw))
	}

	return deskbench.LoadReport(path)
}

func markLatest(outDir string) error {
	return os.WriteFile(filepath.Join(benchRoot, latestFile), []byte(outDir+"\n"), fileMode)
}

func commit() string {
	head, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}

	revision := strings.TrimSpace(string(head))
	status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=no").Output()
	if err == nil && len(strings.TrimSpace(string(status))) > 0 {
		revision += " (with uncommitted changes)"
	}

	return revision
}

func printProgress(result *deskbench.CaseResult) {
	mark := "PASS"
	if !result.Passed {
		mark = "FAIL"
	}

	toolCalls, refused := 0, 0
	for _, step := range result.Steps {
		for _, turn := range step.Turns {
			toolCalls += len(turn.Tools)
			for _, tool := range turn.Tools {
				if tool.Refused() {
					refused++
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "%s %s #%d (%s, %d tool calls, %d refused)\n", mark, result.Key(),
		result.Repeat, result.FinishedAt.Sub(result.StartedAt).Round(100*time.Millisecond),
		toolCalls, refused)
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "     error: %s\n", result.Error)
	}
}

func printFiles(outDir string) {
	fmt.Printf("\nFiles: %s/{summary.md,transcript.md,live.log,report.json,calls/,prompts/,tools/,worker.log}\n",
		outDir)
}

func listFindings(_ *cobra.Command, _ []string) error {
	findings, err := deskbench.LoadFindings(flags.findings)
	if err != nil {
		return err
	}

	filter := deskbench.FindingFilter{Area: flags.area}
	if !slices.Contains(flags.statuses, statusAll) {
		for _, status := range flags.statuses {
			filter.Statuses = append(filter.Statuses, deskbench.FindingStatus(status))
		}
	}
	for _, severity := range flags.severity {
		filter.Severities = append(filter.Severities, deskbench.FindingSeverity(severity))
	}
	kept := deskbench.FilterFindings(findings, filter)

	var rendered string
	switch flags.format {
	case formatMarkdown:
		rendered = deskbench.RenderFindingsMarkdown(kept)
	case formatCSV:
		rendered, err = deskbench.RenderFindingsCSV(kept)
	case formatJSON:
		rendered, err = deskbench.RenderFindingsJSON(kept)
	default:
		return fmt.Errorf("--format must be md, csv or json, not %q", flags.format)
	}
	if err != nil {
		return err
	}

	if flags.out == "" {
		fmt.Print(rendered)
		return nil
	}
	if err = os.WriteFile(flags.out, []byte(rendered), fileMode); err != nil {
		return fmt.Errorf("write %s: %w", flags.out, err)
	}
	fmt.Fprintf(os.Stderr, "%d findings written to %s\n", len(kept), flags.out)

	return nil
}

func cleanThreads(cmd *cobra.Command, _ []string) error {
	return withSession(cmd.Context(), func(ctx context.Context, session *deskbench.Session) error {
		deleted, err := session.CleanBenchThreads(ctx)
		fmt.Printf("deleted %d bench conversations\n", deleted)

		return err
	})
}

func listAgents(cmd *cobra.Command, _ []string) error {
	return withSession(cmd.Context(), func(ctx context.Context, session *deskbench.Session) error {
		agents, err := session.Agents(ctx)
		if err != nil {
			return err
		}

		table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(table, "NAME\tTEMPLATE\tID")
		for _, definition := range agents {
			fmt.Fprintf(table, "%s\t%s\t%s\n", definition.Name, definition.Template, definition.ID)
		}

		return table.Flush()
	})
}

func listProviders(cmd *cobra.Command, _ []string) error {
	return withSession(cmd.Context(), func(ctx context.Context, session *deskbench.Session) error {
		providers, err := session.Providers(ctx)
		if err != nil {
			return err
		}

		table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(table, "NAME\tMODEL\tKIND\tPRIORITY\tID")
		for _, provider := range providers {
			fmt.Fprintf(table, "%s\t%s\t%s\t%d\t%s\n", provider.Name, provider.Model, provider.Kind,
				provider.Priority, provider.ID)
		}

		return table.Flush()
	})
}

func withSession(
	parent context.Context,
	fn func(context.Context, *deskbench.Session) error,
) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	outDir, err := os.MkdirTemp("", "deskbench-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)

	bench, err := openBench(ctx, outDir)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := bench.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "stopping the bench: %v\n", closeErr)
		}
	}()

	session, err := bench.SessionFor(ctx, flags.user)
	if err != nil {
		return err
	}

	return fn(ctx, session)
}
