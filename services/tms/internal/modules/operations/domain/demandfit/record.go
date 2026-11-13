package demandfit

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
)

func (t *training) trackRecord() []*demandforecast.Day {
	grid := t.h.Grid
	days := make([]*demandforecast.Day, 0, demandforecast.RecordDays+1)
	for back := demandforecast.RecordDays + 1; back >= 1; back-- {
		start := grid.DayStart(t.h.Now, back)
		if grid.DayStart(t.h.Now, back-1) > t.h.Now {
			continue
		}
		if day := t.day(start); day != nil {
			days = append(days, day)
		}
	}

	return days[max(len(days)-demandforecast.RecordDays, 0):]
}

func (t *training) day(start int64) *demandforecast.Day {
	wall := t.h.Grid.Wall(start)
	week := dispatchmarket.WeekOf(wall)
	model := t.weeks[week]
	if model == nil {
		return nil
	}
	predicted, actual := model.day(
		t.actuals[week],
		int(dispatchmarket.SlotOf(wall))/int(dispatchmarket.SlotsPerDay),
	)
	dispersion, ok := t.dispersions[week]
	if !ok {
		dispersion = 1
	}
	ranged := predicted.Widen(dispersion)

	return &demandforecast.Day{
		DayStart: start,
		Low:      clampInt32(ranged.Quantile(demandforecast.IntervalLow)),
		High:     clampInt32(ranged.Quantile(demandforecast.IntervalHigh)),
		Mean:     predicted.Mean,
		Actual:   actual,
		FittedAt: t.h.Now,
	}
}

func clampInt32(v int) int32 {
	return int32(min(max(v, 0), math.MaxInt32)) //nolint:gosec // clamped to the int32 range above
}
