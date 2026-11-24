package workforceboardreader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func weekOf(worked, remaining, threshold int32) weekhours.Week {
	return weekhours.Week{
		Worked: worked, Remaining: remaining, Threshold: threshold, Shift: "Dock nights",
		CrossDay: -1,
	}
}

func TestHoursReadingStandsOverOnOrUnderTheLine(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	over := weekOf(690, 2220, 2400)
	reading := hoursReading(ctx, &over)
	assert.Equal(t, "{0} over", reading.Standing.Text)
	assert.True(t, reading.Standing.Value.Number.Equal(decimal.NewFromInt(510*60)))
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	assert.True(t, reading.Value.Number.Equal(decimal.NewFromInt(510*60)))
	assert.Equal(t, boardcatalog.ValueHours, reading.Value.Kind, "a week's overtime is hours worked")
	assert.Equal(t, boardcatalog.ValueHours, reading.Standing.Value.Kind)
	assert.Equal(t, boardcatalog.ValueHours, reading.Fact.Value.Kind)
	require.NotNil(t, reading.Limit)
	assert.True(t, reading.Limit.Used.Equal(decimal.NewFromInt(690)))
	assert.True(t, reading.Limit.Planned.Equal(decimal.NewFromInt(2910)))
	assert.True(t, reading.Limit.Bound.Equal(decimal.NewFromInt(2400)))
	assert.Equal(t, "{0} worked", reading.Fact.Text)
	assert.True(t, reading.Fact.Value.Number.Equal(decimal.NewFromInt(690*60)))
	assert.Equal(t, "on Dock nights", reading.Fact.Place)

	already := weekOf(2500, 0, 2400)
	assert.Equal(t, boardcatalog.ToneBad, hoursReading(ctx, &already).Tone)

	line := weekOf(480, 1920, 2400)
	reading = hoursReading(ctx, &line)
	assert.Equal(t, "On the line", reading.Standing.Text)
	assert.Nil(t, reading.Standing.Value)
	assert.True(t, reading.Value.Number.IsZero())

	under := weekOf(0, 960, 2400)
	reading = hoursReading(ctx, &under)
	assert.Equal(t, "{0} under", reading.Standing.Text)
	assert.True(t, reading.Standing.Value.Number.Equal(decimal.NewFromInt(1440*60)))
	assert.Empty(t, reading.Tone)
}

func TestHoursThisWeekIsReadOnlyWithTimesheetsAndTheRota(t *testing.T) {
	t.Parallel()

	catalog := boardcatalog.New(boardcatalog.Entries{
		Kinds:    workforceboard.Kinds,
		Measures: workforceboard.Measures,
		Metrics:  workforceboard.Metrics,
	})
	measure, ok := catalog.MeasureOf(
		recordkind.Kind(permission.ResourceWorker),
		"projectedOvertime",
	)
	require.True(t, ok)
	assert.Equal(t, workforceboard.MeasureHoursThisWeek, measure.Name)
	assert.Equal(t, boardcatalog.ValueHours, measure.Value)
	assert.ElementsMatch(t, []permission.Resource{
		permission.ResourceTimesheet, permission.ResourceWorkerSchedule,
	}, measure.Requires)
	metric, ok := catalog.MetricNamed("rota")
	require.True(t, ok)
	assert.Equal(t, workforceboard.MetricRotaWeek, metric.Name)
	assert.Contains(t, metric.Requires, permission.ResourceTimesheet)
	assert.Contains(t, metric.Requires, permission.ResourceWorkerSchedule)
}

type fakeWeeks struct {
	weeks   map[pulid.ID]*weekhoursports.WorkerWeek
	leading *weekhoursports.Leading
	asked   *weekhoursports.LeadingRequest
	workers []*weekhoursports.WorkersRequest
}

func (f *fakeWeeks) Workers(
	_ context.Context,
	req *weekhoursports.WorkersRequest,
) (map[pulid.ID]*weekhoursports.WorkerWeek, error) {
	f.workers = append(f.workers, req)

	return f.weeks, nil
}

func (f *fakeWeeks) Leading(
	_ context.Context,
	req *weekhoursports.LeadingRequest,
) (*weekhoursports.Leading, error) {
	f.asked = req

	return f.leading, nil
}

type fakeNarrow struct {
	ids []pulid.ID
	err error
}

func (f fakeNarrow) Narrow(
	context.Context,
	*permissionports.RequestActor,
	pagination.TenantInfo,
) ([]pulid.ID, error) {
	return f.ids, f.err
}

func TestHoursThisWeekLeavesOutAWorkerWithNoWeek(t *testing.T) {
	t.Parallel()

	onShift, idle := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	week := weekOf(600, 2400, 2400)
	reader := NewHoursThisWeek(
		WeekParams{Weeks: &fakeWeeks{weeks: map[pulid.ID]*weekhoursports.WorkerWeek{
			onShift: {WorkerID: onShift, Week: week},
			idle:    {WorkerID: idle, Week: weekhours.Week{Threshold: 2400, CrossDay: -1}},
		}}, Narrow: fakeNarrow{}},
	)

	readings, err := reader.Readings(context.Background(), &boardreaderports.RecordRequest{
		IDs: []pulid.ID{onShift, idle},
	})
	require.NoError(t, err)
	assert.Contains(t, readings, onShift)
	assert.NotContains(t, readings, idle)
}

func crew(name string, days [7]weekhours.Day, worked, remaining int32) *weekhoursports.WorkerWeek {
	return &weekhoursports.WorkerWeek{
		WorkerID: pulid.MustNew("wrk_"),
		Name:     name,
		Week: weekhours.Week{
			Days: days, Worked: worked, Remaining: remaining, Threshold: 2400, CrossDay: -1,
		},
	}
}

func TestRotaWeekDrawsTheWeekAndCountsOnlyWhenEveryWorkerWasRead(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.November, 22, 0, 0, 0, 0, time.UTC)
	days := [7]weekhours.Day{
		{State: weekhours.DayWorked, Minutes: 600},
		{State: weekhours.DayScheduled, Minutes: 600},
		{State: weekhours.DayOff},
		{State: weekhours.DayScheduled, Minutes: 600},
		{State: weekhours.DayScheduled, Minutes: 600},
		{State: weekhours.DayAway},
		{State: weekhours.DayScheduled, Minutes: 600, Crosses: true},
	}
	leading := &weekhoursports.Leading{
		Start:  start.Unix(),
		Rows:   []*weekhoursports.WorkerWeek{crew("Hector Alvarez", days, 600, 2400)},
		Room:   []*weekhoursports.WorkerWeek{crew("Rosa Lind", [7]weekhours.Day{}, 0, 960)},
		OnRota: 6,
		Over:   3,
		OnLine: 1,
		Exact:  true,
	}
	weeks := &fakeWeeks{leading: leading}
	reader := NewRotaWeek(WeekParams{Weeks: weeks, Narrow: fakeNarrow{}})

	read, err := reader.Read(context.Background(), &boardreaderports.MetricRequest{
		Location: time.UTC,
	})
	require.NoError(t, err)
	assert.Equal(t, rotaRows, weeks.asked.Limit)
	assert.Equal(t, rotaRoom, weeks.asked.Room)
	require.NotNil(t, read.Figures[figureOver])
	assert.True(t, read.Figures[figureOver].Number.Equal(decimal.NewFromInt(3)))
	assert.True(t, read.Figures[figureRoom].Number.Equal(decimal.NewFromInt(2)))
	assert.True(t, read.Figures[figureRota].Number.Equal(decimal.NewFromInt(6)))

	require.NotNil(t, read.Table)
	require.Len(t, read.Table.Columns, 9)
	assert.Equal(t, "Sun 22", read.Table.Columns[1].Label)
	assert.Equal(t, "Sat 28", read.Table.Columns[7].Label)
	require.Len(t, read.Table.Rows, 2)
	hector := read.Table.Rows[0]
	assert.Equal(t, "Hector Alvarez", hector[0].Text)
	assert.Equal(t, boardcatalog.MarkHot, hector[1].Mark)
	assert.Empty(t, hector[2].Mark)
	assert.Equal(t, boardcatalog.MarkOff, hector[3].Mark)
	assert.Equal(t, "Time off", hector[6].Text)
	assert.Equal(t, boardcatalog.MarkW, hector[7].Mark)
	assert.Equal(t, boardcatalog.MarkW, hector[8].Mark)
	assert.True(t, hector[8].Value.Number.Equal(decimal.NewFromInt(3000*60)))
	assert.Empty(t, read.Table.Rows[1][8].Mark)

	leading.Exact = false
	read, err = reader.Read(context.Background(), &boardreaderports.MetricRequest{})
	require.NoError(t, err)
	assert.Nil(t, read.Figures[figureOver], "a count over part of the rota is withheld")
	assert.Contains(t, read.Figures, figureRoom)
	assert.Nil(t, read.Figures[figureRoom])
	require.NotNil(t, read.Figures[figureRota], "who is on the rota is counted either way")
}

func TestRotaWeekNarrowsToTheTeamAndDrawsNothingForNoTeam(t *testing.T) {
	t.Parallel()

	managers := []pulid.ID{pulid.MustNew("usr_")}
	weeks := &fakeWeeks{leading: &weekhoursports.Leading{Exact: true}}
	_, err := NewRotaWeek(WeekParams{Weeks: weeks, Narrow: fakeNarrow{ids: managers}}).
		Read(context.Background(), &boardreaderports.MetricRequest{})
	require.NoError(t, err)
	assert.Equal(t, managers, weeks.asked.ManagerIDs)

	weeks.asked = nil
	read, err := NewRotaWeek(WeekParams{
		Weeks:  weeks,
		Narrow: fakeNarrow{err: weekhoursports.ErrNoTeam},
	}).Read(context.Background(), &boardreaderports.MetricRequest{})
	require.NoError(t, err)
	assert.Nil(t, weeks.asked)
	assert.Nil(t, read.Table)
	assert.Nil(t, read.Figures[figureOver])

	_, err = NewRotaWeek(WeekParams{
		Weeks:  weeks,
		Narrow: fakeNarrow{err: errors.New("down")},
	}).Read(context.Background(), &boardreaderports.MetricRequest{})
	require.Error(t, err)
}

func TestHoursThisWeekReadsOnlyTheWorkersOnTheReadersTeam(t *testing.T) {
	t.Parallel()

	manager, worker := pulid.MustNew("usr_"), pulid.MustNew("wrk_")
	weeks := &fakeWeeks{weeks: map[pulid.ID]*weekhoursports.WorkerWeek{
		worker: {WorkerID: worker, Week: weekOf(600, 2400, 2400)},
	}}
	actor := &permissionports.RequestActor{UserID: manager}
	reader := NewHoursThisWeek(
		WeekParams{Weeks: weeks, Narrow: fakeNarrow{ids: []pulid.ID{manager}}},
	)

	_, err := reader.Readings(context.Background(), &boardreaderports.RecordRequest{
		Actor: actor,
		IDs:   []pulid.ID{worker},
	})
	require.NoError(t, err)
	require.Len(t, weeks.workers, 1)
	assert.Equal(t, []pulid.ID{manager}, weeks.workers[0].ManagerIDs,
		"a team-scoped timesheet reader reads only their team's weeks")

	outside := NewHoursThisWeek(WeekParams{
		Weeks:  weeks,
		Narrow: fakeNarrow{err: weekhoursports.ErrNoTeam},
	})
	readings, err := outside.Readings(context.Background(), &boardreaderports.RecordRequest{
		Actor: actor,
		IDs:   []pulid.ID{worker},
	})
	require.NoError(t, err)
	assert.Empty(t, readings, "a reader whose team reaches no one reads no hours")
	assert.Len(t, weeks.workers, 1, "and nothing is read for them")
}
