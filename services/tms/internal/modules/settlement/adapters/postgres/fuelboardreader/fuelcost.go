package fuelboardreader

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const (
	rateDecimals  = 3
	priceDecimals = 3
)

type FuelCostPerMile struct{ db *postgres.Connection }

type dayRow struct {
	DayAt       int64
	Purchases   int64
	Gallons     decimal.Decimal
	AmountMinor int64
	Miles       decimal.Decimal
	Currencies  int64
	Currency    string
}

type fuelTotals struct {
	purchases int64
	gallons   decimal.Decimal
	amount    int64
	miles     decimal.Decimal
}

func (t *fuelTotals) add(row *dayRow) {
	t.purchases += row.Purchases
	t.gallons = t.gallons.Add(row.Gallons)
	t.amount += row.AmountMinor
	t.miles = t.miles.Add(row.Miles)
}

func perMile(amountMinor int64, miles decimal.Decimal) *decimal.Decimal {
	if !miles.IsPositive() {
		return nil
	}
	rate := decimal.New(amountMinor, -2).Div(miles).Round(rateDecimals)

	return &rate
}

type monthWindow struct {
	first, last time.Time
	prevFirst   time.Time
	prevLast    time.Time
	running     bool
	loc         *time.Location
}

func windowOf(period boardcatalog.Period, now int64, loc *time.Location) (monthWindow, bool) {
	if loc == nil {
		loc = time.UTC
	}
	first, last, ok := period.Days(now, loc)
	if !ok {
		return monthWindow{}, false
	}
	prevFirst := first.AddDate(0, -1, 0)

	return monthWindow{
		first:     first,
		last:      last,
		prevFirst: prevFirst,
		prevLast:  first.AddDate(0, 0, -1),
		running:   period == boardcatalog.PeriodThisMonth,
		loc:       loc,
	}, true
}

func NewFuelCostPerMile(p Params) *FuelCostPerMile { return &FuelCostPerMile{db: p.DB} }

func (*FuelCostPerMile) Metric() string { return fuelboard.MetricFuelCostPerMile }

func (r *FuelCostPerMile) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	window, ok := windowOf(req.Period, req.Now, req.Location)
	if !ok {
		return nil, nil //nolint:nilnil // a period the metric does not take reads as nothing
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*boardcatalog.MetricRead, error) {
		t := fuelpurchasedb.TractorDays
		from := window.prevFirst.AddDate(0, 0, -(fuelboard.RunDays - 1)).Unix()
		rows, err := keel.Into[dayRow](ctx, r.db.Keel(ctx), t.Pick(
			t.DayAt,
			keel.Int64Of(t.Purchases.Sum()).Coalesce(0),
			t.Gallons.Sum(),
			keel.Int64Of(t.AmountMinor.Sum()).Coalesce(0),
			t.Miles.Sum(),
			keel.Int64Of(t.Currencies.Max()).Coalesce(0),
			keel.CastText(t.CurrencyCode.Min()).Coalesce(""),
		).Where(t.DayAt.Gte(from), t.DayAt.Lte(window.last.Unix())).
			Group(t.DayAt).
			Order(t.DayAt.Asc()))
		if err != nil {
			return nil, err
		}

		return costRead(ctx, &window, rows), nil
	})
}

func mainCurrency(rows []dayRow) string {
	counts := map[string]int{}
	main := ""
	for idx := range rows {
		code := rows[idx].Currency
		if code == "" {
			continue
		}
		counts[code]++
		if counts[code] > counts[main] || (counts[code] == counts[main] && code < main) {
			main = code
		}
	}

	return main
}

func (row *dayRow) foreign(currency string) bool {
	return row.Currencies > 1 || (row.Currency != "" && row.Currency != currency)
}

type costDays struct {
	byDay    map[int64]*dayRow
	currency string
}

func (c *costDays) total(first, last time.Time) (fuelTotals, bool) {
	var sum fuelTotals
	for day := first; !day.After(last); day = day.AddDate(0, 0, 1) {
		row := c.byDay[day.Unix()]
		if row == nil {
			continue
		}
		if row.foreign(c.currency) {
			return fuelTotals{}, false
		}
		sum.add(row)
	}

	return sum, true
}

func (c *costDays) running(day time.Time) *decimal.Decimal {
	run, ok := c.total(day.AddDate(0, 0, -(fuelboard.RunDays-1)), day)
	if !ok {
		return nil
	}

	return perMile(run.amount, run.miles)
}

func costRead(ctx context.Context, window *monthWindow, rows []dayRow) *boardcatalog.MetricRead {
	days := &costDays{byDay: make(map[int64]*dayRow, len(rows)), currency: mainCurrency(rows)}
	for idx := range rows {
		days.byDay[rows[idx].DayAt] = &rows[idx]
	}

	ticks := make([]boardcatalog.Tick, 0, 31)
	current := make([]*decimal.Decimal, 0, 31)
	earlier := make([]*decimal.Decimal, 0, 31)
	for day := window.first; !day.After(window.last); day = day.AddDate(0, 0, 1) {
		ticks = append(ticks, boardcatalog.Tick{At: day.Unix()})
		current = append(current, days.running(day))
		then := window.prevFirst.AddDate(0, 0, int(daysBetween(window.first, day)))
		if then.After(window.prevLast) {
			earlier = append(earlier, nil)
		} else {
			earlier = append(earlier, days.running(then))
		}
	}
	now, nowOne := days.total(window.first, window.last)
	was, wasOne := days.total(window.prevFirst, window.prevLast)

	month := monthName(ctx, window.first)
	prevMonth := monthName(ctx, window.prevFirst)
	currency := days.currency
	chart := &boardcatalog.Chart{
		Unit:     boardcatalog.ValueMoney,
		Currency: currency,
		X:        ticks,
		Series: []boardcatalog.Series{
			{Name: month, Role: boardcatalog.RoleNow, Points: current},
			{Name: prevMonth, Role: boardcatalog.RoleWas, Points: earlier},
		},
		Ghost: window.running,
	}
	figures := map[string]*boardcatalog.Value{
		fuelboard.FigureCostPerMile:         nil,
		fuelboard.FigurePreviousCostPerMile: nil,
		fuelboard.FigurePricePerGallon:      nil,
		fuelboard.FigureFuelSpend:           nil,
		fuelboard.FigureMilesRun:            nil,
	}
	if nowOne {
		figures[fuelboard.FigureMilesRun] = numberValue(now.miles.Round(0))
		figures[fuelboard.FigureFuelSpend] = moneyValue(boardcatalog.Money(now.amount, currency))
		if rate := perMile(now.amount, now.miles); rate != nil {
			figures[fuelboard.FigureCostPerMile] = moneyValue(boardcatalog.Amount(*rate, currency))
		}
		if now.gallons.IsPositive() {
			price := decimal.New(now.amount, -2).Div(now.gallons).Round(priceDecimals)
			figures[fuelboard.FigurePricePerGallon] = moneyValue(
				boardcatalog.Amount(price, currency),
			)
		}
	}
	changes := map[string]boardcatalog.FigureShift{}
	if rate := perMile(was.amount, was.miles); wasOne && rate != nil {
		previous := boardcatalog.Amount(*rate, currency)
		figures[fuelboard.FigurePreviousCostPerMile] = &previous
		chart.Ref = &boardcatalog.Line{Value: *rate, Label: prevMonth}
		if current := figures[fuelboard.FigureCostPerMile]; current != nil {
			shift := boardcatalog.FigureShift{Was: &previous, WasAt: window.prevFirst.Unix()}
			if current.Number.GreaterThan(previous.Number) {
				shift.Tone = boardcatalog.ToneWarn
			}
			changes[fuelboard.FigureCostPerMile] = shift
		}
	}

	return &boardcatalog.MetricRead{
		Title:   i18n.T(ctx, "Fuel cost per mile, {0}", month),
		Source:  costSource(ctx, now.purchases+was.purchases, month, prevMonth, nowOne, wasOne),
		Figures: figures,
		Changes: changes,
		Chart:   chart,
	}
}

func costSource(
	ctx context.Context,
	purchases int64,
	month, prevMonth string,
	nowOne, wasOne bool,
) string {
	count := i18n.FormatNumber(i18n.FromContext(ctx), float64(purchases))
	switch {
	case !nowOne:
		return i18n.T(
			ctx,
			"{0} fuel purchases and the miles tractors ran; {1} holds fuel bought in another currency, so its cost per mile is not added up",
			count,
			month,
		)
	case !wasOne:
		return i18n.T(
			ctx,
			"{0} fuel purchases and the miles tractors ran; {1} holds fuel bought in another currency, so it is not added up to compare",
			count,
			prevMonth,
		)
	default:
		return i18n.T(ctx, "{0} fuel purchases and the miles tractors ran, {1} and {2}",
			count, prevMonth, month)
	}
}

func monthName(ctx context.Context, at time.Time) string {
	return i18n.T(ctx, at.Month().String())
}

func moneyValue(value boardcatalog.Value) *boardcatalog.Value { return &value }

func numberValue(number decimal.Decimal) *boardcatalog.Value {
	value := boardcatalog.Number(number)

	return &value
}
