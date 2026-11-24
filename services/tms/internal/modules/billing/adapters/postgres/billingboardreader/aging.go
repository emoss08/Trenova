package billingboardreader

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type AnalyticsParams struct {
	fx.In

	Analytics customerledgerports.ARAnalyticsRepository
}

type AgingCompare struct {
	analytics customerledgerports.ARAnalyticsRepository
}

func NewAgingCompare(p AnalyticsParams) *AgingCompare {
	return &AgingCompare{analytics: p.Analytics}
}

func (*AgingCompare) Metric() string { return billingboard.MetricAgingCompare }

func earlierThan(now int64, period boardcatalog.Period, loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	at := time.Unix(now, 0).In(loc)
	switch period {
	case boardcatalog.PeriodLastWeek:
		return at.AddDate(0, 0, -7).Unix()
	case boardcatalog.PeriodLastQuarter:
		return at.AddDate(0, -3, 0).Unix()
	case boardcatalog.PeriodLastMonth, boardcatalog.PeriodToday, boardcatalog.PeriodYesterday,
		boardcatalog.PeriodThisWeek, boardcatalog.PeriodNextWeek, boardcatalog.PeriodThisMonth,
		boardcatalog.PeriodThisQuarter, boardcatalog.PeriodNext48Hours, boardcatalog.PeriodSlip,
		boardcatalog.PeriodNext30Days:
	}

	return at.AddDate(0, -1, 0).Unix()
}

func bucketsOf(totals *customerledgerports.ARAgingBucketTotals) []int64 {
	return []int64{
		totals.CurrentMinor,
		totals.Days1To30Minor,
		totals.Days31To60Minor,
		totals.Days61To90Minor,
		totals.DaysOver90Minor,
	}
}

func bucketTicks(ctx context.Context) []string {
	return []string{
		i18n.T(ctx, "Current"),
		i18n.T(ctx, "1–30"),
		i18n.T(ctx, "31–60"),
		i18n.T(ctx, "61–90"),
		i18n.T(ctx, "90+"),
	}
}

func bucketNames(ctx context.Context) []string {
	return []string{
		i18n.T(ctx, "Current"),
		i18n.T(ctx, "1 to 30 days past due"),
		i18n.T(ctx, "31 to 60 days past due"),
		i18n.T(ctx, "61 to 90 days past due"),
		i18n.T(ctx, "Over 90 days past due"),
	}
}

func grewMost(then, now []int64) int {
	grew, most := -1, int64(0)
	for idx := 1; idx < len(now); idx++ {
		if change := now[idx] - then[idx]; change > most {
			grew, most = idx, change
		}
	}

	return grew
}

func points(minor []int64) []*decimal.Decimal {
	out := make([]*decimal.Decimal, 0, len(minor))
	for _, value := range minor {
		amount := money.DecimalFromMinor(value)
		out = append(out, &amount)
	}

	return out
}

func moneyOf(minor int64) *boardcatalog.Value {
	value := boardcatalog.Money(minor, "")

	return &value
}

func (r *AgingCompare) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	earlier := earlierThan(req.Now, req.Period, req.Location)
	found, err := r.analytics.CompareAging(ctx, customerledgerports.CompareARAgingRequest{
		TenantInfo:  req.Tenant,
		AsOfDate:    req.Now,
		EarlierDate: earlier,
	})
	if err != nil {
		return nil, fmt.Errorf("read the aging at two dates: %w", err)
	}
	thenPoint, nowPoint := pointAt(found, earlier), pointAt(found, req.Now)

	return agingRead(ctx, &agingFacts{
		then:     thenPoint,
		now:      nowPoint,
		thenAt:   earlier,
		nowAt:    req.Now,
		location: req.Location,
	}), nil
}

func pointAt(
	found []*customerledgerports.ARAgingTrendPoint,
	at int64,
) *customerledgerports.ARAgingTrendPoint {
	for _, point := range found {
		if point != nil && point.PeriodEnd == at {
			return point
		}
	}

	return &customerledgerports.ARAgingTrendPoint{PeriodEnd: at}
}

type agingFacts struct {
	then, now     *customerledgerports.ARAgingTrendPoint
	thenAt, nowAt int64
	location      *time.Location
}

func agingRead(ctx context.Context, facts *agingFacts) *boardcatalog.MetricRead {
	thenDay, nowDay := dayText(
		ctx,
		facts.thenAt,
		facts.location,
		facts.nowAt,
	), dayText(
		ctx,
		facts.nowAt,
		facts.location,
		facts.nowAt,
	)
	then, now := bucketsOf(&facts.then.Buckets), bucketsOf(&facts.now.Buckets)
	labels := bucketTicks(ctx)
	grew := grewMost(then, now)

	ticks := make([]boardcatalog.Tick, 0, len(labels))
	for idx, label := range labels {
		ticks = append(ticks, boardcatalog.Tick{Label: label, Marked: idx == grew})
	}
	chart := &boardcatalog.Chart{
		Unit: boardcatalog.ValueMoney,
		X:    ticks,
		Series: []boardcatalog.Series{
			{Name: thenDay, Role: boardcatalog.RoleWas, Points: points(then)},
			{Name: nowDay, Role: boardcatalog.RoleNow, Points: points(now)},
		},
	}
	openInvoices := boardcatalog.Count(int64(facts.now.OpenInvoiceCount))
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Aging, {0} against {1}", nowDay, thenDay),
		Source: i18n.T(ctx,
			"{0, plural, one {# open invoice} other {# open invoices}}, as of each date",
			facts.now.OpenInvoiceCount),
		Chart: chart,
		Figures: map[string]*boardcatalog.Value{
			"openNow":         moneyOf(facts.now.Buckets.TotalOpenMinor),
			"openThen":        moneyOf(facts.then.Buckets.TotalOpenMinor),
			"openInvoicesNow": &openInvoices,
			"grewBucket":      nil,
			"grewBucketNow":   nil,
			"grewBucketThen":  nil,
		},
		Changes: map[string]boardcatalog.FigureShift{
			"openNow": {Was: moneyOf(facts.then.Buckets.TotalOpenMinor), WasAt: facts.thenAt},
		},
	}
	if grew < 0 {
		read.Figures["grewBucket"] = &boardcatalog.Value{
			Kind: boardcatalog.ValueText,
			Text: i18n.T(ctx, "No past-due bucket grew"),
		}

		return read
	}
	chart.Marked = []int{grew}
	read.Figures["grewBucket"] = &boardcatalog.Value{
		Kind: boardcatalog.ValueText,
		Text: bucketNames(ctx)[grew],
	}
	read.Figures["grewBucketNow"] = moneyOf(now[grew])
	read.Figures["grewBucketThen"] = moneyOf(then[grew])
	read.Changes["grewBucketNow"] = boardcatalog.FigureShift{
		Was:   moneyOf(then[grew]),
		WasAt: facts.thenAt,
		Tone:  boardcatalog.ToneWarn,
	}

	return read
}
