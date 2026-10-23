package agentquerytoolservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	toolWorkerGetSchedule  = "worker_get_schedule"
	toolListDOTRandomDraws = "worker_list_dot_random_draws"
	toolListShiftTemplates = "worker_list_shift_templates"
	opWorkerAvailability   = "WorkerAvailabilityPreferences"
	opShiftSwapRequests    = "ShiftSwapRequests"
	opDOTRandomPools       = "DotRandomPools"
	paramPoolID            = "poolId"
	paramActiveOnly        = "activeOnly"
	shiftDaysOfWeek        = "daysOfWeek"
	shiftStartMinute       = "startMinute"
	shiftDurationMinutes   = "durationMinutes"
	openSwapsShown         = 20
	daysInWeek             = 7

	workerScheduleSummary       = "Read one worker's schedule: their shift patterns, what they said about each weekday, and their open shift swaps. Items are the patterns they have been on, the current one open-ended; preferences give each weekday's statement (dayOfWeek 0 for Sunday); openSwaps their open swaps. An item's id is the shiftAssignmentId and a swap's id the swapId the scheduling tools take."
	dotRandomDrawsSummary       = "List the rounds drawn for random drug and alcohol testing, newest first, beside the active pools they are drawn from (pools, with their annual rates). Each round is draft, final or cancelled, with how many drivers each substance needed and got. It yields the poolId and drawId the random testing tools take; worker_get_dot_random_draw names who was picked."
	shiftTemplatesSummary       = "List the organization's active shift patterns: the days, start time and length of each, and how many weeks it rotates over. worker_assign_shift takes the id."
	noteActivePatternsByDefault = "Defaults to true, the patterns in use only."
	noteEveryPool               = "Leave it out for every pool's."
)

// workforceProgrammeReadSpecs are the scheduling and random testing reads,
// each through the operation its page sends: a worker's schedule with what
// they said about each weekday and their open swaps beside it, the shift
// patterns, and the random testing rounds with the pools beside them.
func workforceProgrammeReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolWorkerGetSchedule,
			entityPlural: "shift assignments",
			summary:      workerScheduleSummary,
			resource:     permission.ResourceWorkerSchedule,
			references:   theWorker,
			alongside: []alongsideSpec{
				{
					key:       "preferences",
					operation: opWorkerAvailability,
					what:      "availability preferences",
				},
				{
					key:       "openSwaps",
					operation: opShiftSwapRequests,
					what:      "open shift swaps",
					inputs:    map[string]any{wfParamOpenOnly: true, paramLimit: openSwapsShown},
				},
			},
		},
		{
			name:         toolListShiftTemplates,
			entityPlural: "shift patterns",
			summary:      shiftTemplatesSummary,
			resource:     permission.ResourceShiftTemplate,
			notes:        map[string]string{paramActiveOnly: noteActivePatternsByDefault},
			defaults:     map[string]any{paramActiveOnly: true, paramLimit: graphQLListLimit},
			shape:        shapeShiftTemplate,
		},
		{
			name:         toolListDOTRandomDraws,
			entityPlural: "random testing rounds",
			summary:      dotRandomDrawsSummary,
			resource:     permission.ResourceDOTRandomPool,
			references: map[string]idReference{paramPoolID: {
				resource: permission.ResourceDOTRandomPool,
				what:     "The random testing pool",
				source:   "the pools this list returns or the page you are on",
				kinds:    []permission.RecordKind{permission.KindDOTRandomPool},
			}},
			notes: map[string]string{paramPoolID: noteEveryPool},
			alongside: []alongsideSpec{{
				key:       "pools",
				operation: opDOTRandomPools,
				what:      "random testing pools",
				inputs:    map[string]any{inputStatus: activeStatus},
			}},
		},
	}
}

// shapeShiftTemplate writes a pattern's days, start and length as a person
// says them (Mon Tue Wed, 06:00, 10:00) beside the stored mask and minutes.
func shapeShiftTemplate(row map[string]any, _ map[string]struct{}) {
	if mask, ok := row[shiftDaysOfWeek].(string); ok {
		row["days"] = shiftDays(mask)
	}
	if start := optionalInt(row, shiftStartMinute, -1); start >= 0 {
		row["starts"] = time.Date(0, 1, 1, 0, start, 0, 0, time.UTC).Format("15:04")
	}
	if length := optionalInt(row, shiftDurationMinutes, -1); length >= 0 {
		row["hours"] = fmt.Sprintf("%d:%02d", length/minutesPerHour, length%minutesPerHour)
	}
}

// shiftDays reads the pattern's seven flags, Sunday first.
func shiftDays(flags string) string {
	days := make([]string, 0, len(flags))
	for index, flag := range flags {
		if flag == '1' && index < daysInWeek {
			days = append(days, time.Weekday(index).String()[:3])
		}
	}
	if len(days) == 0 {
		return "none"
	}

	return strings.Join(days, " ")
}
