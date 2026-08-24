package ratesimulationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cityRule  = pulid.MustNew("ragr_")
	stateRule = pulid.MustNew("ragr_")
	deadRule  = pulid.MustNew("ragr_")
)

func simulatedRules() []*rateagreement.RateAgreementRule {
	return []*rateagreement.RateAgreementRule{
		{ID: cityRule, Label: "chicago-atlanta", LaneKey: "CS:chicago>CS:atlanta"},
		{ID: stateRule, Label: "il-ga", LaneKey: "ST:il>ST:ga"},
		{ID: deadRule, Label: "alaska-only", LaneKey: "ST:ak>ST:ga"},
	}
}

func candidate(ruleID pulid.ID, label string, won bool) ratetypes.Candidate {
	return ratetypes.Candidate{
		RuleID:    ruleID.String(),
		RuleLabel: label,
		Won:       won,
	}
}

func traceWith(candidates ...ratetypes.Candidate) *ratetypes.Trace {
	return &ratetypes.Trace{Candidates: candidates}
}

// A rule that priced shipments is doing its job, and saying so is what makes
// the two failure states below mean something by contrast.
func TestCoverage_ARuleThatPricedShipmentsIsMarkedWon(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(candidate(cityRule, "chicago-atlanta", true)),
	})

	byRule := coverageByRule(coverage)

	require.Contains(t, byRule, cityRule)
	assert.Equal(t, ratesimulation.RuleOutcomeWon, byRule[cityRule].Outcome)
	assert.Equal(t, 1, byRule[cityRule].WonCount)
}

// A lane written for freight the organization does not move never matches
// anything. It is invisible in the revenue total and it is why somebody's
// carefully written tariff does nothing.
func TestCoverage_ARuleThatNeverMatchedIsMarkedNeverFired(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(candidate(cityRule, "chicago-atlanta", true)),
	})

	byRule := coverageByRule(coverage)

	require.Contains(t, byRule, deadRule)
	assert.Equal(t, ratesimulation.RuleOutcomeNeverFired, byRule[deadRule].Outcome)
	assert.Zero(t, byRule[deadRule].WonCount)
	assert.Zero(t, byRule[deadRule].LostCount)
}

// The subtler failure: a lane that matches every time and never wins, because
// something narrower already covers the same freight. It looks alive from the
// candidate list and is dead in the invoice.
func TestCoverage_ARuleThatAlwaysLosesIsMarkedLost(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
	})

	byRule := coverageByRule(coverage)

	require.Contains(t, byRule, stateRule)
	assert.Equal(t, ratesimulation.RuleOutcomeLost, byRule[stateRule].Outcome)
	assert.Equal(t, 2, byRule[stateRule].LostCount)
}

// Knowing a lane loses is half the answer. Naming what beats it is the half
// somebody can act on.
func TestCoverage_ALosingRuleNamesWhatBeatsItMostOften(t *testing.T) {
	t.Parallel()

	other := pulid.MustNew("ragr_")

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
		traceWith(
			candidate(other, "someone-else", true),
			candidate(stateRule, "il-ga", false),
		),
	})

	byRule := coverageByRule(coverage)

	require.NotNil(t, byRule[stateRule].LostTo)
	assert.Equal(t, cityRule, *byRule[stateRule].LostTo)
	assert.Equal(t, "chicago-atlanta", byRule[stateRule].LostToLabel)
}

// A rule that wins sometimes and loses others is working. Reporting it as lost
// because it also lost once would bury the rules that never win at all.
func TestCoverage_ARuleThatSometimesWinsIsNotReportedAsLost(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(candidate(stateRule, "il-ga", true)),
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
	})

	byRule := coverageByRule(coverage)

	assert.Equal(t, ratesimulation.RuleOutcomeWon, byRule[stateRule].Outcome)
	assert.Equal(t, 1, byRule[stateRule].WonCount)
	assert.Equal(t, 1, byRule[stateRule].LostCount)
}

// A trace naming a rule from some other contract is not this simulation's
// business, and reporting it would fill the coverage list with rules nobody
// asked about.
func TestCoverage_RulesOutsideTheSimulatedAgreementAreIgnored(t *testing.T) {
	t.Parallel()

	stranger := pulid.MustNew("ragr_")

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(candidate(stranger, "not-ours", true)),
	})

	assert.NotContains(t, coverageByRule(coverage), stranger)
	assert.Len(t, coverage, 3)
}

func TestCoverage_ASimulationOverNoShipmentsReportsEveryRuleAsNeverFired(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), nil)

	require.Len(t, coverage, 3)
	for _, row := range coverage {
		assert.Equal(t, ratesimulation.RuleOutcomeNeverFired, row.Outcome)
	}
}

// The coverage list is what somebody reads top to bottom, and the rules that
// did nothing are the reason they opened it.
func TestCoverage_IsOrderedWithTheProblemsFirst(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{
		traceWith(
			candidate(cityRule, "chicago-atlanta", true),
			candidate(stateRule, "il-ga", false),
		),
	})

	require.Len(t, coverage, 3)
	assert.Equal(t, ratesimulation.RuleOutcomeNeverFired, coverage[0].Outcome)
	assert.Equal(t, ratesimulation.RuleOutcomeLost, coverage[1].Outcome)
	assert.Equal(t, ratesimulation.RuleOutcomeWon, coverage[2].Outcome)
}

// A trace with no candidates at all is a shipment no rule covered. It is
// ordinary, and it must not be mistaken for a rule having fired.
func TestCoverage_ATraceWithNoCandidatesChangesNothing(t *testing.T) {
	t.Parallel()

	coverage := buildCoverage(simulatedRules(), []*ratetypes.Trace{traceWith(), nil})

	for _, row := range coverage {
		assert.Equal(t, ratesimulation.RuleOutcomeNeverFired, row.Outcome)
	}
}

func coverageByRule(
	coverage []*ratesimulation.RuleCoverage,
) map[pulid.ID]*ratesimulation.RuleCoverage {
	byRule := make(map[pulid.ID]*ratesimulation.RuleCoverage, len(coverage))
	for _, row := range coverage {
		byRule[row.RuleID] = row
	}

	return byRule
}
