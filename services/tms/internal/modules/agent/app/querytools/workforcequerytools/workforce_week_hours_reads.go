package workforcequerytools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramWeekHoursLimit  = "limit"
	defaultWeekHoursRows = 12
	weekHoursRoomRows    = 3
	timesheetHoursField  = "workedMinutes"
	weekHoursWithheld    = "This agent's data access does not reach timesheet hours, so no " +
		"worker's week is shown."
)

type weekHoursDay struct {
	Day   string `json:"day"`
	State string `json:"state"`
	Hours string `json:"hours,omitempty"`
}

type weekHoursRow struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Shift             string         `json:"shift,omitempty"`
	Worked            string         `json:"worked"`
	StillScheduled    string         `json:"stillScheduled"`
	OvertimeLine      string         `json:"overtimeLine"`
	ProjectedOvertime string         `json:"projectedOvertime"`
	RoomUnderLine     string         `json:"roomUnderLine"`
	CrossesOn         string         `json:"crossesOn,omitempty"`
	Days              []weekHoursDay `json:"days"`
}

type weekHoursResult struct {
	WeekStart  querytool.OptionalDate `json:"weekStart"`
	WeekEnd    querytool.OptionalDate `json:"weekEnd"`
	OnRota     int64                  `json:"workersOnRota"`
	PastLine   *int64                 `json:"pastTheLine,omitempty"`
	Partial    string                 `json:"partial,omitempty"`
	Workers    []weekHoursRow         `json:"workers"`
	WithRoom   []weekHoursRow         `json:"withRoom"`
	Restricted string                 `json:"restricted,omitempty"`
	HowToShow  string                 `json:"howToShow,omitempty"`
	Withheld   []string               `json:"withheldByAccess,omitempty"`
}

func clockHours(minutes int32) string {
	return fmt.Sprintf(
		"%d:%02d",
		minutes/querytool.MinutesPerHour,
		minutes%querytool.MinutesPerHour,
	)
}

func weekHoursRowOf(worker *weekhoursports.WorkerWeek, loc *time.Location) weekHoursRow {
	week := &worker.Week
	row := weekHoursRow{
		ID:                worker.WorkerID.String(),
		Name:              worker.Name,
		Shift:             week.Shift,
		Worked:            clockHours(week.Worked),
		StillScheduled:    clockHours(week.Remaining),
		OvertimeLine:      clockHours(week.Threshold),
		ProjectedOvertime: clockHours(week.Overtime()),
		RoomUnderLine:     clockHours(week.Room()),
		Days:              make([]weekHoursDay, 0, weekhours.DaysInWeek),
	}
	for idx := range week.Days {
		day := &week.Days[idx]
		label := time.Unix(day.Date, 0).In(loc).Format("Mon Jan 2")
		entry := weekHoursDay{Day: label, State: string(day.State)}
		if day.Minutes > 0 {
			entry.Hours = clockHours(day.Minutes)
		}
		if day.Crosses {
			row.CrossesOn = label
		}
		row.Days = append(row.Days, entry)
	}

	return row
}

func newListWeekHoursTool(
	weeks weekhoursports.Reader,
	narrow weekhoursports.TeamNarrower,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &workforceRead{
		name: "worker_list_week_hours",
		description: "List this week's hours for the workers on the rota against their " +
			"overtime line: worked so far, still scheduled on their shifts, projected " +
			"overtime and the day the shift that crosses the line falls on, most projected " +
			"overtime first, then those with the most room. Each worker's id is the workerId " +
			"the worker tools take. Its howToShow says how to show the week.",
		searchTerms: []string{"overtime", "hours this week", "rota", "timesheet", "shifts"},
		resource:    permission.ResourceTimesheet,
		access:      fieldgate.NewAccess(permissions),
		properties: map[string]any{
			paramWeekHoursLimit: map[string]any{
				toolschema.KeyType: toolschema.TypeInteger,
				toolschema.KeyDescription: fmt.Sprintf(
					"How many workers nearest or past the line, %d by default, at most %d.",
					defaultWeekHoursRows, weekhoursports.MaxLeading,
				),
				toolschema.KeyMinimum: 1,
				toolschema.KeyMaximum: weekhoursports.MaxLeading,
			},
		},
		run: func(
			ctx context.Context,
			params *agenttoolports.QueryToolParams,
			gate *fieldgate.Gate,
		) (any, error) {
			return readWeekHours(ctx, &weekHoursCall{
				weeks:       weeks,
				narrow:      narrow,
				params:      params,
				hoursHidden: !gate.Show(timesheetHoursField, "hours"),
			})
		},
	}
}

type weekHoursCall struct {
	weeks       weekhoursports.Reader
	narrow      weekhoursports.TeamNarrower
	params      *agenttoolports.QueryToolParams
	hoursHidden bool
}

func readWeekHours(ctx context.Context, call *weekHoursCall) (*weekHoursResult, error) {
	if call.hoursHidden {
		return &weekHoursResult{
			Workers:    []weekHoursRow{},
			WithRoom:   []weekHoursRow{},
			Restricted: weekHoursWithheld,
			Withheld:   []string{"hours"},
		}, nil
	}
	tenant := querytool.TenantOf(call.params)
	managers, err := call.narrow.Narrow(ctx, call.params.Actor, tenant)
	if errors.Is(err, weekhoursports.ErrNoTeam) {
		return &weekHoursResult{
			Workers:    []weekHoursRow{},
			WithRoom:   []weekHoursRow{},
			Restricted: weekhoursports.ErrNoTeam.Error(),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	clock := querytool.ClockFor(call.params)
	loc := clock.Location()
	limit := querytool.OptionalInt(call.params.Params, paramWeekHoursLimit, defaultWeekHoursRows)
	leading, err := call.weeks.Leading(ctx, &weekhoursports.LeadingRequest{
		Tenant:     tenant,
		ManagerIDs: managers,
		Now:        clock.Instant(),
		Location:   loc,
		Limit:      min(max(limit, 1), weekhoursports.MaxLeading) + weekHoursRoomRows,
		Room:       weekHoursRoomRows,
	})
	if err != nil {
		return nil, err
	}

	result := &weekHoursResult{
		WeekStart: querytool.RecordedDate(leading.Start),
		WeekEnd:   querytool.RecordedDate(leading.End - 1),
		OnRota:    leading.OnRota,
		Workers:   make([]weekHoursRow, 0, len(leading.Rows)),
		WithRoom:  make([]weekHoursRow, 0, len(leading.Room)),
		HowToShow: boardhint.Line(&boardhint.Show{
			Metric: workforceboard.MetricRotaWeek,
			Records: &boardhint.Records{
				Words:   "these workers",
				Measure: workforceboard.MeasureHoursThisWeek,
			},
		}),
	}
	if leading.Exact {
		result.PastLine = &leading.Over
	} else {
		result.Partial = "More workers are on the rota than one read covers; the count of " +
			"workers past the line is left out rather than guessed."
	}
	for _, worker := range leading.Rows {
		result.Workers = append(result.Workers, weekHoursRowOf(worker, loc))
	}
	for _, worker := range leading.Room {
		result.WithRoom = append(result.WithRoom, weekHoursRowOf(worker, loc))
	}

	return result, nil
}

func provideListWeekHoursTool(
	weeks weekhoursports.Reader,
	narrow weekhoursports.TeamNarrower,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListWeekHoursTool(weeks, narrow, permissions)
}
