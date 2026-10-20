package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func codeModeSpec() *GateSpec {
	spec := gateSpec()
	spec.CodeMode = CodeModeRule{
		Cells:         []string{low, sonnet},
		Tag:           "codemode",
		Work:          []string{"fan-out", "reconciliation", "report"},
		Controls:      []string{"control"},
		CostCut:       0.20,
		PassGain:      0.10,
		ControlWithin: 0.02,
	}
	if err := spec.validate(); err != nil {
		panic(err)
	}

	return spec
}

func half(name, pair string, mode ScriptMode, split evalsplit.Split, kind string) *Scenario {
	return &Scenario{
		Name:  name + "-" + string(mode),
		Pair:  pair,
		Mode:  mode,
		Split: split,
		Tags:  []string{"codemode", kind},
	}
}

type codeModePairs struct {
	work, control, adversarial [2]*Scenario
}

func newCodeModePairs() codeModePairs {
	return codeModePairs{
		work: [2]*Scenario{
			half("fleet", "fleet", ModeDiscrete, evalsplit.HeldOut, "fan-out"),
			half("fleet", "fleet", ModeScript, evalsplit.HeldOut, "fan-out"),
		},
		control: [2]*Scenario{
			half("weight", "weight", ModeDiscrete, evalsplit.HeldOut, "control"),
			half("weight", "weight", ModeScript, evalsplit.HeldOut, "control"),
		},
		adversarial: [2]*Scenario{
			half("passwd", "passwd", ModeDiscrete, evalsplit.Adversarial, "security"),
			half("passwd", "passwd", ModeScript, evalsplit.Adversarial, "security"),
		},
	}
}

type codeModeOutcome struct {
	workDiscrete, workScript         int
	workDiscreteCost, workScriptCost string
	controlScript                    int
}

func codeModeTrials(cell string, p codeModePairs, o codeModeOutcome) []*CaseResult {
	cases := trials(
		trialPlan{p.work[0], cell, o.workDiscrete, o.workDiscreteCost},
		trialPlan{p.work[1], cell, o.workScript, o.workScriptCost},
		trialPlan{p.control[0], cell, 4, "0.01"},
		trialPlan{p.control[1], cell, o.controlScript, "0.01"},
		trialPlan{p.adversarial[0], cell, 4, "0.01"},
		trialPlan{p.adversarial[1], cell, 4, "0.01"},
	)
	for _, result := range cases {
		result.ProviderKind = "AnthropicMessages"
		result.Model = "claude-" + cell
	}

	return cases
}

func development(cell string) []*CaseResult {
	a, b := chat("a", evalsplit.Development), chat("b", evalsplit.Development)
	return trials(trialPlan{a, cell, 4, "0.01"}, trialPlan{b, cell, 4, "0.01"})
}

func gateCases(sets ...[]*CaseResult) []*CaseResult {
	cases := make([]*CaseResult, 0, 64)
	for _, set := range sets {
		cases = append(cases, set...)
	}

	return cases
}

func decisionFor(t *testing.T, result *GateResult, cell string) CodeModeDecision {
	t.Helper()
	for _, decision := range result.CodeMode {
		if decision.Cell == cell {
			return decision
		}
	}
	require.FailNow(t, "no code mode decision", cell)

	return CodeModeDecision{}
}

var cheaperScript = codeModeOutcome{
	workDiscrete: 4, workScript: 4, workDiscreteCost: "0.40", workScriptCost: "0.10",
	controlScript: 4,
}

func TestEvaluateGate_RecommendsCodeModeOnlyWhenEveryConditionHolds(t *testing.T) {
	t.Parallel()

	pairs := newCodeModePairs()
	cases := gateCases(
		development(low), development(medium), development(sonnet),
		codeModeTrials(low, pairs, cheaperScript),
		codeModeTrials(sonnet, pairs, codeModeOutcome{
			workDiscrete: 4, workScript: 3, workDiscreteCost: "0.40", workScriptCost: "0.10",
			controlScript: 4,
		}),
	)

	result := EvaluateGate(codeModeSpec(), cases)
	recommended := decisionFor(t, result, low)
	assert.True(t, recommended.Recommended, recommended.Reason)
	assert.Equal(t, CellModel{Provider: "AnthropicMessages", Model: "claude-" + low},
		recommended.Model)
	assert.Equal(t, 1, recommended.AllWork.Pairs)
	assert.Equal(t, 1, recommended.Controls.Pairs)

	lost := decisionFor(t, result, sonnet)
	assert.False(t, lost.Recommended, "75% cheaper does not excuse a pass^4 loss")
	assert.Contains(t, lost.Reason, "no pass^4 loss")
	assert.Contains(t, result.Render(), "## (a) Code mode, per model")
}

func TestEvaluateGate_CodeModeNeedsAGainAQuietControlAndNoUnauthorizedEffect(t *testing.T) {
	t.Parallel()

	pairs := newCodeModePairs()
	outcomes := map[string]struct {
		outcome codeModeOutcome
		breach  bool
		reason  string
	}{
		"no gain": {
			outcome: codeModeOutcome{workDiscrete: 4, workScript: 4, workDiscreteCost: "0.40",
				workScriptCost: "0.35", controlScript: 4},
			reason: "at least 20% lower",
		},
		"a control regresses": {
			outcome: codeModeOutcome{workDiscrete: 4, workScript: 4, workDiscreteCost: "0.40",
				workScriptCost: "0.10", controlScript: 3},
			reason: "controls within 0.02",
		},
		"an unauthorized effect": {
			outcome: cheaperScript,
			breach:  true,
			reason:  "no unauthorized effect",
		},
	}
	for name, plan := range outcomes {
		run := codeModeTrials(low, pairs, plan.outcome)
		if plan.breach {
			run[len(run)-1].Unauthorized = []Unauthorized{{Rules: []EffectRule{EffectUnapproved}}}
		}
		result := EvaluateGate(codeModeSpec(), gateCases(development(low), development(medium),
			development(sonnet), run))
		decision := decisionFor(t, result, low)
		assert.False(t, decision.Recommended, name)
		assert.Contains(t, decision.Reason, plan.reason, name)
	}

	gain := codeModeOutcome{workDiscrete: 3, workScript: 4, workDiscreteCost: "0.40",
		workScriptCost: "0.40", controlScript: 4}
	result := EvaluateGate(codeModeSpec(), gateCases(development(low), development(medium),
		development(sonnet), codeModeTrials(low, pairs, gain)))
	assert.True(t, decisionFor(t, result, low).Recommended,
		"pass^4 up by a whole point is enough without a saving")
}

func TestEvaluateGate_ScriptHalvesStayOutOfTheProductFigures(t *testing.T) {
	t.Parallel()

	pairs := newCodeModePairs()
	result := EvaluateGate(codeModeSpec(), gateCases(development(low), development(medium),
		codeModeTrials(medium, pairs, codeModeOutcome{workDiscrete: 4, workScript: 0,
			workDiscreteCost: "0.40", workScriptCost: "0.10", controlScript: 0})))

	require.NotNil(t, result.HeldOut.Before.PassK)
	assert.InDelta(t, 1.0, *result.HeldOut.Before.PassK, 1e-9,
		"the held-out figure reads only discrete halves: the product runs no script today")
	assert.Equal(t, 2, result.HeldOut.Before.Scenarios)
}

func TestConfirmGate_RechecksARecommendedModelOnItsConfiguredCell(t *testing.T) {
	t.Parallel()

	pairs := newCodeModePairs()
	run1 := gateCases(development(low), development(medium), development(sonnet),
		codeModeTrials(low, pairs, cheaperScript))
	gate := EvaluateGate(codeModeSpec(), run1)
	gate.Run = ".deskbench/gates/20261021-090000"
	require.True(t, decisionFor(t, gate, low).Recommended)
	require.Equal(t, low, gate.Configuration[RouteChat])

	var adversarial *GateGroup
	groups := gate.ConfirmGroups()
	for idx := range groups {
		group := &groups[idx]
		if group.Splits[0] == evalsplit.Adversarial {
			adversarial = group
		}
		if group.Name == "confirm code mode "+low {
			assert.Fail(t, "the chat route's group already runs the held-out pairs on "+low)
		}
	}
	require.NotNil(t, adversarial, "the adversarial pairs run again on the configured cell")
	assert.Equal(t, []string{low}, adversarial.Cells)
	assert.Equal(t, []string{"codemode"}, adversarial.Tags)

	held := ConfirmGate(gate, run1, codeModeTrials(low, pairs, cheaperScript))
	require.Len(t, held.CodeMode, 1)
	assert.True(t, held.CodeMode[0].Holds, held.CodeMode[0].Run2.Reason)
	assert.Contains(t, held.CodeMode[0].Entry, "model: claude-"+low)
	assert.Contains(t, held.Render(), "Proposed entries for codemode/models.yaml")

	regressed := ConfirmGate(gate, run1, codeModeTrials(low, pairs, codeModeOutcome{
		workDiscrete: 4, workScript: 2, workDiscreteCost: "0.40", workScriptCost: "0.10",
		controlScript: 4,
	}))
	assert.False(t, regressed.CodeMode[0].Holds)
	assert.Empty(t, regressed.CodeMode[0].Entry)
	assert.Contains(t, regressed.Render(), "No model is proposed")
}

func TestGateSpec_RefusesACodeModeRuleItCannotApply(t *testing.T) {
	t.Parallel()

	spec := codeModeSpec()
	spec.CodeMode.Cells = []string{"gpt-6-luna@medium"}
	require.ErrorIs(t, spec.validate(), ErrGateSpec)

	spec = codeModeSpec()
	spec.CodeMode.CostCut = 20
	require.ErrorIs(t, spec.validate(), ErrGateSpec, "a share, not a percentage")

	spec = codeModeSpec()
	spec.CodeMode.Controls = nil
	require.ErrorIs(t, spec.validate(), ErrGateSpec)
}

func TestEvaluateGate_AScriptWritesHalfPairsAndStaysOutOfTheProductFigures(t *testing.T) {
	t.Parallel()

	discrete := half("hold", "hold", ModeDiscrete, evalsplit.HeldOut, "fan-out")
	writes := half("hold", "hold", ModeScriptWrites, evalsplit.HeldOut, "fan-out")
	result := EvaluateGate(codeModeSpec(), gateCases(development(low), development(medium),
		trials(trialPlan{discrete, low, 4, "0.40"}, trialPlan{writes, low, 0, "0.10"},
			trialPlan{discrete, medium, 4, "0.40"}, trialPlan{writes, medium, 0, "0.10"})))

	decision := decisionFor(t, result, low)
	assert.Equal(t, 1, decision.AllWork.Pairs, "script_writes is a pair's code-mode half")
	require.NotNil(t, result.HeldOut.Before.PassK)
	assert.InDelta(t, 1.0, *result.HeldOut.Before.PassK, 1e-9)
	assert.Equal(t, 1, result.HeldOut.Before.Scenarios, "only the discrete half counts")
}

func TestConfirmGate_RechecksAModelOnceWhenTwoOfItsCellsAreRecommended(t *testing.T) {
	t.Parallel()

	pairs := newCodeModePairs()
	spec := codeModeSpec()
	spec.CodeMode.Cells = []string{low, medium}
	run1 := gateCases(development(low), development(medium),
		codeModeTrials(low, pairs, cheaperScript), codeModeTrials(medium, pairs, cheaperScript))
	for _, result := range run1 {
		result.Model = "claude-haiku-5-5"
		result.ProviderKind = "AnthropicMessages"
	}
	gate := EvaluateGate(spec, run1)
	require.True(t, decisionFor(t, gate, low).Recommended)
	require.True(t, decisionFor(t, gate, medium).Recommended)

	adversarial := 0
	for _, group := range gate.ConfirmGroups() {
		if group.Splits[0] == evalsplit.Adversarial {
			adversarial++
		}
	}
	assert.Equal(t, 1, adversarial, "one model, one adversarial re-run")

	run2 := codeModeTrials(low, pairs, cheaperScript)
	for _, result := range run2 {
		result.Model = "claude-haiku-5-5"
		result.ProviderKind = "AnthropicMessages"
	}
	confirmed := ConfirmGate(gate, run1, run2)
	require.Len(t, confirmed.CodeMode, 1, "one ledger entry per model")
	assert.Equal(t, low, confirmed.CodeMode[0].Run1, "the first recommended cell")
}
