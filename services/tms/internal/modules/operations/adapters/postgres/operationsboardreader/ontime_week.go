package operationsboardreader

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type OnTimeByWeek struct {
	db *postgres.Connection
}

func NewOnTimeByWeek(p OnTimeParams) *OnTimeByWeek {
	return &OnTimeByWeek{db: p.DB}
}

func (*OnTimeByWeek) Metric() string { return operationsboard.MetricOnTimeByWeek }

func (r *OnTimeByWeek) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	period := req.Period
	if period != boardcatalog.PeriodLastQuarter {
		period = boardcatalog.PeriodThisQuarter
	}
	loc := serviceZone(req.Location)
	first, last, _ := period.Days(req.Now, loc)
	window := newServiceWindow(first, last, req.Now, loc)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*boardcatalog.MetricRead, error) {
		var (
			days    []serviceDayRow
			targets []serviceTargetRow
		)
		if err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(serviceDaysQuery(window.readFrom(), window.readTo), &days),
			keel.IntoSlice(serviceTargetQuery(), &targets),
		); err != nil {
			return nil, err
		}
		weeks := foldServiceWeeks(days, &window)
		var target *float64
		if len(targets) > 0 && targets[0].Target != nil && *targets[0].Target > 0 {
			target = targets[0].Target
		}

		return onTimeRead(ctx, period, &weeks, target, req.Now), nil
	})
}

func onTimeRead(
	ctx context.Context,
	period boardcatalog.Period,
	weeks *serviceWeeks,
	target *float64,
	now int64,
) *boardcatalog.MetricRead {
	title := i18n.T(ctx, "On time by week, this quarter")
	if period == boardcatalog.PeriodLastQuarter {
		title = i18n.T(ctx, "On time by week, last quarter")
	}
	chart := &boardcatalog.Chart{
		Unit:   boardcatalog.ValuePercent,
		X:      make([]boardcatalog.Tick, len(weeks.weeks)),
		Series: []boardcatalog.Series{{Name: i18n.T(ctx, "On time"), Role: boardcatalog.RoleNow}},
	}
	points := make([]*decimal.Decimal, len(weeks.weeks))
	drawn := int64(0)
	for idx := range weeks.weeks {
		week := &weeks.weeks[idx]
		drawn += week.Delivered
		chart.X[idx] = boardcatalog.Tick{At: week.Monday.Unix(), Marked: idx == weeks.slip}
		points[idx] = serviceShare(week.OnTime, week.Delivered)
	}
	chart.Series[0].Points = points
	chart.Ghost = len(weeks.weeks) > 0 && weeks.weeks[len(weeks.weeks)-1].Running
	figures := map[string]*boardcatalog.Value{
		"average":    servicePercent(serviceShare(weeks.onTime, weeks.delivered)),
		"lowest":     nil,
		"lowestWeek": nil,
		"latest":     servicePercent(weeks.latest()),
		"target":     nil,
		"previous":   servicePercent(serviceShare(weeks.prevOnTime, weeks.prevDelivered)),
	}
	if weeks.slip != noServiceWeek {
		slip := weeks.slip
		chart.Marked = []int{slip}
		chart.Band = &boardcatalog.Band{From: &slip, To: &slip}
		figures["lowest"] = servicePercent(points[slip])
		figures["lowestWeek"] = serviceText(serviceWeekDay(ctx, weeks.weeks[slip].Monday, now))
	}
	if target != nil {
		value := decimal.NewFromFloat(*target).Div(decimal.NewFromInt(servicePercentPoints))
		chart.Target = &boardcatalog.Line{Value: value, Label: i18n.T(ctx, "Target")}
		figures["target"] = servicePercent(&value)
	}

	return &boardcatalog.MetricRead{
		Title: title,
		Source: i18n.T(
			ctx,
			"{0, plural, one {# delivery} other {# deliveries}}, "+
				"{1, plural, one {# week} other {# weeks}}",
			drawn,
			len(weeks.weeks),
		),
		Figures: figures,
		Chart:   chart,
	}
}
