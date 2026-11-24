package operationsboardreader

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicerollup"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

const lateCustomersShown = 4

var lateCustomerKind = recordkind.Kind(permission.ResourceCustomer)

type LateByCustomer struct {
	db *postgres.Connection
}

func NewLateByCustomer(p OnTimeParams) *LateByCustomer {
	return &LateByCustomer{db: p.DB}
}

func (*LateByCustomer) Metric() string { return operationsboard.MetricLateByCustomer }

type lateCustomerRow struct {
	CustomerID pulid.ID
	Name       string
	Late       int64
	Delivered  int64
}

type lateTotalsRow struct {
	Delivered int64
	Late      int64
	Customers int64
}

type lateWeek struct {
	monday time.Time
	last   time.Time
}

func (r *LateByCustomer) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	loc := serviceZone(req.Location)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*boardcatalog.MetricRead, error) {
		week, err := r.week(ctx, req, loc)
		if err != nil {
			return nil, err
		}
		var (
			rows   []lateCustomerRow
			totals []lateTotalsRow
		)
		if err = keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(lateCustomersQuery(week.monday, week.last), &rows),
			keel.IntoSlice(lateTotalsQuery(week.monday, week.last), &totals),
		); err != nil {
			return nil, err
		}
		var total lateTotalsRow
		if len(totals) > 0 {
			total = totals[0]
		}

		return lateRead(ctx, week, rows, total, req.Now), nil
	})
}

func (r *LateByCustomer) week(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
	loc *time.Location,
) (lateWeek, error) {
	today := serviceLocalDay(time.Unix(req.Now, 0), loc)
	thisMonday := serviceMonday(today)
	lastWeek := lateWeek{
		monday: thisMonday.AddDate(0, 0, -serviceDaysPerWeek),
		last:   thisMonday.AddDate(0, 0, -1),
	}
	if req.Period == boardcatalog.PeriodThisWeek {
		return lateWeek{monday: thisMonday, last: today}, nil
	}
	if req.Period == boardcatalog.PeriodLastWeek {
		return lastWeek, nil
	}

	first, last, _ := boardcatalog.PeriodThisQuarter.Days(req.Now, loc)
	window := newServiceWindow(first, last, req.Now, loc)
	var days []serviceDayRow
	if err := keel.Batch(ctx, r.db.Keel(ctx),
		keel.IntoSlice(serviceDaysQuery(window.start, window.readTo), &days),
	); err != nil {
		return lateWeek{}, err
	}
	weeks := foldServiceWeeks(days, &window)
	if weeks.slip == noServiceWeek {
		return lastWeek, nil
	}
	monday := weeks.weeks[weeks.slip].Monday

	return lateWeek{monday: monday, last: monday.AddDate(0, 0, serviceDaysPerWeek-1)}, nil
}

func lateCustomersQuery(from, to time.Time) keel.Projection {
	t := servicerollup.CustomerDailyTable
	cus := customerdb.Customers
	customer := t.Col(servicerollup.Customer)
	late := t.Count(servicerollup.Late)

	return keel.RowsOf(t.Ref).
		LeftJoin(cus, cus.ID.EqCol(customer)).
		Group(customer, cus.Name).
		Pick(
			keel.StringOf(customer).Coalesce(""),
			cus.Name.Coalesce(""),
			late,
			t.Count(servicerollup.Delivered),
		).
		Where(t.Bucket().Gte(from), t.Bucket().Lte(to)).
		Having(late.Gt(0)).
		Order(keel.Desc(late), customer.Asc()).
		Limit(lateCustomersShown)
}

func lateTotalsQuery(from, to time.Time) keel.Projection {
	t := servicerollup.CustomerDailyTable
	customer := t.Col(servicerollup.Customer)

	return keel.RowsOf(t.Ref).
		Pick(
			t.Count(servicerollup.Delivered),
			t.Count(servicerollup.Late),
			customer.CountDistinct().Filter(keel.Int64Of(t.Col(servicerollup.Late)).Gt(0)),
		).
		Where(t.Bucket().Gte(from), t.Bucket().Lte(to))
}

func lateRead(
	ctx context.Context,
	week lateWeek,
	rows []lateCustomerRow,
	total lateTotalsRow,
	now int64,
) *boardcatalog.MetricRead {
	chart := &boardcatalog.Chart{
		Unit: boardcatalog.ValueCount,
		Rows: make([]boardcatalog.ChartRow, 0, len(rows)+1),
	}
	figures := map[string]*boardcatalog.Value{
		"late":        serviceCount(total.Late),
		"delivered":   serviceCount(total.Delivered),
		"top":         nil,
		"topCustomer": nil,
	}
	shown := int64(0)
	for idx := range rows {
		row := &rows[idx]
		shown += row.Late
		value := boardcatalog.Count(row.Late)
		chartRow := boardcatalog.ChartRow{
			Label:  row.Name,
			Value:  &value,
			Marked: idx == 0,
		}
		if row.CustomerID.IsNotNil() {
			chartRow.Ref = &boardcatalog.Ref{Kind: lateCustomerKind, ID: row.CustomerID}
		}
		if chartRow.Label == "" {
			chartRow.Label = i18n.T(ctx, "No customer")
		}
		chart.Rows = append(chart.Rows, chartRow)
	}
	if len(rows) > 0 {
		figures["top"] = serviceCount(rows[0].Late)
		figures["topCustomer"] = serviceText(chart.Rows[0].Label)
	}
	if others := total.Customers - int64(len(rows)); others > 0 && total.Late > shown {
		value := boardcatalog.Count(total.Late - shown)
		chart.Rows = append(chart.Rows, boardcatalog.ChartRow{
			Label: i18n.T(ctx,
				"{0, plural, one {# other customer} other {# other customers}}", others),
			Value: &value,
			Rest:  true,
		})
	}

	return &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Late in the week of {0}, by customer",
			serviceWeekDay(ctx, week.monday, now)),
		Source: i18n.T(ctx,
			"{0, plural, one {# late delivery} other {# late deliveries}}, {1} to {2}",
			total.Late, serviceWeekDay(ctx, week.monday, now), serviceWeekDay(ctx, week.last, now)),
		Figures: figures,
		Chart:   chart,
	}
}
