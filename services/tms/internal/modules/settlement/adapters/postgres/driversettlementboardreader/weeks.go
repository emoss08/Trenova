package driversettlementboardreader

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type SettlementWeeks struct{ release settlementreleaseports.Reader }

func NewSettlementWeeks(p Params) *SettlementWeeks {
	return &SettlementWeeks{release: p.Release}
}

func (*SettlementWeeks) Metric() string { return settlementboard.MetricSettlementWeeks }

func (r *SettlementWeeks) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	weeks, err := r.release.Weeks(ctx, &settlementreleaseports.WeeksRequest{
		Weeks: settlementboard.WeeksShown,
	})
	if err != nil {
		return nil, err
	}

	return weeksRead(ctx, weeks, req.Location, req.Now), nil
}

func minorPoint(minor int64) *decimal.Decimal {
	point := decimal.New(minor, -2)

	return &point
}

func weeksRead(
	ctx context.Context,
	weeks []*settlementreleaseports.Week,
	loc *time.Location,
	now int64,
) *boardcatalog.MetricRead {
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "{0, plural, one {The last pay batch} other {The last # pay batches}}",
			len(weeks)),
		Figures: map[string]*boardcatalog.Value{
			settlementboard.FigureLargestBatchEarned: nil,
			settlementboard.FigureLargestBatchPeriod: nil,
			settlementboard.FigureLatestBatchEarned:  nil,
			settlementboard.FigureLatestBatchNet:     nil,
		},
	}
	var batches int64
	for _, week := range weeks {
		batches += week.Batches
	}
	read.Source = i18n.T(ctx, "{0, plural, one {# pay batch} other {# pay batches}}", batches)
	if len(weeks) == 0 || mixedCurrencies(weeks) {
		return read
	}

	currency := weeks[0].Currency
	chart := &boardcatalog.Chart{
		Unit:     boardcatalog.ValueMoney,
		Currency: currency,
		X:        make([]boardcatalog.Tick, 0, len(weeks)),
		Series: []boardcatalog.Series{
			{Name: i18n.T(ctx, "Net pay"), Role: boardcatalog.RoleStack},
			{Name: i18n.T(ctx, "Deducted"), Role: boardcatalog.RoleStack},
		},
		Marked: []int{len(weeks) - 1},
	}
	largest := 0
	for idx, week := range weeks {
		chart.X = append(chart.X, boardcatalog.Tick{
			At:     settlementreleaseports.Day(week.PeriodStart, loc).Unix(),
			Marked: idx == len(weeks)-1,
		})
		chart.Series[0].Points = append(chart.Series[0].Points, minorPoint(week.NetMinor))
		chart.Series[1].Points = append(chart.Series[1].Points, minorPoint(week.DeductedMinor()))
		if week.NetMinor+week.DeductedMinor() >
			weeks[largest].NetMinor+weeks[largest].DeductedMinor() {
			largest = idx
		}
	}
	read.Chart = chart
	latest := weeks[len(weeks)-1]
	money := func(minor int64) *boardcatalog.Value {
		return valueOf(boardcatalog.Money(minor, currency))
	}
	read.Figures[settlementboard.FigureLargestBatchEarned] = money(
		weeks[largest].NetMinor + weeks[largest].DeductedMinor())
	read.Figures[settlementboard.FigureLargestBatchPeriod] = &boardcatalog.Value{
		Kind: boardcatalog.ValueText,
		Text: i18n.T(ctx, "Week of {0}", dayText(ctx, weeks[largest].PeriodStart, loc, now)),
	}
	read.Figures[settlementboard.FigureLatestBatchEarned] = money(
		latest.NetMinor + latest.DeductedMinor())
	read.Figures[settlementboard.FigureLatestBatchNet] = money(latest.NetMinor)

	return read
}

func mixedCurrencies(weeks []*settlementreleaseports.Week) bool {
	for _, week := range weeks {
		if week.Currencies > 1 || week.Currency != weeks[0].Currency {
			return true
		}
	}

	return false
}
