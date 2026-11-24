package workforceboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureExpiry        = "expiry"
	MeasureHoursThisWeek = "hoursThisWeek"
	MetricRotaWeek       = "rotaWeek"
)

var Kinds = []boardcatalog.Kind{
	{
		Kind: recordkind.Kind(permission.ResourceWorker),
		Called: []string{
			"driver", "drivers", "person", "persons", "people", "employee", "employees",
		},
	},
	{
		Kind:   recordkind.Kind(permission.ResourceWorkerCredential),
		PartOf: recordkind.Kind(permission.ResourceWorker),
	},
}

var weekHoursReads = append(
	boardcatalog.Read(permission.ResourceTimesheet, "workedMinutes"),
	boardcatalog.Read(permission.ResourceWorkerSchedule, "startTime", "endTime")...,
)

var Measures = []boardcatalog.Measure{
	{
		Name:      MeasureExpiry,
		Kind:      recordkind.Kind(permission.ResourceWorkerCredential),
		Shape:     boardcatalog.ShapeCountdown,
		Value:     boardcatalog.ValueDays,
		Ascending: true,
		Default:   true,
		Aliases: []string{
			"expiresAt", "expirationDate", "expires", "daysLeft", "expiration",
			"daysUntilExpiry", "expiry_date",
		},
		Reads:   boardcatalog.Read(permission.ResourceWorkerCredential, "expiresAt"),
		Words:   "time left before it expires, soonest first",
		Label:   "Time left",
		Total:   "Total time left",
		Largest: "Soonest to run out",
		Heading: "Left",
		FollowUp: &boardcatalog.FollowUp{
			Label: "Draft renewal reminders",
			Ask: "Draft a short renewal reminder to each person whose credential on this " +
				"board has lapsed or runs out soon, as a document I can read and send myself.",
		},
	},
	{
		Name:     MeasureHoursThisWeek,
		Kind:     recordkind.Kind(permission.ResourceWorker),
		Shape:    boardcatalog.ShapeLimit,
		Value:    boardcatalog.ValueHours,
		Summable: true,
		Adverse:  true,
		Aliases: []string{
			"projectedOvertime", "overtime", "overtimeHours", "overtimeMinutes", "hours",
			"weekHours", "hoursWorked", "worked", "overtimeRisk", "hoursAgainstOvertime",
		},
		Requires: []permission.Resource{
			permission.ResourceTimesheet, permission.ResourceWorkerSchedule,
		},
		Reads:   weekHoursReads,
		Words:   "hours this week against the overtime line, most over first",
		Label:   "Overtime projected",
		Total:   "Total overtime projected",
		Largest: "Furthest over",
		Heading: "Over",
	},
}

var Metrics = []boardcatalog.Metric{
	{
		Name:    MetricRotaWeek,
		Form:    boardcatalog.FormTable,
		Periods: []boardcatalog.Period{boardcatalog.PeriodThisWeek},
		Requires: []permission.Resource{
			permission.ResourceWorker,
			permission.ResourceTimesheet,
			permission.ResourceWorkerSchedule,
		},
		Aliases: []string{
			"rota", "weekRota", "rotaThisWeek", "overtimeWeek", "hoursByDay", "weekByDay",
			"shiftsThisWeek",
		},
		Words: "this week day by day: worked, still on the rota, the shift that " +
			"crosses the overtime line; figures over, room, onRota",
		Reads:  weekHoursReads,
		Sketch: boardcatalog.Sketch{Rows: 8},
		Figures: []boardcatalog.Figure{
			{
				Name:  "over",
				Label: "Past the overtime line this week",
				Value: boardcatalog.ValueCount,
			},
			{Name: "room", Label: "With room under the line", Value: boardcatalog.ValueCount},
			{Name: "onRota", Label: "On this week's rota", Value: boardcatalog.ValueCount},
		},
	},
}
