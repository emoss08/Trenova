package demandforecast

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
)

type IndexParams struct {
	Quality       *Quality
	Rates         []*Rate
	Areas         []*Area
	Curves        []*BookingCurve
	CustomerNames map[pulid.ID]string
}

type Index struct {
	quality       *Quality
	ready         bool
	grid          dispatchmarket.Grid
	areas         map[int64]*Area
	cells         []int64
	rates         map[rateKey]*Rate
	customers     map[rateKey][]*Rate
	curves        map[pulid.ID][]float64
	tenant        []float64
	names         map[pulid.ID]string
	holidays      Holidays
	holidayFactor float64
	dispersion    float64
	weight        int
}

func NewIndex(p *IndexParams) *Index {
	i := &Index{
		quality:       p.Quality,
		ready:         p.Quality.Ready(),
		areas:         make(map[int64]*Area, len(p.Areas)),
		cells:         make([]int64, 0, len(p.Areas)),
		rates:         make(map[rateKey]*Rate, len(p.Rates)),
		customers:     map[rateKey][]*Rate{},
		curves:        make(map[pulid.ID][]float64, len(p.Curves)),
		names:         p.CustomerNames,
		holidays:      Holidays{},
		holidayFactor: 1,
		dispersion:    1,
		weight:        1 + len(p.Rates) + len(p.Areas) + len(p.Curves),
	}
	if p.Quality != nil {
		i.grid = dispatchmarket.GridNamed(p.Quality.Timezone)
		i.dispersion = p.Quality.Dispersion
		if p.Quality.HolidayFactor > 0 {
			i.holidayFactor = p.Quality.HolidayFactor
		}
		for _, day := range p.Quality.Holidays {
			i.holidays[day] = struct{}{}
		}
	}
	for _, area := range p.Areas {
		i.areas[area.Cell] = area
		i.cells = append(i.cells, area.Cell)
	}
	slices.Sort(i.cells)
	for _, rate := range p.Rates {
		key := rateKey{cell: rate.Cell, slot: rate.Slot}
		if rate.CustomerID.IsNil() {
			i.rates[key] = rate
			continue
		}
		i.customers[key] = append(i.customers[key], rate)
	}
	for _, curve := range p.Curves {
		if curve.CustomerID.IsNil() {
			i.tenant = curve.Shares
			continue
		}
		i.curves[curve.CustomerID] = curve.Shares
	}

	return i
}

func (i *Index) Ready() bool {
	return i != nil && i.ready
}

func (i *Index) Quality() *Quality {
	return i.quality
}

func (i *Index) Grid() dispatchmarket.Grid {
	return i.grid
}

func (i *Index) Weight() int {
	return i.weight
}

func (i *Index) Cells() []int64 {
	return i.cells
}

func (i *Index) Area(cell int64) (*Area, bool) {
	area, ok := i.areas[cell]

	return area, ok
}

func (i *Index) scale(start int64) float64 {
	if i.holidays.OnDay(i.grid.Day(start)) {
		return i.holidayFactor
	}

	return 1
}

func (i *Index) RateAt(cell, start int64) (*Rate, bool) {
	rate, ok := i.rates[rateKey{cell: cell, slot: i.grid.Slot(start)}]

	return rate, ok
}

func OutlookStarts(grid dispatchmarket.Grid, now int64) []int64 {
	return grid.Starts(now, OutlookSlots)
}
