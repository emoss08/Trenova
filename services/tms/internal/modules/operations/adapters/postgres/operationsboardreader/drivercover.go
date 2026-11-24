package operationsboardreader

import (
	"cmp"
	"context"
	"math"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
)

const (
	coverDeepShort = 2
	coverNoonHour  = 12

	figureCoverAreas     = "areasShort"
	figureCoverShort     = "short"
	figureCoverMaybe     = "mayBeShort"
	figureCoverWorst     = "worst"
	figureCoverWorstArea = "worstArea"
	figureCoverWorstSlot = "worstSlot"
	figureCoverSpare     = "spare"
)

type coverBalances interface {
	Balances(
		ctx context.Context,
		req *freightoutlookports.Request,
	) (*freightoutlookports.Balances, error)
}

type DriverCoverParams struct {
	fx.In

	Outlook freightoutlookports.Service
}

type DriverCover struct {
	outlook coverBalances
}

func NewDriverCover(p DriverCoverParams) *DriverCover {
	return &DriverCover{outlook: p.Outlook}
}

func (*DriverCover) Metric() string { return operationsboard.MetricDriverCover }

func (r *DriverCover) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	balances, err := r.outlook.Balances(ctx, &freightoutlookports.Request{TenantInfo: req.Tenant})
	if err != nil {
		return nil, err
	}

	return coverRead(ctx, balances, coverZone(balances, req.Location)), nil
}

func coverZone(balances *freightoutlookports.Balances, fallback *time.Location) *time.Location {
	if balances != nil && balances.Quality != nil && balances.Quality.Timezone != "" {
		return dispatchmarket.GridNamed(balances.Quality.Timezone).Location()
	}

	return serviceZone(fallback)
}

type coverArea struct {
	label string
	cells []boardcatalog.Cell
	short int
	maybe int
	need  float64
	cell  int64
}

type coverWorst struct {
	short int
	area  string
	start int64
	end   int64
}

func coverRead(
	ctx context.Context,
	balances *freightoutlookports.Balances,
	loc *time.Location,
) *boardcatalog.MetricRead {
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Drivers against need by forecast area, the next 48 hours"),
		Figures: map[string]*boardcatalog.Value{
			figureCoverAreas:     nil,
			figureCoverShort:     nil,
			figureCoverMaybe:     nil,
			figureCoverWorst:     nil,
			figureCoverWorstArea: nil,
			figureCoverWorstSlot: nil,
			figureCoverSpare:     nil,
		},
		Grid: &boardcatalog.Grid{
			Columns: []boardcatalog.GridHead{},
			Rows:    []boardcatalog.GridRow{},
		},
	}
	if balances == nil || balances.Quality == nil || !balances.Quality.Ready() ||
		len(balances.Starts) == 0 {
		read.Source = i18n.T(ctx, "the freight forecast for the next 48 hours, not ready yet")
		return read
	}

	areas := make([]coverArea, 0, len(balances.Areas))
	worst := coverWorst{}
	short, maybe, spare, shortAreas := 0, 0, 0, 0
	for idx := range balances.Areas {
		balance := &balances.Areas[idx]
		area := coverAreaOf(ctx, balance)
		short += area.short
		if area.short > 0 {
			shortAreas++
		}
		maybe += area.maybe
		if last := len(balance.Slots) - 1; last >= 0 {
			spare += coverRound(balance.Slots[last].Spare)
		}
		for slot := range balance.Slots {
			if gap := coverRound(balance.Slots[slot].Short); gap > worst.short {
				worst = coverWorst{
					short: gap,
					area:  area.label,
					start: balance.Slots[slot].Start,
					end:   balance.Slots[slot].End,
				}
			}
		}
		if area.need > 0 {
			areas = append(areas, area)
		}
	}

	read.Source = i18n.T(
		ctx,
		"the freight forecast, {0, plural, one {# forecast area} other {# forecast areas}}, "+
			"the next 48 hours",
		len(balances.Areas),
	)
	read.Grid = coverGrid(ctx, balances.Starts, areas, loc)
	read.Figures[figureCoverAreas] = coverCount(shortAreas)
	read.Figures[figureCoverShort] = coverCount(short)
	read.Figures[figureCoverMaybe] = coverCount(maybe)
	read.Figures[figureCoverWorst] = coverCount(worst.short)
	read.Figures[figureCoverSpare] = coverCount(spare)
	if worst.short == 0 {
		read.Figures[figureCoverWorstArea] = serviceText(i18n.T(ctx, "No area is short"))
		read.Figures[figureCoverWorstSlot] = serviceText(i18n.T(ctx, "No slot is short"))
		return read
	}
	read.Figures[figureCoverWorstArea] = serviceText(worst.area)
	read.Figures[figureCoverWorstSlot] = serviceText(i18n.T(
		ctx,
		"{0}, {1} to {2}",
		coverDay(ctx, time.Unix(worst.start, 0).In(loc)),
		coverHour(ctx, time.Unix(worst.start, 0).In(loc)),
		coverHour(ctx, time.Unix(worst.end, 0).In(loc)),
	))

	return read
}

func coverAreaOf(ctx context.Context, balance *demandforecast.AreaBalance) coverArea {
	area := coverArea{
		label: cmp.Or(balance.Label, i18n.T(ctx, "An area with no named origin")),
		cells: make([]boardcatalog.Cell, len(balance.Slots)),
		cell:  balance.Cell,
	}
	for idx := range balance.Slots {
		slot := &balance.Slots[idx]
		area.cells[idx] = coverCellOf(slot)
		area.need += slot.Need
		switch area.cells[idx].State {
		case boardcatalog.CellShort:
			area.short += area.cells[idx].Need - area.cells[idx].Have
		case boardcatalog.CellMaybe:
			area.maybe++
		case boardcatalog.CellCovered, boardcatalog.CellDeep:
		}
	}

	return area
}

func coverCellOf(slot *demandforecast.SlotBalance) boardcatalog.Cell {
	need := coverRound(slot.Need)
	short := coverRound(slot.Short)
	have := max(0, need-short+coverRound(slot.Spare))
	high := max(need, slot.Open.Quantile(demandforecast.IntervalHigh))
	cell := boardcatalog.Cell{Have: have, Need: need, NeedHigh: high}
	switch {
	case have < need:
		cell.State = boardcatalog.CellShort
	case high > have:
		cell.State = boardcatalog.CellMaybe
	default:
		cell.State, cell.NeedHigh = boardcatalog.CellCovered, need
	}

	return cell
}

func coverGrid(
	ctx context.Context,
	starts []int64,
	areas []coverArea,
	loc *time.Location,
) *boardcatalog.Grid {
	slices.SortFunc(areas, func(a, b coverArea) int {
		if c := cmp.Compare(b.short, a.short); c != 0 {
			return c
		}
		if c := cmp.Compare(b.maybe, a.maybe); c != 0 {
			return c
		}
		if c := cmp.Compare(b.need, a.need); c != 0 {
			return c
		}
		return cmp.Compare(a.cell, b.cell)
	})
	areas = areas[:min(len(areas), operationsboard.MaxCoverAreas)]

	grid := &boardcatalog.Grid{
		Columns: make([]boardcatalog.GridHead, len(starts)),
		Rows:    make([]boardcatalog.GridRow, 0, len(areas)),
	}
	for idx, start := range starts {
		at := time.Unix(start, 0).In(loc)
		grid.Columns[idx] = boardcatalog.GridHead{
			Label:    coverHour(ctx, at),
			Sublabel: coverDay(ctx, at),
		}
	}
	deepest := 0
	for idx := range areas {
		for _, cell := range areas[idx].cells {
			if cell.State == boardcatalog.CellShort {
				deepest = max(deepest, cell.Need-cell.Have)
			}
		}
	}
	for idx := range areas {
		cells := areas[idx].cells
		if deepest >= coverDeepShort {
			for at := range cells {
				gap := cells[at].Need - cells[at].Have
				if cells[at].State == boardcatalog.CellShort && gap == deepest {
					cells[at].State = boardcatalog.CellDeep
				}
			}
		}
		grid.Rows = append(grid.Rows, boardcatalog.GridRow{Label: areas[idx].label, Cells: cells})
	}

	return grid
}

func coverRound(value float64) int {
	return int(math.Round(value))
}

func coverCount(n int) *boardcatalog.Value {
	return serviceCount(int64(n))
}

func coverHour(ctx context.Context, at time.Time) string {
	hour := at.Hour() % coverNoonHour
	if hour == 0 {
		hour = coverNoonHour
	}
	if at.Hour() < coverNoonHour {
		return i18n.T(ctx, "{0} AM", hour)
	}

	return i18n.T(ctx, "{0} PM", hour)
}

func coverDay(ctx context.Context, at time.Time) string {
	date := at.Day()
	switch at.Weekday() {
	case time.Sunday:
		return i18n.T(ctx, "Sun {0}", date)
	case time.Monday:
		return i18n.T(ctx, "Mon {0}", date)
	case time.Tuesday:
		return i18n.T(ctx, "Tue {0}", date)
	case time.Wednesday:
		return i18n.T(ctx, "Wed {0}", date)
	case time.Thursday:
		return i18n.T(ctx, "Thu {0}", date)
	case time.Friday:
		return i18n.T(ctx, "Fri {0}", date)
	case time.Saturday:
		return i18n.T(ctx, "Sat {0}", date)
	}

	return ""
}
