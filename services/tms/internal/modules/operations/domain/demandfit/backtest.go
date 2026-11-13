package demandfit

import (
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
)

const (
	BacktestWeeks      = 8
	MinHistoryWeeks    = 8
	MinHistoryLoads    = 200
	MinSkill           = 0.05
	MaxDispersion      = 25.0
	LevelWeeks         = 2
	LevelPriorSpread   = 0.1
	MinLevel           = 0.5
	MaxLevel           = 2.0
	HolidayPriorFactor = 0.5
	HolidayPriorLoads  = 20.0
	MinHolidayFactor   = 0.05
	MaxHolidayFactor   = 1.5
	HolidayHorizonDays = 14
	minTrainingWeeks   = 4
	minScoredMean      = 0.25
	daysPerWeek        = 7
)

type slotKey struct {
	cell int64
	slot int16
}

type weekActuals struct {
	bySlot map[slotKey]int32
	totals [slotsPerWeek]int32
}

func weeklyActuals(counts []LoadCount) map[int64]*weekActuals {
	byWeek := map[int64]*weekActuals{}
	for _, count := range counts {
		if count.Slot < 0 || int(count.Slot) >= slotsPerWeek {
			continue
		}
		week := byWeek[count.Week]
		if week == nil {
			week = &weekActuals{bySlot: map[slotKey]int32{}}
			byWeek[count.Week] = week
		}
		week.bySlot[slotKey{cell: count.Cell, slot: count.Slot}] += count.Loads
		week.totals[count.Slot] += count.Loads
	}

	return byWeek
}

func (w *weekActuals) of(key slotKey) int32 {
	if w == nil {
		return 0
	}

	return w.bySlot[key]
}

func (w *weekActuals) total(slot int) int32 {
	if w == nil {
		return 0
	}

	return w.totals[slot]
}

type trainedWeek struct {
	week   int64
	agg    *aggregate
	cells  []int64
	level  float64
	scale  [slotsPerWeek]float64
	base   [slotsPerWeek]demandforecast.Count
	totals [slotsPerWeek]demandforecast.Count
}

func (m *trainedWeek) predict(area *series, slot int) demandforecast.Count {
	alpha, beta := m.agg.areaRate(area, slot)

	return demandforecast.GammaPoisson(alpha, beta/m.scale[slot])
}

type score struct {
	cases int
	loads int64
	miss  float64
	naive float64
}

func (s *score) add(predicted demandforecast.Count, actual, previous int32) {
	s.loads += int64(actual)
	s.miss += predicted.Score(int(actual))
	s.naive += math.Abs(float64(actual - previous))
	if actual != 0 || previous != 0 || predicted.Mean >= minScoredMean {
		s.cases++
	}
}

type calibration struct {
	squared  float64
	variance float64
}

func (c *calibration) add(predicted demandforecast.Count, actual int32) {
	miss := float64(actual) - predicted.Mean
	c.squared += miss * miss
	c.variance += predicted.Variance
}

func (c *calibration) dispersion() float64 {
	if c.variance <= 0 {
		return 1
	}

	return math.Min(math.Max(c.squared/c.variance, 1), MaxDispersion)
}

type holidayEvidence struct {
	loads    float64
	expected float64
}

func (e holidayEvidence) factor(dispersion float64) float64 {
	factor := (HolidayPriorLoads*HolidayPriorFactor + e.loads/dispersion) /
		(HolidayPriorLoads + e.expected/dispersion)

	return math.Min(math.Max(factor, MinHolidayFactor), MaxHolidayFactor)
}

type training struct {
	h           *History
	actuals     map[int64]*weekActuals
	weeks       map[int64]*trainedWeek
	days        calibration
	weekly      calibration
	weeklyAt    map[int64]calibration
	holidays    holidayEvidence
	dispersions map[int64]float64
	total       score
	tested      int
	held        int
}

type Training struct {
	Fitted  *demandforecast.Fitted
	Quality *demandforecast.Quality
	Days    []*demandforecast.Day
}

func Evaluate(h *History) (*demandforecast.Quality, []*demandforecast.Day) {
	trained := Train(h)

	return trained.Quality, trained.Days
}

func Train(h *History) *Training {
	current := h.Grid.Week(h.Now)
	quality := &demandforecast.Quality{
		FittedAt:      h.Now,
		Dispersion:    1,
		Level:         1,
		HolidayFactor: HolidayPriorFactor,
		Timezone:      h.Grid.Timezone(),
		Holidays:      h.upcomingHolidays(),
	}
	weeks := map[int64]struct{}{}
	for _, count := range h.Counts {
		if count.Week < current && count.Loads > 0 {
			weeks[count.Week] = struct{}{}
			quality.HistoryLoads += int64(count.Loads)
		}
	}
	quality.HistoryWeeks = int32(len(weeks))

	t := &training{
		h:           h,
		actuals:     weeklyActuals(h.Counts),
		weeks:       map[int64]*trainedWeek{},
		weeklyAt:    map[int64]calibration{},
		dispersions: map[int64]float64{},
	}
	first := current - BacktestWeeks
	for _, week := range t.visits(first, current) {
		t.weeklyAt[week] = t.weekly
		model := t.train(week)
		if week >= first && week < current {
			t.dispersions[week] = t.days.dispersion()
			if model != nil {
				t.test(model)
				quality.WeeksTested++
			}
		}
		if model != nil && week < current {
			t.observeHolidays(model)
		}
	}
	quality.Dispersion = t.days.dispersion()
	quality.HolidayFactor = t.holidays.factor(quality.Dispersion)
	t.dispersions[current] = quality.Dispersion
	model := t.weeks[current]
	if model != nil {
		quality.Level = model.level
	}

	quality.LoadsTested = t.total.loads
	if t.tested > 0 {
		quality.IntervalCoverage = float64(t.held) / float64(t.tested)
	}
	if t.total.cases > 0 {
		quality.MAE = t.total.miss / float64(t.total.cases)
		quality.NaiveMAE = t.total.naive / float64(t.total.cases)
		if t.total.naive > 0 {
			quality.Skill = 1 - t.total.miss/t.total.naive
		}
	}

	switch {
	case quality.HistoryWeeks < MinHistoryWeeks || quality.HistoryLoads < MinHistoryLoads ||
		quality.WeeksTested == 0:
		quality.Status = demandforecast.StatusInsufficientHistory
	case quality.Skill <= MinSkill:
		quality.Status = demandforecast.StatusNoSkill
	default:
		quality.Status = demandforecast.StatusReady
	}

	return &Training{Fitted: model.fitted(h), Quality: quality, Days: t.trackRecord()}
}

func (h *History) upcomingHolidays() []int64 {
	today := h.Grid.Day(h.Now)
	days := []int64{}
	for day := today - 1; day <= today+HolidayHorizonDays; day++ {
		if h.Holidays.OnDay(day) {
			days = append(days, day)
		}
	}

	return days
}

func (t *training) visits(first, current int64) []int64 {
	weeks := make([]int64, 0, BacktestWeeks+4)
	for day := range t.h.Holidays {
		if week := dispatchmarket.WeekOf(day * dispatchmarket.DaySeconds); week < first {
			if _, seen := t.actuals[week]; seen {
				weeks = append(weeks, week)
			}
		}
	}
	for week := first; week <= current; week++ {
		weeks = append(weeks, week)
	}
	slices.Sort(weeks)

	return slices.Compact(weeks)
}

func (t *training) train(week int64) *trainedWeek {
	w := windowOf(t.h.Counts, week, t.h.Holidays)
	if w.weeks() < minTrainingWeeks {
		t.weeks[week] = nil
		return nil
	}
	agg := aggregateOf(t.h.Counts, w, t.h.Holidays)
	model := &trainedWeek{week: week, agg: agg, cells: sortedCells(agg.areas)}
	for _, cell := range model.cells {
		area := agg.areas[cell]
		for slot := range slotsPerWeek {
			alpha, beta := agg.areaRate(area, slot)
			model.base[slot] = model.base[slot].Plus(demandforecast.GammaPoisson(alpha, beta))
		}
	}
	model.level = t.level(model)
	holiday := t.holidays.factor(t.days.dispersion())
	for slot := range slotsPerWeek {
		model.scale[slot] = model.level
		if t.h.Holidays.On(week, int16(slot)) {
			model.scale[slot] *= holiday
		}
		model.totals[slot] = model.base[slot].Scale(model.scale[slot])
	}
	t.weeks[week] = model

	return model
}

func (t *training) level(model *trainedWeek) float64 {
	loads, expected := 0.0, 0.0
	for back := int64(1); back <= LevelWeeks; back++ {
		week := model.week - back
		if week < model.agg.w.first {
			return 1
		}
		actual := t.actuals[week]
		for slot := range slotsPerWeek {
			if t.h.Holidays.On(week, int16(slot)) {
				continue
			}
			loads += float64(actual.total(slot))
			expected += model.base[slot].Mean
		}
	}
	if expected <= 0 {
		return 1
	}
	settled := t.weeklyAt[model.week-LevelWeeks]
	dispersion := settled.dispersion()
	prior := 1 / (LevelPriorSpread * LevelPriorSpread)
	level := (prior + loads/dispersion) / (prior + expected/dispersion)

	return math.Min(math.Max(level, MinLevel), MaxLevel)
}

func (t *training) test(model *trainedWeek) {
	actual := t.actuals[model.week]
	previous := t.actuals[model.week-1]
	for _, cell := range model.cells {
		area := model.agg.areas[cell]
		for slot := range slotsPerWeek {
			key := slotKey{cell: cell, slot: int16(slot)}
			t.total.add(model.predict(area, slot), actual.of(key), previous.of(key))
		}
	}
	if actual != nil {
		for key, loads := range actual.bySlot {
			if _, trained := model.agg.areas[key.cell]; !trained {
				t.total.add(demandforecast.Count{}, loads, previous.of(key))
			}
		}
	}

	dispersion := t.dispersions[model.week]
	week, weekLoads := demandforecast.Count{}, int32(0)
	for day := range daysPerWeek {
		predicted, loads := model.day(actual, day)
		ranged := predicted.Widen(dispersion)
		t.tested++
		if int(loads) >= ranged.Quantile(demandforecast.IntervalLow) &&
			int(loads) <= ranged.Quantile(demandforecast.IntervalHigh) {
			t.held++
		}
		t.days.add(predicted, loads)
		week, weekLoads = week.Plus(predicted), weekLoads+loads
	}
	t.weekly.add(week, weekLoads)
}

func (m *trainedWeek) day(actual *weekActuals, day int) (demandforecast.Count, int32) {
	predicted, loads := demandforecast.Count{}, int32(0)
	first := day * int(dispatchmarket.SlotsPerDay)
	for slot := first; slot < first+int(dispatchmarket.SlotsPerDay); slot++ {
		predicted = predicted.Plus(m.totals[slot])
		loads += actual.total(slot)
	}

	return predicted, loads
}

func (t *training) observeHolidays(model *trainedWeek) {
	actual := t.actuals[model.week]
	for slot := range slotsPerWeek {
		if t.h.Holidays.On(model.week, int16(slot)) {
			t.holidays.loads += float64(actual.total(slot))
			t.holidays.expected += model.base[slot].Mean * model.level
		}
	}
}
