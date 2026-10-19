package deskbench

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

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxParallel = 8
	slugLength  = 60
)

type RunOptions struct {
	Scenarios []*Scenario
	User      string
	Agent     string
	Providers []string
	Repeat    int
	Parallel  int
	Keep      bool
	Delete    bool
	ThreadID  pulid.ID
	Progress  func(*CaseResult)
	Sweep     *Sweep
}

type CaseResult struct {
	Scenario   *Scenario     `json:"scenario"`
	Provider   string        `json:"provider"`
	Model      string        `json:"model,omitempty"`
	Effort     string        `json:"effort,omitempty"`
	Metrics    CaseMetrics   `json:"metrics"`
	Repeat     int           `json:"repeat"`
	Agent      string        `json:"agent,omitempty"`
	ThreadID   string        `json:"threadId,omitempty"`
	Steps      []*StepResult `json:"steps"`
	Cleanup    []*TurnRecord `json:"cleanup,omitempty"`
	Passed     bool          `json:"passed"`
	Skipped    string        `json:"skipped,omitempty"`
	Error      string        `json:"error,omitempty"`
	StartedAt  time.Time     `json:"startedAt"`
	FinishedAt time.Time     `json:"finishedAt"`
	Usage      Usage         `json:"usage"`
}

func (c *CaseResult) Key() string {
	return c.Scenario.Name + " @ " + c.Provider
}

func (c *CaseResult) Errors() []string {
	errs := make([]string, 0, len(c.Steps)+1)
	if c.Error != "" {
		errs = append(errs, c.Error)
	}
	for _, step := range c.Steps {
		if step.Error != "" {
			errs = append(errs, "step "+strconv.Itoa(step.Index)+": "+step.Error)
		}
	}

	return errs
}

func (c *CaseResult) ID() string {
	return stringutils.SlugifyASCII(c.Scenario.Name, slugLength) + "--" +
		stringutils.SlugifyASCII(c.Provider, slugLength) + "--" + strconv.Itoa(c.Repeat)
}

type StepResult struct {
	Index     int               `json:"index"`
	Kind      string            `json:"kind"`
	Label     string            `json:"label"`
	Turns     []*TurnRecord     `json:"turns"`
	Decisions []*DecisionRecord `json:"decisions,omitempty"`
	Checks    []CheckResult     `json:"checks"`
	Rubric    string            `json:"rubric,omitempty"`
	Formula   string            `json:"formula,omitempty"`
	Error     string            `json:"error,omitempty"`
	Passed    bool              `json:"passed"`
}

type plannedCase struct {
	scenario      *Scenario
	provider      string
	repeat        int
	scenarioOrder int
	providerOrder int
}

func (p plannedCase) label() string {
	provider := p.provider
	if provider == "" {
		provider = DefaultProviderLabel
	}

	return p.scenario.Name + " @ " + provider + " #" + strconv.Itoa(p.repeat)
}

func (b *Bench) Run(ctx context.Context, opts RunOptions) ([]*CaseResult, error) {
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

	results := make([]*CaseResult, len(planned))
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
				result := b.runCase(ctx, &opts, planned[idx])
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

	finished := make([]*CaseResult, 0, len(results))
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

func (b *Bench) runCase(ctx context.Context, opts *RunOptions, planned plannedCase) *CaseResult {
	scenario := planned.scenario
	if key := exclusiveKey(scenario); key != "" {
		unlock := b.exclusive(key)
		defer unlock()
	}
	result := &CaseResult{
		Scenario:  scenario,
		Provider:  DefaultProviderLabel,
		Repeat:    planned.repeat,
		StartedAt: time.Now(),
	}
	b.live.Line(planned.label(), ">>> start: %s", firstLine(scenario.Description))
	defer func() {
		result.FinishedAt = time.Now()
		result.Usage = caseUsage(result)
		result.Passed = result.Error == "" && stepsPassed(result.Steps)
		result.Metrics = measureCase(result)
		b.liveVerdict(planned.label(), result)
	}()

	user := scenario.User
	if user == "" {
		user = opts.User
	}
	session, err := b.SessionFor(ctx, user)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	cell, err := session.Cell(ctx, planned.provider)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	provider := cell.Provider
	result.Provider = cell.Label()
	result.Model = providerModel(provider)
	result.Effort = string(cell.Effort)

	b.tidied.Do(func() {
		if tidyErr := b.runStatements(ctx, session, benchLeftovers()); tidyErr != nil {
			b.live.Line("bench", "could not cancel what earlier bench runs left open: %v", tidyErr)
		}
	})
	if err = b.runSetup(ctx, session, scenario.Setup); err != nil {
		result.Error = err.Error()
		return result
	}
	if len(scenario.Setup) > 0 {
		b.live.Line(
			planned.label(),
			"setup: %d statement(s) put the data in place",
			len(scenario.Setup),
		)
	}

	if scenario.Unattended() {
		if opts.Sweep != nil {
			result.Skipped = "the tool-count sweep runs chat scenarios"
			return result
		}
		result.Steps = append(result.Steps, b.runFireStep(ctx, session, cell, &scenario.Steps[0],
			planned.label()))
		return result
	}

	swept, release, err := b.sweptAgent(ctx, session, opts, scenario)
	if release != nil {
		defer release()
	}
	if err != nil {
		if errors.Is(err, ErrSweepSkipped) {
			result.Skipped = err.Error()
		} else {
			result.Error = err.Error()
		}
		return result
	}

	conv, err := b.openCase(ctx, session, opts, planned, provider, swept)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.ThreadID = conv.Thread.ID.String()
	if cell.Effort != "" {
		unpin := b.recorder.Pin(conv.Thread.ID, ProviderPin{Effort: cell.Effort})
		defer unpin()
	}
	if conv.Agent != nil {
		result.Agent = conv.Agent.Name
	}
	closing := CloseOptions{
		LeavePending: opts.Keep || opts.ThreadID.IsNotNil(),
		Delete:       opts.Delete && opts.ThreadID.IsNil(),
	}

	for idx := range scenario.Steps {
		step := &scenario.Steps[idx]
		stepResult := b.runStep(ctx, session, conv, idx, step)
		result.Steps = append(result.Steps, stepResult)
		if ctx.Err() != nil {
			break
		}
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupWithin+time.Minute)
	defer cancel()
	cleanup, closeErr := conv.Close(cleanupCtx, closing)
	result.Cleanup = cleanup
	if closeErr != nil {
		result.Error = joinErrors(result.Error, closeErr.Error())
	}

	return result
}

func (b *Bench) openCase(
	ctx context.Context,
	session *Session,
	opts *RunOptions,
	planned plannedCase,
	provider *aiprovider.Provider,
	swept *agentdefinition.Definition,
) (*Conversation, error) {
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
		conv, err := session.Open(ctx, OpenConversation{
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

	return session.Open(ctx, OpenConversation{
		Label:    planned.label(),
		Agent:    definition,
		Provider: provider,
		Title:    fmt.Sprintf("%s #%d", planned.scenario.Name, planned.repeat),
	})
}

func (b *Bench) runStep(
	ctx context.Context,
	session *Session,
	conv *Conversation,
	idx int,
	step *Step,
) *StepResult {
	result := &StepResult{
		Index:  idx + 1,
		Label:  step.Label(),
		Rubric: step.Expect.Rubric,
	}
	evidence := &stepEvidence{}

	if step.Decide != "" {
		result.Kind = "decide"
		decisions, turns, err := conv.Decide(ctx, Decide{
			Action:        step.Decide,
			Note:          step.Note,
			Tool:          step.Tool,
			Modifications: step.Modifications,
		})
		result.Decisions = decisions
		result.Turns = turns
		evidence.turns = turns
		evidence.decisions = decisions
		if err != nil {
			result.Error = err.Error()
		}
	} else {
		result.Kind = "say"
		utterance, err := step.Utterance()
		if err == nil {
			err = b.findMentions(ctx, session, step, &utterance)
		}
		if err == nil && step.Agent != "" {
			utterance.Directed, err = conv.directedAgent(ctx, step.Agent)
		}
		if err == nil {
			result.Turns, err = conv.Say(ctx, &utterance)
		}
		evidence.turns = result.Turns
		if err != nil {
			result.Error = err.Error()
		}
	}

	result.Formula = conv.formula.describe()
	evidence.formula = conv.formula.expression()
	result.Checks = b.evaluate(ctx, session, &step.Expect, evidence)
	result.Passed = result.Error == "" && checksPassed(result.Checks)

	return result
}

func (b *Bench) runFireStep(
	ctx context.Context,
	session *Session,
	cell Cell,
	step *Step,
	label string,
) *StepResult {
	result := &StepResult{
		Index:  1,
		Kind:   "fire",
		Label:  step.Label(),
		Rubric: step.Expect.Rubric,
	}

	record, outcome, err := b.runFire(ctx, session, cell, step, label)
	if err != nil {
		result.Error = err.Error()
	}
	evidence := &stepEvidence{run: outcome}
	if record != nil {
		result.Turns = []*TurnRecord{record}
		evidence.turns = result.Turns
	}
	result.Checks = b.evaluate(ctx, session, &step.Expect, evidence)
	result.Passed = result.Error == "" && checksPassed(result.Checks)

	return result
}

func (b *Bench) sweptAgent(
	ctx context.Context,
	session *Session,
	opts *RunOptions,
	scenario *Scenario,
) (*agentdefinition.Definition, func(), error) {
	if opts.Sweep == nil {
		return nil, nil, nil
	}
	if scenario.Palette || scenario.Formula != nil || opts.ThreadID.IsNotNil() {
		return nil, nil, fmt.Errorf("%w: the palette, the formula studio and a continued "+
			"conversation keep their own agent", ErrSweepSkipped)
	}

	agentRef := scenario.Agent
	if agentRef == "" {
		agentRef = opts.Agent
	}
	base, err := session.Agent(ctx, agentRef)
	if err != nil {
		return nil, nil, err
	}
	tools, err := sweepTools(base.ToolNames, scenario.ExpectedTools, opts.Sweep.Held)
	if err != nil {
		return nil, nil, err
	}

	release := b.exclusive("sweep:" + base.ID.String())
	swept, err := session.sweepAgent(ctx, base, tools)
	if err != nil {
		return nil, release, err
	}

	return swept, release, nil
}

func exclusiveKey(scenario *Scenario) string {
	switch {
	case scenario.Exclusive != "":
		return "scenario:" + strings.ToLower(strings.TrimSpace(scenario.Exclusive))
	case scenario.Formula != nil:
		return "studio:" + strings.ToLower(strings.TrimSpace(scenario.Formula.Template))
	default:
		return ""
	}
}

func (b *Bench) liveVerdict(label string, result *CaseResult) {
	if result.Skipped != "" {
		b.live.Line(label, "<<< skipped: %s", result.Skipped)
		return
	}
	mark := passMark
	if !result.Passed {
		mark = failMark
	}
	b.live.Line(
		label,
		"<<< %s in %s · %s",
		mark,
		result.FinishedAt.Sub(result.StartedAt).
			Round(100*time.Millisecond),
		usageLine(result.Usage),
	)
	if result.Error != "" {
		b.live.Line(label, "    error: %s", result.Error)
	}
	for _, step := range result.Steps {
		if step.Error != "" {
			b.live.Line(label, "    step %d error: %s", step.Index, step.Error)
		}
		for _, check := range step.Checks {
			if !check.Passed {
				b.live.Line(
					label,
					"    step %d failed %s — %s",
					step.Index,
					check.Name,
					firstLine(check.Detail),
				)
			}
		}
	}
}

func checksPassed(checks []CheckResult) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}

	return true
}

func stepsPassed(steps []*StepResult) bool {
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

func caseUsage(result *CaseResult) Usage {
	usage := Usage{}
	for _, step := range result.Steps {
		for _, turn := range step.Turns {
			usage.add(turn.Usage)
		}
	}
	for _, turn := range result.Cleanup {
		usage.add(turn.Usage)
	}

	return usage
}

func providerModel(provider *aiprovider.Provider) string {
	if provider == nil {
		return ""
	}

	return provider.Model
}
