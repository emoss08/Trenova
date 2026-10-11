package aicorrection

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	TrendWeeks             = 12
	DriftBaselineWeeks     = 4
	MinDriftWeekFields     = 100
	MinDriftBaselineFields = 200
	DriftPoints            = 5

	secondsPerWeek = 7 * timeutils.SecondsPerDay
)

type WeekTotal struct {
	ProviderID  pulid.ID `bun:"provider_id"`
	WeekStart   int64    `bun:"week_start"`
	Corrections int      `bun:"corrections"`
	Scored      int      `bun:"scored"`
	Correct     int      `bun:"correct"`
}

type WeekAccuracy struct {
	WeekStart   int64
	Corrections int
	Scored      int
	Correct     int
	Accuracy    float64
}

func (w *WeekAccuracy) add(total *WeekTotal) {
	w.Corrections += total.Corrections
	w.Scored += total.Scored
	w.Correct += total.Correct
	w.Accuracy = Accuracy(w.Correct, w.Scored)
}

type ProviderTrend struct {
	ProviderID pulid.ID
	Weeks      []WeekAccuracy
	Checked    WeekAccuracy
	Baseline   WeekAccuracy
	DropPoints float64
	Comparable bool
	Drifting   bool
}

type TrendWindow struct {
	Weeks         []int64
	CheckedWeek   int64
	BaselineStart int64
}

func NewTrendWindow(now int64) TrendWindow {
	current := timeutils.WeekStartUTC(now)
	weeks := make([]int64, TrendWeeks)
	for i := range weeks {
		weeks[i] = current - int64(TrendWeeks-1-i)*secondsPerWeek
	}
	checked := current - secondsPerWeek

	return TrendWindow{
		Weeks:         weeks,
		CheckedWeek:   checked,
		BaselineStart: checked - DriftBaselineWeeks*secondsPerWeek,
	}
}

func (w TrendWindow) Since() int64 { return w.Weeks[0] }

func (w TrendWindow) inBaseline(weekStart int64) bool {
	return weekStart >= w.BaselineStart && weekStart < w.CheckedWeek
}

func BuildProviderTrends(window TrendWindow, totals []WeekTotal) []ProviderTrend {
	index := make(map[int64]int, len(window.Weeks))
	for i, week := range window.Weeks {
		index[week] = i
	}

	byProvider := map[pulid.ID]*ProviderTrend{}
	order := make([]pulid.ID, 0)
	for i := range totals {
		total := &totals[i]
		slot, ok := index[total.WeekStart]
		if !ok || total.ProviderID.IsNil() {
			continue
		}
		trend, seen := byProvider[total.ProviderID]
		if !seen {
			trend = newProviderTrend(total.ProviderID, window)
			byProvider[total.ProviderID] = trend
			order = append(order, total.ProviderID)
		}
		trend.Weeks[slot].add(total)
		switch {
		case total.WeekStart == window.CheckedWeek:
			trend.Checked.add(total)
		case window.inBaseline(total.WeekStart):
			trend.Baseline.add(total)
		}
	}

	trends := make([]ProviderTrend, 0, len(order))
	for _, id := range order {
		trend := byProvider[id]
		trend.judge()
		trends = append(trends, *trend)
	}
	slices.SortFunc(trends, func(a, b ProviderTrend) int {
		if c := cmp.Compare(b.Checked.Scored, a.Checked.Scored); c != 0 {
			return c
		}
		if c := cmp.Compare(recentScored(&b), recentScored(&a)); c != 0 {
			return c
		}
		return cmp.Compare(a.ProviderID, b.ProviderID)
	})

	return trends
}

func newProviderTrend(providerID pulid.ID, window TrendWindow) *ProviderTrend {
	weeks := make([]WeekAccuracy, len(window.Weeks))
	for i, week := range window.Weeks {
		weeks[i].WeekStart = week
	}

	return &ProviderTrend{
		ProviderID: providerID,
		Weeks:      weeks,
		Checked:    WeekAccuracy{WeekStart: window.CheckedWeek},
		Baseline:   WeekAccuracy{WeekStart: window.BaselineStart},
	}
}

func (t *ProviderTrend) judge() {
	t.Comparable = t.Checked.Scored >= MinDriftWeekFields &&
		t.Baseline.Scored >= MinDriftBaselineFields
	if !t.Comparable {
		return
	}
	t.DropPoints = (t.Baseline.Accuracy - t.Checked.Accuracy) * 100
	t.Drifting = intutils.RatioLeadExceedsPoints(
		t.Baseline.Correct, t.Baseline.Scored,
		t.Checked.Correct, t.Checked.Scored,
		DriftPoints,
	)
}

func recentScored(t *ProviderTrend) int {
	total := 0
	for i := range t.Weeks {
		total += t.Weeks[i].Scored
	}
	return total
}
