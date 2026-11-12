package demandforecast

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
)

const (
	BacktestWeeks     = 8
	MinHistoryWeeks   = 8
	MinHistoryLoads   = 200
	RefitAfterSeconds = int64(3600)
	MinSkill          = 0.05
	IntervalLow       = 0.1
	IntervalHigh      = 0.9
	minTrainingWeeks  = 4
	minScoredMean     = 0.25
)

type slotKey struct {
	cell int64
	slot int16
}

func weeklyActuals(counts []LoadCount) map[int64]map[slotKey]int32 {
	byWeek := map[int64]map[slotKey]int32{}
	for _, count := range counts {
		week := byWeek[count.Week]
		if week == nil {
			week = map[slotKey]int32{}
			byWeek[count.Week] = week
		}
		week[slotKey{cell: count.Cell, slot: count.Slot}] += count.Loads
	}

	return byWeek
}

type score struct {
	cases      int
	inInterval int
	loads      int64
	absError   float64
	naiveError float64
}

func (s *score) add(predicted Count, actual, previous int32) {
	s.cases++
	s.loads += int64(actual)
	s.absError += math.Abs(float64(actual) - predicted.Mean)
	s.naiveError += math.Abs(float64(actual - previous))
	low, high := predicted.Quantile(IntervalLow), predicted.Quantile(IntervalHigh)
	if int(actual) >= low && int(actual) <= high {
		s.inInterval++
	}
}

func Backtest(h *History) *Quality {
	current := dispatchmarket.WeekOf(h.Now)
	quality := &Quality{FittedAt: h.Now}

	weeks := map[int64]struct{}{}
	for _, count := range h.Counts {
		if count.Week < current && count.Loads > 0 {
			weeks[count.Week] = struct{}{}
			quality.HistoryLoads += int64(count.Loads)
		}
	}
	quality.HistoryWeeks = int32(len(weeks))

	actuals := weeklyActuals(h.Counts)
	var total score
	for test := current - BacktestWeeks; test < current; test++ {
		w := windowOf(h.Counts, test)
		if w.weeks() < minTrainingWeeks {
			continue
		}
		scoreWeek(&total, aggregateOf(h.Counts, w), w, actuals[test], actuals[test-1])
		quality.WeeksTested++
	}

	quality.LoadsTested = total.loads
	if total.cases > 0 {
		quality.IntervalCoverage = float64(total.inInterval) / float64(total.cases)
		quality.MAE = total.absError / float64(total.cases)
		quality.NaiveMAE = total.naiveError / float64(total.cases)
		if quality.NaiveMAE > 0 {
			quality.Skill = 1 - quality.MAE/quality.NaiveMAE
		}
	}

	switch {
	case quality.HistoryWeeks < MinHistoryWeeks || quality.HistoryLoads < MinHistoryLoads ||
		quality.WeeksTested == 0:
		quality.Status = StatusInsufficientHistory
	case quality.Skill <= MinSkill:
		quality.Status = StatusNoSkill
	default:
		quality.Status = StatusReady
	}

	return quality
}

func scoreWeek(
	total *score,
	agg *aggregate,
	w window,
	actual, previous map[slotKey]int32,
) {
	for _, cell := range sortedCells(agg.areas) {
		area := agg.areas[cell]
		for slot := range slotsPerWeek {
			key := slotKey{cell: cell, slot: int16(slot)}
			predicted := agg.predict(area, w, slot)
			if actual[key] == 0 && previous[key] == 0 && predicted.Mean < minScoredMean {
				continue
			}
			total.add(predicted, actual[key], previous[key])
		}
	}
}

func (a *aggregate) predict(area *series, w window, slot int) Count {
	alpha, beta := posterior(area.total/w.total*a.tenantShare(slot), area.slot[slot], w.total)

	return GammaPoisson(alpha, beta)
}
