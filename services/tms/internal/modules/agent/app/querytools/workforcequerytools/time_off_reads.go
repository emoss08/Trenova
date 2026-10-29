package workforcequerytools

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolListTimeOff = "worker_list_time_off"
	// defaultTimeOffWindowDays is how far ahead an unqualified question
	// reaches. "Who is off?" means this week and next, not the whole year of
	// accrued leave, and a model handed a year of rows summarises the wrong
	// ones.
	defaultTimeOffWindowDays = 14

	timeOffSummary = "List worker (driver) time-off requests: who is out, who is asking to be, " +
		"and who was turned down. The tool for availability by date, and the one that yields " +
		"the request ids the approve, reject and cancel tools need."
	timeOffFrom        = "startDateFrom"
	timeOffTo          = "startDateTo"
	timeOffOverlapping = "overlapping"
	timeOffIncludes    = "includeWorker"
	timeOffWorkerID    = "workerId"
	noteTimeOffStatus  = "Requested means still awaiting a decision."
	noteTimeOffDefault = "Defaults to true."
)

var (
	timeOffSearchTerms = []string{
		"who is off", "time off calendar", "booked time off", "out of office",
	}
	noteTimeOffWindow = fmt.Sprintf("Give neither date for leave on at any point from today "+
		"through %d days out.", defaultTimeOffWindowDays)
)

// TimeOffReadSpecs are the time-off requests, read through the operation
// the time-off table sends. A call that names no dates reads the leave on
// in the next two weeks, overlapping the window as the repository tool's
// withinDays did, with each request's worker as the table shows it.
func TimeOffReadSpecs() []readspec.List {
	return []readspec.List{{
		Name:         toolListTimeOff,
		EntityPlural: "time-off requests",
		Summary:      timeOffSummary,
		SearchTerms:  timeOffSearchTerms,
		Resource:     permission.ResourceWorkerPTO,
		Notes: map[string]string{
			querytool.ParamStatus: noteTimeOffStatus,
			timeOffFrom:           noteTimeOffWindow,
			timeOffOverlapping:    noteTimeOffDefault,
		},
		References: map[string]readspec.Reference{timeOffWorkerID: readspec.WorkerReference},
		Defaults:   map[string]any{timeOffOverlapping: true},
		DefaultsAt: timeOffWindow,
		Inputs:     map[string]any{timeOffIncludes: true},
	}}
}

// timeOffWindow is today through the last day of the default window, by the
// calendar in the organization's zone, so a change of clocks inside it moves
// neither end: leave that started this morning is leave on now.
func timeOffWindow(clk filtercatalog.Clock) map[string]any {
	today := time.Unix(clk.Today(), 0).In(clk.Location())
	last := today.AddDate(0, 0, defaultTimeOffWindowDays)

	return map[string]any{
		timeOffFrom: today.Format(timeutils.ISODateLayout),
		timeOffTo:   last.Format(timeutils.ISODateLayout),
	}
}
