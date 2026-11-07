package demandforecast

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
)

const (
	RecordDays  = 14
	RecordWeeks = 8
	DaySeconds  = int64(24 * 3600)
	slotsPerDay = int16(DaySeconds / dispatchmarket.SlotSeconds)
)

type trainedWeek struct {
	agg   *aggregate
	w     window
	cells []int64
}

func TrackRecord(h *History) []*Day {
	today := h.Now - h.Now%DaySeconds
	actuals := weeklyActuals(h.Counts)
	trained := map[int64]*trainedWeek{}
	days := make([]*Day, 0, RecordDays)

	for back := int64(RecordDays); back >= 1; back-- {
		start := today - back*DaySeconds
		week := dispatchmarket.WeekOf(start)
		model, ok := trained[week]
		if !ok {
			model = trainWeek(h.Counts, week)
			trained[week] = model
		}
		if model == nil {
			continue
		}

		first := dispatchmarket.SlotOf(start)
		predicted := Count{}
		actual := int32(0)
		for slot := first; slot < first+slotsPerDay; slot++ {
			for _, cell := range model.cells {
				predicted = predicted.Plus(
					model.agg.predict(model.agg.areas[cell], model.w, int(slot)),
				)
			}
			for key, loads := range actuals[week] {
				if key.slot == slot {
					actual += loads
				}
			}
		}

		days = append(days, &Day{
			DayStart: start,
			Low:      clampInt32(predicted.Quantile(IntervalLow)),
			High:     clampInt32(predicted.Quantile(IntervalHigh)),
			Mean:     predicted.Mean,
			Actual:   actual,
			FittedAt: h.Now,
		})
	}

	return days
}

func trainWeek(counts []LoadCount, week int64) *trainedWeek {
	w := windowOf(counts, week)
	if w.weeks() < minTrainingWeeks {
		return nil
	}
	agg := aggregateOf(counts, w)

	return &trainedWeek{agg: agg, w: w, cells: sortedCells(agg.areas)}
}

func clampInt32(v int) int32 {
	return int32(min(max(v, 0), math.MaxInt32)) //nolint:gosec // clamped to the int32 range above
}
