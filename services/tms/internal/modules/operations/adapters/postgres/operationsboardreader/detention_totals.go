package operationsboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type DetentionTotals struct {
	intelligence detentionports.DetentionIntelligenceRepository
}

func NewDetentionTotals(p DetentionParams) *DetentionTotals {
	return &DetentionTotals{intelligence: p.Intelligence}
}

func (*DetentionTotals) Metric() string { return detentionboard.MetricDetentionTotals }

func (r *DetentionTotals) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	ask := &detentionAsk{
		tenant:   req.Tenant,
		period:   req.Period,
		now:      req.Now,
		location: req.Location,
	}
	window, ok := ask.window()
	if !ok {
		return &boardcatalog.MetricRead{}, nil
	}
	summary, err := r.intelligence.TotalFigures(ctx, window.req)
	if err != nil {
		return nil, err
	}
	summary.Totals.WithholdFields(req.Visible)

	return detentionTotals(ctx, req.Period, summary), nil
}

func detentionTotals(
	ctx context.Context,
	period boardcatalog.Period,
	summary *detentionports.IntelligenceSummary,
) *boardcatalog.MetricRead {
	totals := &summary.Totals
	ready := summary.Bucket(detentionports.RecoveryPending)
	readyToBill := boardcatalog.Amount(ready.BillableAmount, "")
	readyStays := boardcatalog.Count(int64(ready.OccurrenceCount))
	pastFree := boardcatalog.Duration(totals.PastFreeMinutes * stayMinute)
	billed := boardcatalog.Amount(totals.BilledAmount, "")
	waived := boardcatalog.Amount(totals.WaivedAmount, "")
	figures := map[string]*boardcatalog.Value{
		"readyToBill":      &readyToBill,
		"readyToBillStays": &readyStays,
		"pastFreeTime":     &pastFree,
		"billed":           &billed,
		"waived":           &waived,
		"noticeOnTime":     nil,
	}
	if totals.NoticeDueCount > 0 {
		share := boardcatalog.Percent(decimal.NewFromInt(int64(totals.NoticeOnTimeCount)).
			Div(decimal.NewFromInt(int64(totals.NoticeDueCount))))
		figures["noticeOnTime"] = &share
	}

	return &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Detention {0}", detentionPeriodWords(ctx, period)),
		Source: i18n.T(
			ctx,
			"{0, plural, one {# detained stop} other {# detained stops}} at every facility, {1}",
			totals.DetainedCount,
			detentionPeriodWords(ctx, period),
		),
		Figures: figures,
	}
}

func detentionPeriodWords(ctx context.Context, period boardcatalog.Period) string {
	switch period {
	case boardcatalog.PeriodThisQuarter:
		return i18n.T(ctx, "this quarter")
	case boardcatalog.PeriodLastQuarter:
		return i18n.T(ctx, "last quarter")
	case boardcatalog.PeriodThisMonth:
		return i18n.T(ctx, "this month")
	case boardcatalog.PeriodLastMonth:
		return i18n.T(ctx, "last month")
	case boardcatalog.PeriodThisWeek:
		return i18n.T(ctx, "this week")
	case boardcatalog.PeriodLastWeek:
		return i18n.T(ctx, "last week")
	case boardcatalog.PeriodToday:
		return i18n.T(ctx, "today")
	case boardcatalog.PeriodYesterday:
		return i18n.T(ctx, "yesterday")
	case boardcatalog.PeriodNextWeek:
		return i18n.T(ctx, "next week")
	case boardcatalog.PeriodNext48Hours:
		return i18n.T(ctx, "in the next 48 hours")
	case boardcatalog.PeriodNext30Days:
		return i18n.T(ctx, "in the next 30 days")
	case boardcatalog.PeriodSlip:
	}

	return ""
}
