package operationsboardreader

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicerollup"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const (
	serviceDaysPerWeek   = 7
	serviceQuarterMonths = 3
	serviceShareDigits   = 4
	servicePercentPoints = 100
	noServiceWeek        = -1
)

type OnTimeParams struct {
	fx.In

	DB *postgres.Connection
}

type serviceDayRow struct {
	Bucket    time.Time
	Delivered int64
	OnTime    int64
	Late      int64
}

type serviceTargetRow struct {
	Target *float64
}

type serviceWeek struct {
	Monday    time.Time
	Delivered int64
	OnTime    int64
	Running   bool
}

type serviceWindow struct {
	first     time.Time
	last      time.Time
	prevFirst time.Time
	prevLast  time.Time
	start     time.Time
	readTo    time.Time
	today     time.Time
	loc       *time.Location
}

type serviceWeeks struct {
	weeks         []serviceWeek
	slip          int
	delivered     int64
	onTime        int64
	prevDelivered int64
	prevOnTime    int64
}

func serviceZone(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}

	return loc
}

func serviceLocalDay(at time.Time, loc *time.Location) time.Time {
	at = at.In(loc)

	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
}

func serviceMonday(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + serviceDaysPerWeek - 1) % serviceDaysPerWeek))
}

func serviceDaysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	return int(b.Sub(a).Hours() / 24)
}

func newServiceWindow(first, last time.Time, now int64, loc *time.Location) serviceWindow {
	today := serviceLocalDay(time.Unix(now, 0), loc)
	start := serviceMonday(first)
	readTo := serviceMonday(last).AddDate(0, 0, serviceDaysPerWeek-1)
	if readTo.After(today) {
		readTo = today
	}

	return serviceWindow{
		first:     first,
		last:      last,
		prevFirst: first.AddDate(0, -serviceQuarterMonths, 0),
		prevLast:  first.AddDate(0, 0, -1),
		start:     start,
		readTo:    readTo,
		today:     today,
		loc:       loc,
	}
}

func (w *serviceWindow) readFrom() time.Time {
	if w.prevFirst.Before(w.start) {
		return w.prevFirst
	}

	return w.start
}

func (w *serviceWindow) weekCount() int {
	return serviceDaysBetween(w.start, w.readTo)/serviceDaysPerWeek + 1
}

func serviceDaysQuery(from, to time.Time) keel.Projection {
	t := servicerollup.DailyTable

	return keel.RowsOf(t.Ref).
		Group(t.Bucket()).
		Pick(
			t.Bucket(),
			t.Count(servicerollup.Delivered),
			t.Count(servicerollup.OnTime),
			t.Count(servicerollup.Late),
		).
		Where(t.Bucket().Gte(from), t.Bucket().Lte(to)).
		Order(t.Bucket().Asc())
}

func serviceTargetQuery() keel.Projection {
	dc := dispatchcontroldb.DispatchControls

	return dc.Pick(dc.ServiceFailureTarget).Limit(1)
}

func foldServiceWeeks(days []serviceDayRow, window *serviceWindow) serviceWeeks {
	count := window.weekCount()
	out := serviceWeeks{weeks: make([]serviceWeek, count), slip: noServiceWeek}
	for idx := range out.weeks {
		out.weeks[idx].Monday = window.start.AddDate(0, 0, serviceDaysPerWeek*idx)
	}
	out.weeks[count-1].Running = !serviceMonday(window.today).After(out.weeks[count-1].Monday)
	for idx := range days {
		row := &days[idx]
		day := serviceLocalDay(row.Bucket, window.loc)
		if !day.Before(window.prevFirst) && !day.After(window.prevLast) {
			out.prevDelivered += row.Delivered
			out.prevOnTime += row.OnTime
		}
		if !day.Before(window.first) && !day.After(window.last) {
			out.delivered += row.Delivered
			out.onTime += row.OnTime
		}
		if day.Before(window.start) || day.After(window.readTo) {
			continue
		}
		week := &out.weeks[serviceDaysBetween(window.start, day)/serviceDaysPerWeek]
		week.Delivered += row.Delivered
		week.OnTime += row.OnTime
	}
	var lowest *decimal.Decimal
	for idx := range out.weeks {
		week := &out.weeks[idx]
		value := serviceShare(week.OnTime, week.Delivered)
		if week.Running || value == nil {
			continue
		}
		if lowest == nil || value.LessThan(*lowest) {
			lowest, out.slip = value, idx
		}
	}

	return out
}

func serviceShare(onTime, delivered int64) *decimal.Decimal {
	if delivered <= 0 {
		return nil
	}
	value := decimal.NewFromInt(onTime).DivRound(decimal.NewFromInt(delivered), serviceShareDigits)

	return &value
}

func servicePercent(value *decimal.Decimal) *boardcatalog.Value {
	if value == nil {
		return nil
	}
	out := boardcatalog.Percent(*value)

	return &out
}

func serviceCount(n int64) *boardcatalog.Value {
	out := boardcatalog.Count(n)

	return &out
}

func serviceText(text string) *boardcatalog.Value {
	return &boardcatalog.Value{Kind: boardcatalog.ValueText, Text: text}
}

func serviceWeekDay(ctx context.Context, monday time.Time, now int64) string {
	return i18n.Day(i18n.FromContext(ctx), monday, time.Unix(now, 0).In(monday.Location()))
}

func (w *serviceWeeks) latest() *decimal.Decimal {
	for idx := len(w.weeks) - 1; idx >= 0; idx-- {
		if w.weeks[idx].Running {
			continue
		}
		return serviceShare(w.weeks[idx].OnTime, w.weeks[idx].Delivered)
	}

	return nil
}
