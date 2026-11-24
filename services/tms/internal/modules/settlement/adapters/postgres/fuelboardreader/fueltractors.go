package fuelboardreader

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	namedEachSide  = 3
	ghostsDrawn    = 3
	minPurchases   = 2
	minPlacedMiles = 500
	minForRange    = 5
)

var (
	usualLowShare  = decimal.RequireFromString("0.1")
	usualHighShare = decimal.RequireFromString("0.9")
)

type FuelCostByTractor struct{ db *postgres.Connection }

func NewFuelCostByTractor(p Params) *FuelCostByTractor {
	return &FuelCostByTractor{db: p.DB}
}

func (*FuelCostByTractor) Metric() string { return fuelboard.MetricFuelCostByTractor }

type tractorRow struct {
	TractorID   pulid.ID
	Code        string
	Purchases   int64
	AmountMinor int64
	Miles       decimal.Decimal
	Currencies  int64
	Currency    string
}

type placedTractor struct {
	row   *tractorRow
	rate  decimal.Decimal
	ghost bool
	above bool
	below bool
	named bool
}

func (r *FuelCostByTractor) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	window, ok := windowOf(req.Period, req.Now, req.Location)
	if !ok {
		return nil, nil //nolint:nilnil // a period the metric does not take reads as nothing
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*boardcatalog.MetricRead, error) {
		t := fuelpurchasedb.TractorDays
		tr := tractordb.Tractors
		rows, err := keel.Into[tractorRow](ctx, r.db.Keel(ctx), t.
			Join(tr, tr.ID.EqCol(t.TractorID)).
			Pick(
				t.TractorID,
				text(tr.Code),
				keel.Int64Of(t.Purchases.Sum()).Coalesce(0),
				keel.Int64Of(t.AmountMinor.Sum()).Coalesce(0),
				t.Miles.Sum(),
				keel.Int64Of(t.Currencies.Max()).Coalesce(0),
				keel.CastText(t.CurrencyCode.Min()).Coalesce(""),
			).
			Where(t.DayAt.Gte(window.first.Unix()), t.DayAt.Lte(window.last.Unix())).
			Group(t.TractorID, tr.Code))
		if err != nil {
			return nil, err
		}

		return tractorsRead(ctx, &window, rows), nil
	})
}

func tractorsRead(
	ctx context.Context,
	window *monthWindow,
	rows []tractorRow,
) *boardcatalog.MetricRead {
	month := monthName(ctx, window.first)
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Each tractor's fuel cost per mile, {0}", month),
		Figures: map[string]*boardcatalog.Value{
			fuelboard.FigureTractorsAbove:      nil,
			fuelboard.FigureHighestTractor:     nil,
			fuelboard.FigureHighestCostPerMile: nil,
			fuelboard.FigureTractorsBelow:      nil,
			fuelboard.FigureLowestTractor:      nil,
			fuelboard.FigureLowestCostPerMile:  nil,
			fuelboard.FigureUsualLow:           nil,
			fuelboard.FigureUsualHigh:          nil,
			fuelboard.FigureFleetCostPerMile:   nil,
		},
	}
	var fleet fuelTotals
	currency := ""
	for idx := range rows {
		row := &rows[idx]
		if row.Currencies > 1 ||
			(currency != "" && row.Currency != "" && row.Currency != currency) {
			read.Source = i18n.T(
				ctx,
				"Fuel bought in more than one currency cannot be put on one line",
			)
			return read
		}
		if currency == "" {
			currency = row.Currency
		}
		fleet.amount += row.AmountMinor
		fleet.miles = fleet.miles.Add(row.Miles)
	}
	line := perMile(fleet.amount, fleet.miles)
	if line == nil {
		read.Source = i18n.T(ctx, "No tractor ran miles in {0}", month)
		return read
	}
	placed := placeTractors(rows)
	usual, ok := usualRange(placed)
	if ok {
		markOutside(placed, usual)
	}
	drawn := drawnTractors(placed)
	read.Chart = &boardcatalog.Chart{
		Unit:     boardcatalog.ValueMoney,
		Currency: currency,
		Ref:      &boardcatalog.Line{Value: *line, Label: i18n.T(ctx, "Fleet")},
		Rows:     chartRows(drawn, currency),
	}
	fleetRate := boardcatalog.Amount(*line, currency)
	read.Figures[fuelboard.FigureFleetCostPerMile] = &fleetRate
	read.Source = i18n.T(ctx, "{0} of {1} tractors that ran in {2}",
		i18n.FormatNumber(i18n.FromContext(ctx), float64(len(drawn))),
		i18n.FormatNumber(i18n.FromContext(ctx), float64(len(placed))), month)
	if !ok {
		return read
	}
	read.Chart.Band = &boardcatalog.Band{Low: &usual.low, High: &usual.high}
	above, below := countOutside(placed)
	setFigure(read, fuelboard.FigureTractorsAbove, boardcatalog.Count(int64(above)))
	setFigure(read, fuelboard.FigureTractorsBelow, boardcatalog.Count(int64(below)))
	setFigure(read, fuelboard.FigureUsualLow, boardcatalog.Amount(usual.low, currency))
	setFigure(read, fuelboard.FigureUsualHigh, boardcatalog.Amount(usual.high, currency))
	if top := furthest(placed, true); top != nil {
		setFigure(read, fuelboard.FigureHighestTractor, textValue(top.row.Code))
		setFigure(read, fuelboard.FigureHighestCostPerMile, boardcatalog.Amount(top.rate, currency))
	}
	if bottom := furthest(placed, false); bottom != nil {
		setFigure(read, fuelboard.FigureLowestTractor, textValue(bottom.row.Code))
		setFigure(read, fuelboard.FigureLowestCostPerMile,
			boardcatalog.Amount(bottom.rate, currency))
	}

	return read
}

type rateRange struct {
	low, high decimal.Decimal
}

func setFigure(read *boardcatalog.MetricRead, name string, value boardcatalog.Value) {
	read.Figures[name] = &value
}

func textValue(text string) boardcatalog.Value {
	return boardcatalog.Value{Kind: boardcatalog.ValueText, Text: text}
}

func placeTractors(rows []tractorRow) []placedTractor {
	placed := make([]placedTractor, 0, len(rows))
	for idx := range rows {
		rate := perMile(rows[idx].AmountMinor, rows[idx].Miles)
		if rate == nil {
			continue
		}
		placed = append(placed, placedTractor{
			row:  &rows[idx],
			rate: *rate,
			ghost: rows[idx].Purchases < minPurchases ||
				rows[idx].Miles.LessThan(decimal.NewFromInt(minPlacedMiles)),
		})
	}

	return placed
}

func usualRange(placed []placedTractor) (rateRange, bool) {
	rates := make([]decimal.Decimal, 0, len(placed))
	for idx := range placed {
		if !placed[idx].ghost {
			rates = append(rates, placed[idx].rate)
		}
	}
	if len(rates) < minForRange {
		return rateRange{}, false
	}
	slices.SortFunc(rates, func(a, b decimal.Decimal) int { return a.Cmp(b) })

	return rateRange{
		low:  percentile(rates, usualLowShare).Round(rateDecimals),
		high: percentile(rates, usualHighShare).Round(rateDecimals),
	}, true
}

func percentile(sorted []decimal.Decimal, share decimal.Decimal) decimal.Decimal {
	position := share.Mul(decimal.NewFromInt(int64(len(sorted) - 1)))
	below := position.Floor()
	at := int(below.IntPart())
	if at+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	fraction := position.Sub(below)

	return sorted[at].Add(sorted[at+1].Sub(sorted[at]).Mul(fraction))
}

func markOutside(placed []placedTractor, usual rateRange) {
	for idx := range placed {
		if placed[idx].ghost {
			continue
		}
		placed[idx].above = placed[idx].rate.GreaterThan(usual.high)
		placed[idx].below = placed[idx].rate.LessThan(usual.low)
	}
	for _, high := range []bool{true, false} {
		order := outside(placed, high)
		for _, at := range order[:min(len(order), namedEachSide)] {
			placed[at].named = true
		}
	}
}

func outside(placed []placedTractor, high bool) []int {
	order := make([]int, 0, len(placed))
	for idx := range placed {
		if (high && placed[idx].above) || (!high && placed[idx].below) {
			order = append(order, idx)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		byRate := placed[a].rate.Cmp(placed[b].rate)
		if high {
			byRate = -byRate
		}

		return cmp.Or(byRate, strings.Compare(placed[a].row.Code, placed[b].row.Code))
	})

	return order
}

func countOutside(placed []placedTractor) (above, below int) {
	for idx := range placed {
		if placed[idx].above {
			above++
		}
		if placed[idx].below {
			below++
		}
	}

	return above, below
}

func furthest(placed []placedTractor, high bool) *placedTractor {
	order := outside(placed, high)
	if len(order) == 0 {
		return nil
	}

	return &placed[order[0]]
}

func drawnTractors(placed []placedTractor) []placedTractor {
	order := slices.Clone(placed)
	slices.SortStableFunc(order, func(a, b placedTractor) int {
		return cmp.Or(b.row.Miles.Cmp(a.row.Miles), strings.Compare(a.row.Code, b.row.Code))
	})
	drawn := make([]placedTractor, 0, fuelboard.MaxTractorDots)
	taken := make(map[pulid.ID]bool, fuelboard.MaxTractorDots)
	take := func(tractor placedTractor) {
		if len(drawn) < fuelboard.MaxTractorDots && !taken[tractor.row.TractorID] {
			taken[tractor.row.TractorID] = true
			drawn = append(drawn, tractor)
		}
	}
	for _, high := range []bool{true, false} {
		for _, at := range outside(placed, high) {
			if placed[at].named {
				take(placed[at])
			}
		}
	}
	ghosts := 0
	for _, tractor := range order {
		if tractor.ghost && ghosts < ghostsDrawn {
			take(tractor)
			ghosts++
		}
	}
	for _, tractor := range order {
		if !tractor.ghost {
			take(tractor)
		}
	}

	return drawn
}

func chartRows(drawn []placedTractor, currency string) []boardcatalog.ChartRow {
	rows := make([]boardcatalog.ChartRow, 0, len(drawn))
	for _, tractor := range drawn {
		rate := tractor.rate
		value := boardcatalog.Amount(rate, currency)
		dot := boardcatalog.Dot{Value: &rate, Marked: tractor.named, Ghost: tractor.ghost}
		if tractor.named {
			dot.Label = tractor.row.Code
		}
		rows = append(rows, boardcatalog.ChartRow{
			Label: tractor.row.Code,
			Ref: &boardcatalog.Ref{
				Kind: recordkind.Kind(permission.ResourceTractor),
				ID:   tractor.row.TractorID,
			},
			Value:  &value,
			Marked: tractor.named,
			Dots:   []boardcatalog.Dot{dot},
		})
	}

	return rows
}
