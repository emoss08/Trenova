package demandforecast

import "github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"

type Holidays map[int64]struct{}

func (h Holidays) On(week int64, slot int16) bool {
	_, ok := h[dispatchmarket.WallDay(week, slot)]

	return ok
}

func (h Holidays) OnDay(day int64) bool {
	_, ok := h[day]

	return ok
}

type Fitted struct {
	Rates  []*Rate
	Areas  []*Area
	Curves []*BookingCurve
}
