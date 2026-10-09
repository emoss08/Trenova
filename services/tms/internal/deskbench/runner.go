package deskbench

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
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
}

type CaseResult struct {
	Scenario   *Scenario     `json:"scenario"`
	Provider   string        `json:"provider"`
	Model      string        `json:"model,omitempty"`
	Repeat     int           `json:"repeat"`
	Agent      string        `json:"agent,omitempty"`
	ThreadID   string        `json:"threadId,omitempty"`
	Steps      []*StepResult `json:"steps"`
	Cleanup    []*TurnRecord `json:"cleanup,omitempty"`
	Passed     bool          `json:"passed"`
	Error      string        `json:"error,omitempty"`
	StartedAt  time.Time     `json:"startedAt"`
	FinishedAt time.Time     `json:"finishedAt"`
	Usage      Usage         `json:"usage"`
}

func (c *CaseResult) Key() string {
	return c.Scenario.Name + " @ " + c.Provider
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
	scenario *Scenario
	provider string
	repeat   int
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
	for _, scenario := range opts.Scenarios {
		for _, provider := range providers {
			chosen := provider
			if chosen == "" {
				chosen = scenario.Provider
			}
			for rep := 1; rep <= repeat; rep++ {
				planned = append(planned, plannedCase{scenario: scenario, provider: chosen, repeat: rep})
			}
		}
	}

	results := make([]*CaseResult, len(planned))
	var progressMu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, parallel)
	for idx := range planned {
		if ctx.Err() != nil {
			break
		}
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			result := b.runCase(ctx, &opts, planned[idx])
			results[idx] = result
			if opts.Progress != nil {
				progressMu.Lock()
				opts.Progress(result)
				progressMu.Unlock()
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

	provider, err := session.Provider(ctx, planned.provider)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Provider = ProviderLabel(provider)
	result.Model = providerModel(provider)

	conv, err := b.openCase(ctx, session, opts, planned, provider)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.ThreadID = conv.Thread.ID.String()
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
) (*Conversation, error) {
	if planned.scenario.Formula != nil {
		conv, err := session.OpenFormula(ctx, planned.label(), planned.scenario.Formula)
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
		if definition, agentErr := session.Agent(ctx, conv.Thread.AgentDefinitionID.String()); agentErr == nil {
			conv.Agent = definition
		}

		return conv, nil
	}

	agentRef := planned.scenario.Agent
	if agentRef == "" {
		agentRef = opts.Agent
	}
	definition, err := session.Agent(ctx, agentRef)
	if err != nil {
		return nil, err
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
		if err == nil && step.Agent != "" {
			utterance.Directed, err = conv.directedAgent(ctx, step.Agent)
		}
		if err == nil {
			result.Turns, err = conv.Say(ctx, utterance)
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
	mark := passMark
	if !result.Passed {
		mark = failMark
	}
	b.live.Line(label, "<<< %s in %s · %s", mark,
		result.FinishedAt.Sub(result.StartedAt).Round(100*time.Millisecond), usageLine(result.Usage))
	if result.Error != "" {
		b.live.Line(label, "    error: %s", result.Error)
	}
	for _, step := range result.Steps {
		if step.Error != "" {
			b.live.Line(label, "    step %d error: %s", step.Index, step.Error)
		}
		for _, check := range step.Checks {
			if !check.Passed {
				b.live.Line(label, "    step %d failed %s — %s", step.Index, check.Name, firstLine(check.Detail))
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
