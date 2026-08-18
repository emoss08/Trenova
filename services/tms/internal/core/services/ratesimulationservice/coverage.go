package ratesimulationservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// buildCoverage works out what happened to each of the simulated agreement's
// rules across a whole run.
//
// This is the half of a simulation the revenue total cannot give. A rule can
// fail in two quiet ways that look identical from the money: it never matched
// anything, because it was written for freight the organization does not move;
// or it matched every time and never won, because something narrower already
// covers the same lane. Both are why somebody's carefully written tariff does
// nothing, and neither shows up as a number.
func buildCoverage(
	rules []*rateagreement.RateAgreementRule,
	traces []*ratetypes.Trace,
) []*ratesimulation.RuleCoverage {
	coverage := make(map[pulid.ID]*ratesimulation.RuleCoverage, len(rules))
	order := make([]pulid.ID, 0, len(rules))

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		coverage[rule.ID] = &ratesimulation.RuleCoverage{
			RuleID:  rule.ID,
			Label:   rule.Label,
			LaneKey: rule.LaneKey,
			Outcome: ratesimulation.RuleOutcomeNeverFired,
		}
		order = append(order, rule.ID)
	}

	// beatenBy counts, per losing rule, which rule won instead. Naming what
	// beats a lane is the half of "this never wins" somebody can act on.
	beatenBy := make(map[pulid.ID]map[pulid.ID]int, len(rules))
	labels := make(map[pulid.ID]string, len(rules))

	for _, trace := range traces {
		if trace == nil {
			continue
		}

		tally(trace, coverage, beatenBy, labels)
	}

	rows := make([]*ratesimulation.RuleCoverage, 0, len(order))
	for _, ruleID := range order {
		row := coverage[ruleID]
		settleOutcome(row)
		nameTheWinner(row, beatenBy[ruleID], labels)
		rows = append(rows, row)
	}

	sortByUrgency(rows)

	return rows
}

func tally(
	trace *ratetypes.Trace,
	coverage map[pulid.ID]*ratesimulation.RuleCoverage,
	beatenBy map[pulid.ID]map[pulid.ID]int,
	labels map[pulid.ID]string,
) {
	winner := trace.Winner()

	for i := range trace.Candidates {
		candidate := &trace.Candidates[i]
		ruleID := pulid.ID(candidate.RuleID)
		labels[ruleID] = candidate.RuleLabel

		row, ok := coverage[ruleID]
		if !ok {
			// A rule from some other contract. It is not this simulation's
			// business, and reporting it would fill the list with rules nobody
			// asked about.
			continue
		}

		if candidate.Won {
			row.WonCount++
			continue
		}

		row.LostCount++

		if winner == nil || winner.RuleID == "" {
			continue
		}

		if beatenBy[ruleID] == nil {
			beatenBy[ruleID] = make(map[pulid.ID]int, 2)
		}

		beatenBy[ruleID][pulid.ID(winner.RuleID)]++
	}
}

// settleOutcome decides what a rule's run amounts to.
//
// Winning even once makes a rule working. Reporting it as lost because it also
// lost sometimes would bury the rules that never win at all, which are the ones
// somebody opened this list to find.
func settleOutcome(row *ratesimulation.RuleCoverage) {
	switch {
	case row.WonCount > 0:
		row.Outcome = ratesimulation.RuleOutcomeWon
	case row.LostCount > 0:
		row.Outcome = ratesimulation.RuleOutcomeLost
	default:
		row.Outcome = ratesimulation.RuleOutcomeNeverFired
	}
}

func nameTheWinner(
	row *ratesimulation.RuleCoverage,
	beaten map[pulid.ID]int,
	labels map[pulid.ID]string,
) {
	if row.Outcome != ratesimulation.RuleOutcomeLost || len(beaten) == 0 {
		return
	}

	best := pulid.Nil
	bestCount := 0

	for ruleID, count := range beaten {
		// The tie-break is the rule id, so the same simulation names the same
		// rule on every run.
		if count > bestCount || (count == bestCount && ruleID.String() < best.String()) {
			best, bestCount = ruleID, count
		}
	}

	if best.IsNil() {
		return
	}

	row.LostTo = &best
	row.LostToLabel = labels[best]
}

// sortByUrgency puts the rules that did nothing first.
//
// Somebody opens this list because a lane is not pricing the way they expected.
// The rules that never fired are the likeliest answer, the ones that always
// lost are the next, and the working ones are there for completeness.
func sortByUrgency(rows []*ratesimulation.RuleCoverage) {
	slices.SortStableFunc(rows, func(a, b *ratesimulation.RuleCoverage) int {
		if by := cmp.Compare(urgency(a.Outcome), urgency(b.Outcome)); by != 0 {
			return by
		}

		return cmp.Compare(a.RuleID.String(), b.RuleID.String())
	})
}

func urgency(outcome ratesimulation.RuleOutcome) int {
	switch outcome {
	case ratesimulation.RuleOutcomeNeverFired:
		return 0
	case ratesimulation.RuleOutcomeLost:
		return 1
	case ratesimulation.RuleOutcomeWon:
		return 2
	default:
		return 3
	}
}
