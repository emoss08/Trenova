package benchrun

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchaudit"
	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/deskbench/benchcheck"
	"github.com/emoss08/trenova/internal/deskbench/benchlive"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchsql"
	"github.com/emoss08/trenova/internal/deskbench/benchwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxParallel = 8
)

type Runner struct {
	bench      *deskbench.Bench
	checker    *benchcheck.Checker
	auditor    *benchaudit.Auditor
	database   *benchsql.Database
	live       *benchlive.Feed
	recorder   *benchrecord.Recorder
	watcher    *benchwatch.TurnWatcher
	runs       agentruntimeports.AgentRunService
	runRecords agentruntimeports.AgentRunRepository
	proposals  agentproposalports.AgentProposalRepository
	previews   agentruntimeports.ProposalPreviewService
	decisions  agentdecisionports.AgentDecisionService
	shipments  shipmentserviceports.ShipmentLifecycleService
	exclusive  func(key string) func()
	measure    func(
		ctx context.Context,
		session *deskbench.Session,
		record *benchrecord.TurnRecord,
		from time.Time,
	)
	facts  benchsql.FactSource
	tidied sync.Once
}

func New(bench *deskbench.Bench) *Runner {
	database := benchsql.New(bench.DB)

	return &Runner{
		bench:      bench,
		checker:    benchcheck.New(database.FactValues),
		auditor:    benchaudit.New(bench.DB, bench.ActionTools, bench.Live()),
		database:   database,
		live:       bench.Live(),
		recorder:   bench.Recorder(),
		watcher:    bench.Watcher(),
		runs:       bench.Runs,
		runRecords: bench.RunRecords,
		proposals:  bench.Proposals,
		previews:   bench.Previews,
		decisions:  bench.Decisions,
		shipments:  bench.Shipments,
		exclusive:  bench.Exclusive,
		measure:    bench.Measure,
		facts:      database.FactValues,
	}
}

type RunOptions struct {
	Scenarios []*benchscenario.Scenario
	User      string
	Agent     string
	Providers []string
	Repeat    int
	Parallel  int
	Keep      bool
	Delete    bool
	ThreadID  pulid.ID
	Progress  func(*benchresult.CaseResult)
	Sweep     *benchscenario.Sweep
}

type plannedCase struct {
	scenario      *benchscenario.Scenario
	provider      string
	repeat        int
	scenarioOrder int
	providerOrder int
}

func (p plannedCase) label() string {
	provider := p.provider
	if provider == "" {
		provider = benchcell.DefaultProviderLabel
	}

	return p.scenario.Name + " @ " + provider + " #" + strconv.Itoa(p.repeat)
}

func (r *Runner) Run(ctx context.Context, opts *RunOptions) ([]*benchresult.CaseResult, error) {
	repeat := max(opts.Repeat, 1)
	parallel := min(max(opts.Parallel, 1), maxParallel)
	providers := opts.Providers
	if len(providers) == 0 {
		providers = []string{""}
	}

	planned := make([]plannedCase, 0, len(opts.Scenarios)*len(providers)*repeat)
	for scenarioOrder, scenario := range opts.Scenarios {
		for providerOrder, provider := range providers {
			chosen := provider
			if chosen == "" {
				chosen = scenario.Provider
			}
			for rep := 1; rep <= repeat; rep++ {
				planned = append(planned, plannedCase{
					scenario:      scenario,
					provider:      chosen,
					repeat:        rep,
					scenarioOrder: scenarioOrder,
					providerOrder: providerOrder,
				})
			}
		}
	}

	results := make([]*benchresult.CaseResult, len(planned))
	var progressMu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, parallel)
	for _, unit := range runUnits(planned) {
		if ctx.Err() != nil {
			break
		}
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			for _, idx := range unit {
				if ctx.Err() != nil {
					return
				}
				result := r.runCase(ctx, opts, planned[idx])
				results[idx] = result
				if opts.Progress != nil {
					progressMu.Lock()
					opts.Progress(result)
					progressMu.Unlock()
				}
			}
		})
	}
	wg.Wait()

	finished := make([]*benchresult.CaseResult, 0, len(results))
	for _, result := range results {
		if result != nil {
			finished = append(finished, result)
		}
	}

	return finished, ctx.Err()
}

func runUnits(planned []plannedCase) [][]int {
	units := make([][]int, 0, len(planned))
	chains := make(map[string]int)
	for idx := range planned {
		key := strings.ToLower(strings.TrimSpace(planned[idx].scenario.Exclusive))
		if key == "" {
			units = append(units, []int{idx})
			continue
		}
		if at, ok := chains[key]; ok {
			units[at] = append(units[at], idx)
			continue
		}
		chains[key] = len(units)
		units = append(units, []int{idx})
	}
	for _, unit := range units {
		slices.SortStableFunc(unit, func(a, b int) int {
			return cmp.Or(
				cmp.Compare(planned[a].providerOrder, planned[b].providerOrder),
				cmp.Compare(planned[a].repeat, planned[b].repeat),
				cmp.Compare(planned[a].scenarioOrder, planned[b].scenarioOrder),
			)
		})
	}

	return units
}

func (r *Runner) runCase(
	ctx context.Context,
	opts *RunOptions,
	planned plannedCase,
) *benchresult.CaseResult {
	scenario := planned.scenario
	if key := exclusiveKey(scenario); key != "" {
		unlock := r.exclusive(key)
		defer unlock()
	}
	result := &benchresult.CaseResult{
		Scenario:  scenario,
		Provider:  benchcell.DefaultProviderLabel,
		Cell:      planned.provider,
		Repeat:    planned.repeat,
		StartedAt: time.Now(),
	}
	r.live.Linef(planned.label(), ">>> start: %s", benchrecord.FirstLine(scenario.Description))
	var session *deskbench.Session
	defer func() {
		result.FinishedAt = time.Now()
		result.Usage = caseUsage(result)
		result.Passed = result.Error == "" && stepsPassed(result.Steps)
		auditCtx, auditCancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			benchaudit.AuditTimeout,
		)
		r.auditor.Record(auditCtx, session, result, planned.label())
		auditCancel()
		result.Metrics = benchresult.MeasureCase(result)
		r.liveVerdict(planned.label(), result)
	}()

	session, cell, ready := r.prepareCase(ctx, opts, planned, result)
	if !ready {
		return result
	}

	if scenario.Unattended() {
		if opts.Sweep != nil {
			result.Skipped = "the tool-count sweep runs chat scenarios"
			return result
		}
		result.Steps = append(result.Steps, r.runFireStep(ctx, session, cell, &scenario.Steps[0],
			planned.label()))
		return result
	}

	r.runConversation(ctx, &conversationCase{
		opts:    opts,
		planned: planned,
		session: session,
		cell:    cell,
		result:  result,
	})

	return result
}

func (r *Runner) prepareCase(
	ctx context.Context,
	opts *RunOptions,
	planned plannedCase,
	result *benchresult.CaseResult,
) (session *deskbench.Session, cell benchcell.Cell, ready bool) {
	scenario := planned.scenario
	user := scenario.User
	if user == "" {
		user = opts.User
	}
	session, err := r.bench.SessionFor(ctx, user)
	if err != nil {
		result.Error = err.Error()
		return session, cell, false
	}

	cell, err = session.Cell(ctx, planned.provider)
	if err != nil {
		result.Error = err.Error()
		return session, cell, false
	}
	provider := cell.Provider
	result.Provider = cell.Label()
	result.Model = providerModel(provider)
	if provider != nil {
		result.ProviderKind = string(provider.Kind)
	}
	result.Effort = string(cell.Effort)

	unmet, err := r.database.UnmetRequirements(ctx, session, scenario.Requires)
	if err != nil {
		result.Error = err.Error()
		return session, cell, false
	}
	if unmet != "" {
		result.Skipped = unmet
		return session, cell, false
	}

	r.tidied.Do(func() { r.tidyLeftovers(ctx, session) })
	if err = r.database.RunSetup(ctx, session, scenario.Setup); err != nil {
		result.Error = err.Error()
		return session, cell, false
	}
	if len(scenario.Setup) > 0 {
		r.live.Linef(
			planned.label(),
			"setup: %d statement(s) put the data in place",
			len(scenario.Setup),
		)
	}

	return session, cell, true
}

func (r *Runner) tidyLeftovers(ctx context.Context, session *deskbench.Session) {
	if tidyErr := r.database.RunStatements(ctx, session, benchsql.Leftovers()); tidyErr != nil {
		r.live.Linef("bench", "could not cancel what earlier bench runs left open: %v", tidyErr)
	}
	marked, delayErr := r.bringStatusesCurrent(ctx, session)
	if delayErr != nil {
		r.live.Linef("bench", "could not bring shipment statuses current: %v", delayErr)
	} else if marked > 0 {
		r.live.Linef(
			"bench",
			"marked %d late shipment(s) Delayed, as the auto-delay schedule would",
			marked,
		)
	}
}

type conversationCase struct {
	opts    *RunOptions
	planned plannedCase
	session *deskbench.Session
	cell    benchcell.Cell
	result  *benchresult.CaseResult
}

func (r *Runner) runConversation(ctx context.Context, c *conversationCase) {
	scenario := c.planned.scenario
	result := c.result
	swept, release, err := r.sweptAgent(ctx, c.session, c.opts, scenario)
	if release != nil {
		defer release()
	}
	if err != nil {
		if errors.Is(err, benchscenario.ErrSweepSkipped) {
			result.Skipped = err.Error()
		} else {
			result.Error = err.Error()
		}
		return
	}

	conv, err := r.openCase(ctx, c.session, c.opts, c.planned, c.cell.Provider, swept)
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.ThreadID = conv.Thread.ID.String()
	conv.NeedsText = slices.Contains(
		scenario.Requires,
		benchscenario.Requirement{DocumentText: true},
	)
	unpin := r.recorder.Pin(
		conv.Thread.ID,
		benchrecord.ProviderPin{Effort: c.cell.Effort, Advisor: c.cell.Advisor},
	)
	defer unpin()
	if conv.Agent != nil {
		result.Agent = conv.Agent.Name
	}
	closing := deskbench.CloseOptions{
		LeavePending: c.opts.Keep || c.opts.ThreadID.IsNotNil(),
		Delete:       c.opts.Delete && c.opts.ThreadID.IsNil(),
	}

	for idx := range scenario.Steps {
		step := &scenario.Steps[idx]
		stepResult, unmet := r.runStep(ctx, c.session, conv, idx, step)
		result.Steps = append(result.Steps, stepResult)
		if unmet != "" {
			result.Skipped = unmet
			break
		}
		if ctx.Err() != nil {
			break
		}
	}

	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		deskbench.CleanupWithin+time.Minute,
	)
	defer cancel()
	cleanup, closeErr := conv.Close(cleanupCtx, closing)
	result.Cleanup = cleanup
	if closeErr != nil {
		result.Error = benchrecord.JoinErrors(result.Error, closeErr.Error())
	}
}

func (r *Runner) openCase(
	ctx context.Context,
	session *deskbench.Session,
	opts *RunOptions,
	planned plannedCase,
	provider *aiprovider.Provider,
	swept *agentdefinition.Definition,
) (*deskbench.Conversation, error) {
	if planned.scenario.Formula != nil {
		conv, err := session.OpenFormula(ctx, planned.label(), planned.scenario.Formula)
		if err != nil {
			return nil, err
		}
		conv.Provider = provider

		return conv, nil
	}
	if planned.scenario.Palette {
		conv, err := session.OpenPalette(
			ctx,
			planned.label(),
			fmt.Sprintf("%s #%d", planned.scenario.Name, planned.repeat),
			planned.scenario.Steps[0].Say,
		)
		if err != nil {
			return nil, err
		}
		conv.Provider = provider

		return conv, nil
	}
	if opts.ThreadID.IsNotNil() {
		conv, err := session.Open(ctx, deskbench.OpenConversation{
			Label:    planned.label(),
			ThreadID: opts.ThreadID,
			Provider: provider,
		})
		if err != nil {
			return nil, err
		}
		if definition, agentErr := session.Agent(
			ctx,
			conv.Thread.AgentDefinitionID.String(),
		); agentErr == nil {
			conv.Agent = definition
		}

		return conv, nil
	}

	definition := swept
	if definition == nil {
		agentRef := planned.scenario.Agent
		if agentRef == "" {
			agentRef = opts.Agent
		}
		var err error
		definition, err = session.Agent(ctx, agentRef)
		if err != nil {
			return nil, err
		}
	}

	conv, err := session.Open(ctx, deskbench.OpenConversation{
		Label:    planned.label(),
		Agent:    definition,
		Provider: provider,
		Title:    fmt.Sprintf("%s #%d", planned.scenario.Name, planned.repeat),
	})
	if err != nil {
		return nil, err
	}
	conv.CodeMode = planned.scenario.Mode.Override()

	return conv, nil
}

func (r *Runner) runStep(
	ctx context.Context,
	session *deskbench.Session,
	conv *deskbench.Conversation,
	idx int,
	step *benchscenario.Step,
) (result *benchresult.StepResult, unmet string) {
	result = &benchresult.StepResult{
		Index:  idx + 1,
		Label:  step.Label(),
		Rubric: step.Expect.Rubric,
	}
	evidence := &benchcheck.Evidence{}
	if err := r.database.RunSetup(ctx, session, step.Setup); err != nil {
		result.Kind = "setup"
		result.Error = err.Error()
		return result, ""
	}

	if step.Decide != "" {
		decideStep(ctx, conv, step, result, evidence)
	} else if unmet = r.sayStep(ctx, session, conv, step, result, evidence); unmet != "" {
		return result, unmet
	}

	result.Formula = conv.FormulaDescription()
	evidence.Formula = conv.FormulaExpression()
	result.Checks = r.checker.Evaluate(ctx, session, &step.Expect, evidence)
	result.Passed = result.Error == "" && checksPassed(result.Checks)

	return result, ""
}

func decideStep(
	ctx context.Context,
	conv *deskbench.Conversation,
	step *benchscenario.Step,
	result *benchresult.StepResult,
	evidence *benchcheck.Evidence,
) {
	result.Kind = "decide"
	decisions, turns, err := conv.Decide(ctx, deskbench.Decide{
		Action:        step.Decide,
		Note:          step.Note,
		Tool:          step.Tool,
		Modifications: step.Modifications,
	})
	result.Decisions = decisions
	result.Turns = turns
	evidence.Turns = turns
	evidence.Decisions = decisions
	if err != nil {
		result.Error = err.Error()
	}
}

func (r *Runner) sayStep(
	ctx context.Context,
	session *deskbench.Session,
	conv *deskbench.Conversation,
	step *benchscenario.Step,
	result *benchresult.StepResult,
	evidence *benchcheck.Evidence,
) string {
	result.Kind = "say"
	utterance, err := step.Utterance()
	if err == nil {
		err = r.findMentions(ctx, session, step, &utterance)
	}
	if err == nil && step.Agent != "" {
		utterance.Directed, err = conv.DirectedAgent(ctx, step.Agent)
	}
	if err == nil {
		result.Turns, err = conv.Say(ctx, &utterance)
	}
	evidence.Turns = result.Turns
	if errors.Is(err, deskbench.ErrAttachmentUnreadable) {
		return "needs " + (&benchscenario.Requirement{DocumentText: true}).Label() + ": " +
			err.Error()
	}
	if err != nil {
		result.Error = err.Error()
	}

	return ""
}

func (r *Runner) runFireStep(
	ctx context.Context,
	session *deskbench.Session,
	cell benchcell.Cell,
	step *benchscenario.Step,
	label string,
) *benchresult.StepResult {
	result := &benchresult.StepResult{
		Index:  1,
		Kind:   "fire",
		Label:  step.Label(),
		Rubric: step.Expect.Rubric,
	}

	record, outcome, err := r.runFire(ctx, session, cell, step, label)
	if err != nil {
		result.Error = err.Error()
	}
	evidence := &benchcheck.Evidence{Run: outcome}
	if record != nil {
		result.Turns = []*benchrecord.TurnRecord{record}
		evidence.Turns = result.Turns
	}
	result.Checks = r.checker.Evaluate(ctx, session, &step.Expect, evidence)
	result.Passed = result.Error == "" && checksPassed(result.Checks)

	return result
}

func (r *Runner) sweptAgent(
	ctx context.Context,
	session *deskbench.Session,
	opts *RunOptions,
	scenario *benchscenario.Scenario,
) (*agentdefinition.Definition, func(), error) {
	if opts.Sweep == nil {
		return nil, nil, nil
	}
	if scenario.Palette || scenario.Formula != nil || opts.ThreadID.IsNotNil() {
		return nil, nil, fmt.Errorf("%w: the palette, the formula studio and a continued "+
			"conversation keep their own agent", benchscenario.ErrSweepSkipped)
	}

	agentRef := scenario.Agent
	if agentRef == "" {
		agentRef = opts.Agent
	}
	base, err := session.Agent(ctx, agentRef)
	if err != nil {
		return nil, nil, err
	}
	tools, err := benchscenario.SweepTools(base.ToolNames, scenario.ExpectedTools, opts.Sweep.Held)
	if err != nil {
		return nil, nil, err
	}

	release := r.exclusive("sweep:" + base.ID.String())
	swept, err := session.SweepAgent(ctx, base, tools)
	if err != nil {
		return nil, release, err
	}

	return swept, release, nil
}

func exclusiveKey(scenario *benchscenario.Scenario) string {
	switch {
	case scenario.Exclusive != "":
		return "scenario:" + strings.ToLower(strings.TrimSpace(scenario.Exclusive))
	case scenario.Formula != nil && scenario.Formula.EditsLibrary():
		return "library:" + strings.ToLower(strings.TrimSpace(scenario.Formula.Library))
	case scenario.Formula != nil:
		return "studio:" + strings.ToLower(strings.TrimSpace(scenario.Formula.Template))
	default:
		return ""
	}
}

func (r *Runner) liveVerdict(label string, result *benchresult.CaseResult) {
	if result.Skipped != "" {
		r.live.Linef(label, "<<< skipped: %s", result.Skipped)
		return
	}
	mark := benchresult.PassMark
	if !result.Passed {
		mark = benchresult.FailMark
	}
	r.live.Linef(
		label,
		"<<< %s in %s · %s",
		mark,
		result.FinishedAt.Sub(result.StartedAt).
			Round(100*time.Millisecond),
		benchrecord.UsageLine(&result.Usage),
	)
	if result.Error != "" {
		r.live.Linef(label, "    error: %s", result.Error)
	}
	for _, step := range result.Steps {
		if step.Error != "" {
			r.live.Linef(label, "    step %d error: %s", step.Index, step.Error)
		}
		for _, check := range step.Checks {
			if !check.Passed {
				r.live.Linef(
					label,
					"    step %d failed %s — %s",
					step.Index,
					check.Name,
					benchrecord.FirstLine(check.Detail),
				)
			}
		}
	}
}

func checksPassed(checks []benchresult.CheckResult) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}

	return true
}

func stepsPassed(steps []*benchresult.StepResult) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if !step.Passed {
			return false
		}
	}

	return true
}

func caseUsage(result *benchresult.CaseResult) benchrecord.Usage {
	usage := benchrecord.Usage{}
	for _, step := range result.Steps {
		for _, turn := range step.Turns {
			usage.Add(&turn.Usage)
		}
	}
	for _, turn := range result.Cleanup {
		usage.Add(&turn.Usage)
	}

	return usage
}

func providerModel(provider *aiprovider.Provider) string {
	if provider == nil {
		return ""
	}

	return provider.Model
}
