package deskbench

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	low     = "claude-haiku-5-5@low"
	medium  = "claude-haiku-5-5@medium"
	sonnet  = "claude-sonnet-5-5@medium"
	advisor = "claude-haiku-5-5@low+advisor"
	code    = "claude-haiku-5-5@low+code"
)

func gateSpec() *GateSpec {
	spec := &GateSpec{
		Repeat: 4,
		PassK:  4,
		Before: medium,
		Budget: GateBudget{MaxUSD: 10, MaxHours: 2, Overrun: 1.25, ProbeEvery: 2},
		Groups: []GateGroup{
			{Name: "full", Cells: []string{low, medium, sonnet, advisor}},
			{Name: "pairs", Cells: []string{code}, Tags: []string{codeModeTag}},
		},
		Effort:   EffortRule{Candidates: []string{low, medium}, Within: 0.03, MinScenarios: 2},
		Advisor:  AdvisorRule{Cell: advisor, Gain: 0.05, CostBar: sonnet},
		CodeMode: []CodePair{{Model: "Haiku 5.5", Code: code, Discrete: low}},
		HeldOut:  HeldOutRule{Tolerance: 0.03, SelectionSlack: 0.02},
	}
	if err := spec.validate(); err != nil {
		panic(err)
	}

	return spec
}

type trialPlan struct {
	scenario *Scenario
	cell     string
	passed   int
	cost     string
}

func trials(plans ...trialPlan) []*CaseResult {
	cases := make([]*CaseResult, 0, len(plans)*4)
	started := time.Unix(1_700_000_000, 0)
	for _, plan := range plans {
		for rep := range 4 {
			price := decimal.RequireFromString(plan.cost)
			cases = append(cases, &CaseResult{
				Scenario:   plan.scenario,
				Cell:       plan.cell,
				Provider:   plan.cell,
				Repeat:     rep + 1,
				Passed:     rep < plan.passed,
				StartedAt:  started,
				FinishedAt: started.Add(30 * time.Second),
				Usage:      Usage{ModelCalls: 2, CostUSD: &price},
			})
		}
	}

	return cases
}

func chat(name string, split evalsplit.Split, tags ...string) *Scenario {
	return &Scenario{Name: name, Split: split, Tags: tags}
}

func unattended(name string, split evalsplit.Split) *Scenario {
	return &Scenario{
		Name:  name,
		Split: split,
		Steps: []Step{{Fire: &Fire{Agent: "DetentionDesk"}}},
	}
}

func TestLoadGateSpec_TheShippedSpecIsValid(t *testing.T) {
	t.Parallel()

	spec, err := LoadGateSpec("../../deskbench/gate.yaml")
	require.NoError(t, err)
	assert.Equal(t, 4, spec.Repeat)
	assert.Equal(t, 4, spec.PassK)
	assert.Contains(t, spec.Effort.Candidates, spec.Before)
	for _, cell := range spec.Cells() {
		assert.Contains(t, cell, "@", "every cell names its effort: %s", cell)
	}
}

func TestGateSpec_RefusesRulesThatNameNoRunCell(t *testing.T) {
	t.Parallel()

	spec := gateSpec()
	spec.Advisor.CostBar = "gpt-6-luna@medium"
	require.ErrorIs(t, spec.validate(), ErrGateSpec)

	spec = gateSpec()
	spec.Before = sonnet
	require.ErrorIs(t, spec.validate(), ErrGateSpec)

	spec = gateSpec()
	spec.Repeat = 2
	require.ErrorIs(t, spec.validate(), ErrGateSpec)
}

func TestRouteOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, RouteChat, RouteOf(chat("a", evalsplit.Development)))
	assert.Equal(t, RouteDelegated, RouteOf(chat("b", evalsplit.Development, "delegation")))
	assert.Equal(t, RouteDelegated, RouteOf(&Scenario{ExpectedTools: []string{"delegate_task"}}))
	assert.Equal(t, RouteUnattended, RouteOf(unattended("c", evalsplit.Development)))
}

func TestEvaluateGate_ChoosesTheCheaperEffortWithinTheMargin(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	u1, u2 := unattended("u1", evalsplit.Development), unattended("u2", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 4, "0.01"}, trialPlan{b, low, 4, "0.01"},
		trialPlan{a, medium, 4, "0.02"}, trialPlan{b, medium, 4, "0.02"},
		trialPlan{u1, low, 2, "0.01"}, trialPlan{u2, low, 2, "0.01"},
		trialPlan{u1, medium, 4, "0.03"}, trialPlan{u2, medium, 4, "0.03"},
	)

	result := EvaluateGate(gateSpec(), cases)
	byRoute := map[Route]EffortDecision{}
	for _, decision := range result.Efforts {
		byRoute[decision.Route] = decision
	}
	assert.Equal(t, low, byRoute[RouteChat].Chosen, "equal pass^4, low is cheaper")
	assert.Equal(t, medium, byRoute[RouteUnattended].Chosen, "low is not within 0.03")
	assert.Equal(t, low, byRoute[RouteDelegated].Chosen)
	assert.True(t, byRoute[RouteDelegated].Inherited, "no delegated scenarios: takes chat's")
}

func TestEvaluateGate_OffersTheAdvisorOnlyWhenEveryConditionHolds(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 3, "0.01"}, trialPlan{b, low, 3, "0.01"},
		trialPlan{a, medium, 3, "0.02"}, trialPlan{b, medium, 3, "0.02"},
		trialPlan{a, advisor, 4, "0.05"}, trialPlan{b, advisor, 4, "0.05"},
		trialPlan{a, sonnet, 4, "0.40"}, trialPlan{b, sonnet, 4, "0.40"},
	)

	result := EvaluateGate(gateSpec(), cases)
	require.NotEmpty(t, result.Advisors)
	assert.True(t, result.Advisors[0].Offered, result.Advisors[0].Reason)
	assert.Equal(t, advisor, result.Configuration[RouteChat])

	cases[len(cases)-9].Unauthorized = []Unauthorized{{Rules: []EffectRule{EffectUnapproved}}}
	result = EvaluateGate(gateSpec(), cases)
	assert.False(t, result.Advisors[0].Offered, "one unauthorized effect withholds it")
	assert.Contains(t, result.Advisors[0].Reason, "no unauthorized effect")
}

func TestEvaluateGate_CodeModeMustBeatDiscreteSafely(t *testing.T) {
	t.Parallel()

	p1 := chat("p1", evalsplit.Development, codeModeTag)
	p2 := chat("p2", evalsplit.Development, codeModeTag)
	p3 := chat("p3", evalsplit.Development, codeModeTag)
	base := []trialPlan{
		{p1, low, 2, "0.01"}, {p2, low, 3, "0.01"}, {p3, low, 4, "0.01"},
		{p1, medium, 2, "0.02"}, {p2, medium, 3, "0.02"}, {p3, medium, 4, "0.02"},
	}

	cases := trials(append(base,
		trialPlan{p1, code, 4, "0.01"}, trialPlan{p2, code, 4, "0.01"},
		trialPlan{p3, code, 4, "0.01"})...)
	result := EvaluateGate(gateSpec(), cases)
	require.Len(t, result.CodeMode, 1)
	assert.True(t, result.CodeMode[0].Enabled, result.CodeMode[0].Reason)

	cases = trials(append(base,
		trialPlan{p1, code, 4, "0.01"}, trialPlan{p2, code, 4, "0.01"},
		trialPlan{p3, code, 2, "0.01"})...)
	result = EvaluateGate(gateSpec(), cases)
	assert.False(t, result.CodeMode[0].Enabled, "p3 is a hard regression")
	assert.Equal(t, []string{"p3"}, result.CodeMode[0].Regressions)

	cases = trials(append(base,
		trialPlan{p1, code, 2, "0.01"}, trialPlan{p2, code, 3, "0.01"},
		trialPlan{p3, code, 4, "0.01"})...)
	result = EvaluateGate(gateSpec(), cases)
	assert.False(t, result.CodeMode[0].Enabled, "a tie is not a win")
}

func TestConfirmGate_AVetoedRouteKeepsTheBeforeCell(t *testing.T) {
	t.Parallel()

	da, db := chat("da", evalsplit.Development), chat("db", evalsplit.Development)
	ha, hb := chat("ha", evalsplit.HeldOut), chat("hb", evalsplit.HeldOut)
	run1 := trials(
		trialPlan{da, low, 4, "0.01"}, trialPlan{db, low, 4, "0.01"},
		trialPlan{da, medium, 4, "0.02"}, trialPlan{db, medium, 4, "0.02"},
		trialPlan{ha, low, 4, "0.01"}, trialPlan{hb, low, 4, "0.01"},
		trialPlan{ha, medium, 4, "0.02"}, trialPlan{hb, medium, 4, "0.02"},
	)
	gate := EvaluateGate(gateSpec(), run1)
	require.Equal(t, low, gate.Configuration[RouteChat])

	groups := gate.ConfirmGroups()
	require.NotEmpty(t, groups)
	assert.Contains(t, groups[0].Cells, low)
	assert.Equal(t, []evalsplit.Split{evalsplit.HeldOut}, groups[0].Splits)

	run2 := trials(
		trialPlan{ha, low, 4, "0.01"}, trialPlan{hb, low, 4, "0.01"},
		trialPlan{ha, medium, 4, "0.02"}, trialPlan{hb, medium, 4, "0.02"},
	)
	confirmed := ConfirmGate(gate, run1, run2)
	assert.True(t, confirmed.Routes[0].Holds)
	assert.Equal(t, low, confirmed.Configuration[RouteChat])
	require.NotNil(t, confirmed.After.Figure.PassK)
	assert.InDelta(t, 1.0, *confirmed.After.Figure.PassK, 1e-9)

	run2 = trials(
		trialPlan{ha, low, 2, "0.01"}, trialPlan{hb, low, 2, "0.01"},
		trialPlan{ha, medium, 4, "0.02"}, trialPlan{hb, medium, 4, "0.02"},
	)
	confirmed = ConfirmGate(gate, run1, run2)
	assert.False(t, confirmed.Routes[0].Holds)
	assert.Equal(t, medium, confirmed.Configuration[RouteChat], "falls back to before")
	assert.False(t, confirmed.Descriptions.Passed)
	assert.Contains(t, confirmed.Render(), "## Headline")
}

func TestProject_ScalesTheProbeToThePlannedTrials(t *testing.T) {
	t.Parallel()

	spec := gateSpec()
	scenarios := make([]*Scenario, 0, 10)
	for idx := range 10 {
		scenarios = append(scenarios, chat("s"+strconv.Itoa(idx), evalsplit.Development))
	}
	group := &spec.Groups[0]
	probed := ProbeScenarios(spec, group, scenarios)
	assert.Len(t, probed, 5, "every second development scenario")

	probe := make([]*CaseResult, 0, 4)
	for _, cell := range []string{low, medium, sonnet, advisor, code} {
		probe = append(probe, trials(trialPlan{probed[0], cell, 1, "0.10"})[0])
	}
	projection := Project(spec, probe, scenarios, 2)
	assert.InDelta(t, 4*40*0.10, projection.Cost, 1e-9, "four cells, ten scenarios, four trials")
	assert.False(t, projection.Within, "$16 is over the $10 budget")
	assert.InDelta(t, projection.Cost*1.25, projection.Cap, 1e-9)
	assert.True(t, strings.Contains(projection.Render(), "Within budget: no"))
}
