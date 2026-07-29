package hosprojection

import (
	"fmt"

	"github.com/emoss08/trenova/shared/timeutils"
)

// Project determines the best legal way for a driver to take a trip departing at
// in.Departure, evaluating rest strategies ordered from least to most disruptive:
// drive on current clocks, complete a qualifying split sleeper pairing, take a full
// 10-hour reset, or take a 34-hour cycle restart. Every applicable strategy is
// simulated and the one delivering the earliest arrival wins, so a split pairing that
// avoids an en-route reset beats limping along on depleted clocks. When none works,
// the result carries the binding clock from the most-rested strategy that was
// actually available before departure.
//
//nolint:gocritic // a value input keeps the projection API pure; call frequency is low
func Project(in Input) Result {
	if in.Departure < in.Now {
		in.Departure = in.Now
	}

	e := newEngine(&in)
	candidates := make([]strategyCandidate, 0, 4)
	if candidate, ok := e.currentClocks(in.Departure); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := e.splitSleeper(in.Departure); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := e.tenHourReset(in.Departure); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := e.restart34(in.Departure); ok {
		candidates = append(candidates, candidate)
	}

	if len(candidates) == 0 {
		return infeasibleResult(LimiterShift,
			"No rest plan can complete before the departure time")
	}

	var bestCandidate strategyCandidate
	var bestPlan TripPlan
	feasible := false
	var lastPlan TripPlan
	var lastLimiter Limiter
	var lastCandidate strategyCandidate
	for _, candidate := range candidates {
		plan, limiter := PlanTrip(in.TripDriveMs, candidate.clocks, in.Limits)
		if limiter != LimiterNone {
			lastPlan = plan
			lastLimiter = limiter
			lastCandidate = candidate
			continue
		}
		if !feasible || plan.TotalMs < bestPlan.TotalMs {
			bestCandidate = candidate
			bestPlan = plan
			feasible = true
		}
	}

	if feasible {
		bestPlan.Arrival = in.Departure + bestPlan.TotalMs/1000
		return e.feasibleResult(bestCandidate, bestPlan)
	}

	result := infeasibleResult(lastLimiter, infeasibleDetail(lastCandidate, lastLimiter))
	result.DriveAvailableMs = lastCandidate.clocks.DriveMs
	result.ShiftAvailableMs = lastCandidate.clocks.ShiftMs
	result.CycleAvailableMs = lastCandidate.clocks.CycleMs
	result.Trip = lastPlan
	return result
}

func (e *engine) feasibleResult(candidate strategyCandidate, plan TripPlan) Result {
	return Result{
		Feasible:          true,
		Strategy:          candidate.strategy,
		Limiter:           LimiterNone,
		DriveAvailableMs:  candidate.clocks.DriveMs,
		ShiftAvailableMs:  candidate.clocks.ShiftMs,
		CycleAvailableMs:  candidate.clocks.CycleMs,
		RestStartDeadline: candidate.restStartDeadline,
		Trip:              plan,
		Detail:            e.feasibleDetail(candidate, plan),
	}
}

func infeasibleResult(limiter Limiter, detail string) Result {
	return Result{
		Feasible: false,
		Strategy: "",
		Limiter:  limiter,
		Detail:   detail,
	}
}

func (e *engine) feasibleDetail(candidate strategyCandidate, plan TripPlan) string {
	margin := timeutils.FormatDurationMs(plan.MarginMs)

	var lead string
	switch candidate.strategy {
	case StrategyCurrentClocks:
		lead = "Feasible on current clocks"
	case StrategySplitSleeper:
		lead = "Feasible with a split sleeper berth pairing"
	case StrategyTenHourReset:
		lead = "Feasible after a 10-hour reset"
	case StrategyRestart34:
		lead = "Feasible after a 34-hour cycle restart"
	}

	detail := fmt.Sprintf("%s; %s of clock left after the trip", lead, margin)
	if candidate.restStartDeadline > e.in.Now {
		detail += fmt.Sprintf(", rest must start within %s",
			timeutils.FormatDurationMs((candidate.restStartDeadline-e.in.Now)*1000))
	}
	if plan.Breaks > 0 || plan.Resets > 0 {
		detail += fmt.Sprintf(" (%s en route)", enRouteRests(plan))
	}
	if !e.hasTimeline() && candidate.strategy != StrategyCurrentClocks {
		detail += "; no duty log history, projected from clocks only"
	}
	return detail
}

func enRouteRests(plan TripPlan) string {
	switch {
	case plan.Breaks > 0 && plan.Resets > 0:
		return fmt.Sprintf("%s of break and %d full reset(s)",
			timeutils.FormatDurationMs(int64(plan.Breaks)*breakRestMs), plan.Resets)
	case plan.Resets > 0:
		return fmt.Sprintf("%d full reset(s)", plan.Resets)
	default:
		return fmt.Sprintf(
			"%s of break",
			timeutils.FormatDurationMs(int64(plan.Breaks)*breakRestMs),
		)
	}
}

func infeasibleDetail(candidate strategyCandidate, limiter Limiter) string {
	switch limiter {
	case LimiterCycle:
		if candidate.strategy == StrategyRestart34 {
			return "60/70-hour cycle exhausted for this trip even after a 34-hour restart"
		}
		return "60/70-hour cycle exhausted; a 34-hour restart cannot complete before departure"
	case LimiterDrive:
		return "Not enough drive hours for this trip under any rest plan that fits before departure"
	case LimiterShift:
		return "The 14-hour window cannot cover this trip under any rest plan" +
			" that fits before departure"
	case LimiterBreak:
		return "Required 30-minute break cannot be satisfied for this trip"
	case LimiterNone:
		return ""
	default:
		return "Trip cannot be completed within hours-of-service limits"
	}
}
