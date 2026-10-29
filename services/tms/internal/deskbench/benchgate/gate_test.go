package benchgate

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchreport"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	low         = "claude-haiku-5-5@low"
	medium      = "claude-haiku-5-5@medium"
	sonnet      = "claude-sonnet-5-5@medium"
	advisorCell = "claude-haiku-5-5@low+advisor"
)

func gateSpec() *GateSpec {
	spec := &GateSpec{
		Repeat: 4,
		PassK:  4,
		Before: medium,
		Budget: GateBudget{MaxUSD: 10, MaxHours: 2, Overrun: 1.25, ProbeEvery: 2},
		Groups: []GateGroup{
			{Name: "full", Cells: []string{low, medium, sonnet}},
			{Name: "advisor", Cells: []string{advisorCell}, AdvisorRoutes: true},
		},
		Effort:  EffortRule{Candidates: []string{low, medium}, Within: 0.02, MinScenarios: 2},
		Advisor: AdvisorRule{Cell: advisorCell, Reference: sonnet, Gap: 0.10, Gain: 0.05},
		HeldOut: HeldOutRule{Tolerance: 0.03, SelectionSlack: 0.02},
	}
	if err := spec.validate(); err != nil {
		panic(err)
	}

	return spec
}

type trialPlan struct {
	scenario *benchscenario.Scenario
	cell     string
	passed   int
	cost     string
}

func trials(plans ...trialPlan) []*benchresult.CaseResult {
	cases := make([]*benchresult.CaseResult, 0, len(plans)*4)
	started := time.Unix(1_700_000_000, 0)
	for _, plan := range plans {
		for rep := range 4 {
			price := decimal.RequireFromString(plan.cost)
			cases = append(cases, &benchresult.CaseResult{
				Scenario:   plan.scenario,
				Cell:       plan.cell,
				Provider:   plan.cell,
				Repeat:     rep + 1,
				Passed:     rep < plan.passed,
				StartedAt:  started,
				FinishedAt: started.Add(30 * time.Second),
				Usage:      benchrecord.Usage{ModelCalls: 2, CostUSD: &price},
			})
		}
	}

	return cases
}

func chat(name string, split evalsplit.Split, tags ...string) *benchscenario.Scenario {
	return &benchscenario.Scenario{Name: name, Split: split, Tags: tags}
}

func unattended(name string, split evalsplit.Split) *benchscenario.Scenario {
	return &benchscenario.Scenario{
		Name:  name,
		Split: split,
		Steps: []benchscenario.Step{{Fire: &benchscenario.Fire{Agent: "DetentionDesk"}}},
	}
}

func TestLoadGateSpec_TheShippedSpecIsValid(t *testing.T) {
	t.Parallel()

	spec, err := LoadGateSpec("../../../deskbench/gate.yaml")
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
	spec.Advisor.Reference = "gpt-6-luna@medium"
	require.ErrorIs(t, spec.validate(), ErrGateSpec)

	spec = gateSpec()
	spec.Groups[1].AdvisorRoutes = false
	require.ErrorIs(t, spec.validate(), ErrGateSpec, "the advisor runs only where it may help")

	spec = gateSpec()
	spec.Before = sonnet
	require.ErrorIs(t, spec.validate(), ErrGateSpec)

	spec = gateSpec()
	spec.Repeat = 2
	require.ErrorIs(t, spec.validate(), ErrGateSpec)
}

func TestRouteOf(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		benchscenario.RouteChat,
		benchscenario.RouteOf(chat("a", evalsplit.Development)),
	)
	assert.Equal(
		t,
		benchscenario.RouteDelegated,
		benchscenario.RouteOf(chat("b", evalsplit.Development, "delegation")),
	)
	assert.Equal(
		t,
		benchscenario.RouteDelegated,
		benchscenario.RouteOf(&benchscenario.Scenario{ExpectedTools: []string{"delegate_task"}}),
	)
	assert.Equal(
		t,
		benchscenario.RouteUnattended,
		benchscenario.RouteOf(unattended("c", evalsplit.Development)),
	)
}

func TestEvaluateGate_ChoosesTheLowestEffortWithinTwoPoints(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	u1, u2 := unattended("u1", evalsplit.Development), unattended("u2", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 4, "0.02"}, trialPlan{b, low, 4, "0.02"},
		trialPlan{a, medium, 4, "0.01"}, trialPlan{b, medium, 4, "0.01"},
		trialPlan{u1, low, 3, "0.01"}, trialPlan{u2, low, 4, "0.01"},
		trialPlan{u1, medium, 4, "0.03"}, trialPlan{u2, medium, 4, "0.03"},
	)

	result := EvaluateGate(gateSpec(), cases)
	byRoute := map[benchscenario.Route]EffortDecision{}
	for _, decision := range result.Efforts {
		byRoute[decision.Route] = decision
	}
	assert.Equal(t, low, byRoute[benchscenario.RouteChat].Chosen,
		"equal pass^4: the lower effort, even though medium costs less here")
	assert.Equal(
		t,
		medium,
		byRoute[benchscenario.RouteUnattended].Chosen,
		"low is 0.125 behind, not within 0.02",
	)
	assert.Equal(t, low, byRoute[benchscenario.RouteDelegated].Chosen)
	assert.True(
		t,
		byRoute[benchscenario.RouteDelegated].Inherited,
		"no delegated scenarios: takes chat's",
	)
}

func TestAdvisorRoutes_OnlyWhereHaikuTrailsTheReferenceByTenPoints(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	u1, u2 := unattended("u1", evalsplit.Development), unattended("u2", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 3, "0.01"}, trialPlan{b, low, 3, "0.01"},
		trialPlan{a, medium, 3, "0.02"}, trialPlan{b, medium, 3, "0.02"},
		trialPlan{a, sonnet, 4, "0.40"}, trialPlan{b, sonnet, 4, "0.40"},
		trialPlan{u1, low, 4, "0.01"}, trialPlan{u2, low, 4, "0.01"},
		trialPlan{u1, medium, 4, "0.02"}, trialPlan{u2, medium, 4, "0.02"},
		trialPlan{u1, sonnet, 4, "0.40"}, trialPlan{u2, sonnet, 4, "0.40"},
	)

	assert.Equal(
		t,
		[]benchscenario.Route{benchscenario.RouteChat, benchscenario.RouteDelegated},
		AdvisorRoutes(gateSpec(), cases),
		"chat trails Sonnet by 0.75; unattended does not trail; delegated takes chat's",
	)
}

func TestEvaluateGate_OffersTheAdvisorOnlyWhenEveryConditionHolds(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 3, "0.01"}, trialPlan{b, low, 3, "0.01"},
		trialPlan{a, medium, 3, "0.02"}, trialPlan{b, medium, 3, "0.02"},
		trialPlan{a, advisorCell, 4, "0.05"}, trialPlan{b, advisorCell, 4, "0.05"},
		trialPlan{a, sonnet, 4, "0.40"}, trialPlan{b, sonnet, 4, "0.40"},
	)

	result := EvaluateGate(gateSpec(), cases)
	require.NotEmpty(t, result.Advisors)
	assert.True(t, result.Advisors[0].Offered, result.Advisors[0].Reason)
	assert.Equal(t, advisorCell, result.Configuration[benchscenario.RouteChat])

	cases[len(cases)-9].Unauthorized = []benchresult.Unauthorized{
		{Rules: []benchresult.EffectRule{benchresult.EffectUnapproved}},
	}
	result = EvaluateGate(gateSpec(), cases)
	assert.False(t, result.Advisors[0].Offered, "one unauthorized effect withholds it")
	assert.Contains(t, result.Advisors[0].Reason, "no unauthorized effect")
}

func TestEvaluateGate_NoAdvisorWhereHaikuIsCloseToTheReference(t *testing.T) {
	t.Parallel()

	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	cases := trials(
		trialPlan{a, low, 4, "0.01"}, trialPlan{b, low, 4, "0.01"},
		trialPlan{a, medium, 4, "0.02"}, trialPlan{b, medium, 4, "0.02"},
		trialPlan{a, sonnet, 4, "0.40"}, trialPlan{b, sonnet, 4, "0.40"},
	)

	result := EvaluateGate(gateSpec(), cases)
	require.NotEmpty(t, result.Advisors)
	assert.False(t, result.Advisors[0].Eligible)
	assert.False(t, result.Advisors[0].Offered)
	assert.Contains(t, result.Advisors[0].Reason, "not tried")
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
	require.Equal(t, low, gate.Configuration[benchscenario.RouteChat])

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
	assert.Equal(t, low, confirmed.Configuration[benchscenario.RouteChat])
	require.NotNil(t, confirmed.After.Figure.PassK)
	assert.InDelta(t, 1.0, *confirmed.After.Figure.PassK, 1e-9)

	run2 = trials(
		trialPlan{ha, low, 2, "0.01"}, trialPlan{hb, low, 2, "0.01"},
		trialPlan{ha, medium, 4, "0.02"}, trialPlan{hb, medium, 4, "0.02"},
	)
	confirmed = ConfirmGate(gate, run1, run2)
	assert.False(t, confirmed.Routes[0].Holds)
	assert.Equal(
		t,
		medium,
		confirmed.Configuration[benchscenario.RouteChat],
		"falls back to before",
	)
	assert.False(t, confirmed.Descriptions.Passed)
	assert.Contains(t, confirmed.Render(), "## Headline")
}

func TestProject_ScalesTheProbeToThePlannedTrials(t *testing.T) {
	t.Parallel()

	spec := gateSpec()
	scenarios := make([]*benchscenario.Scenario, 0, 10)
	for idx := range 10 {
		scenarios = append(scenarios, chat("s"+strconv.Itoa(idx), evalsplit.Development))
	}
	group := &spec.Groups[0]
	probed := ProbeScenarios(spec, group, scenarios)
	assert.Len(t, probed, 5, "every second development scenario")

	probe := make([]*benchresult.CaseResult, 0, 4)
	for _, cell := range []string{low, medium, sonnet, advisorCell} {
		probe = append(probe, trials(trialPlan{probed[0], cell, 1, "0.10"})[0])
	}
	projection := Project(spec, probe, scenarios, 2)
	assert.InDelta(t, 4*40*0.10, projection.Cost, 1e-9,
		"four cells, ten scenarios, four trials; the advisor cell is projected on every route")
	assert.False(t, projection.Within, "$16 is over the $10 budget")
	assert.InDelta(t, projection.Cost*1.25, projection.Cap, 1e-9)
	assert.True(t, strings.Contains(projection.Render(), "Within budget: no"))
}

func TestWriteGateRun_HandsOverNoHeldOutDetail(t *testing.T) {
	t.Parallel()

	dev := chat("dev-visible-scenario", evalsplit.Development)
	held := chat("heldout-secret-scenario", evalsplit.HeldOut)
	adversarial := chat("adversarial-secret-scenario", evalsplit.Adversarial)
	cases := trials(
		trialPlan{dev, low, 3, "0.01"}, trialPlan{dev, medium, 4, "0.02"},
		trialPlan{held, low, 2, "0.01"}, trialPlan{held, medium, 3, "0.02"},
		trialPlan{adversarial, low, 4, "0.01"}, trialPlan{adversarial, medium, 4, "0.02"},
	)
	for _, result := range cases {
		label := "dev-check-label"
		if result.Scenario.Split != evalsplit.Development {
			label = "secret-check-label " + result.Scenario.Name
		}
		result.Steps = []*benchresult.StepResult{{
			Index: 1, Kind: "say", Label: label,
			Checks: []benchresult.CheckResult{{Name: label, Detail: label}},
			Turns:  []*benchrecord.TurnRecord{{Reply: "reply for " + result.Scenario.Name}},
		}}
	}

	root := t.TempDir()
	meta := benchreport.ReportMeta{PassK: 4, StartedAt: time.Unix(1_700_000_000, 0)}
	require.NoError(t, WriteGateRun(root, &meta, cases))
	gate := EvaluateGate(gateSpec(), cases)
	gate.Run = root
	require.NoError(t, gate.Write(root))

	secrets := []string{"heldout-secret", "adversarial-secret", "secret-check-label"}
	checked := 0
	read := func(path string) {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		checked++
		for _, secret := range secrets {
			assert.NotContains(t, string(raw), secret, path)
		}
	}
	for _, path := range append(HandoverPaths(root), filepath.Join(root, "summary.md")) {
		info, err := os.Stat(path)
		require.NoError(t, err, path)
		if !info.IsDir() {
			read(path)
			continue
		}
		require.NoError(
			t,
			filepath.WalkDir(path, func(file string, entry fs.DirEntry, err error) error {
				if err == nil && !entry.IsDir() {
					read(file)
				}
				return err
			}),
		)
	}
	assert.Greater(t, checked, 3, "the handover holds the gate and the development run")

	sealed, err := LoadGateRunReport(root)
	require.NoError(t, err)
	assert.Len(t, sealed.Cases, len(cases), "the full report is kept, sealed")
	raw, err := os.ReadFile(filepath.Join(root, "development", "transcript.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "dev-visible-scenario")
}

func TestConfirmGate_ComparesDescriptionsOnlyOnScenariosBothRunsHold(t *testing.T) {
	t.Parallel()

	da, db := chat("da", evalsplit.Development), chat("db", evalsplit.Development)
	ha, hb := chat("ha", evalsplit.HeldOut), chat("hb", evalsplit.HeldOut)
	run1 := trials(
		trialPlan{da, low, 4, "0.01"}, trialPlan{db, low, 4, "0.01"},
		trialPlan{da, medium, 4, "0.02"}, trialPlan{db, medium, 4, "0.02"},
		trialPlan{ha, low, 4, "0.01"}, trialPlan{hb, low, 0, "0.01"},
		trialPlan{ha, medium, 4, "0.02"}, trialPlan{hb, medium, 4, "0.02"},
	)
	gate := EvaluateGate(gateSpec(), run1)
	require.Equal(t, low, gate.Configuration[benchscenario.RouteChat])

	run2 := trials(trialPlan{ha, low, 4, "0.01"}, trialPlan{ha, medium, 4, "0.02"})
	confirmed := ConfirmGate(gate, run1, run2)
	assert.True(t, confirmed.Descriptions.Passed,
		"hb ran only in run 1, so it is left out of both sides: %s", confirmed.Descriptions.Detail)
	assert.Contains(t, confirmed.Descriptions.Detail, "1.000 → 1.000")
}
