package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

const (
	paramReportDefinition = "definitionId"
	fieldRunStatus        = "status"
	fieldRunRowsStored    = "rowsStored"
	fieldRunComparable    = "comparable"
	fieldRunReason        = "reason"
	fieldRunQueuedAt      = "queuedAt"
	fieldRunStartedAt     = "startedAt"
	fieldAlertConditions  = "conditions"
	fieldAlertPriority    = "priority"
	fieldAlertRecordID    = "recordId"
	fieldAlertUpdatedAt   = "updatedAt"

	reportDefinitionSource = "report_list"
)

// savedReportReference is the saved report a report read is narrowed to.
var savedReportReference = idReference{
	resource: permission.ResourceReport,
	what:     "The saved report",
	source:   reportDefinitionSource,
}

// reportReadSpecs are the report lists, each read through the operation the
// Desk reads it with.
func reportReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		reportRunsListSpec(), reportSchedulesListSpec(), tableChangeAlertsListSpec(),
	}
}

// reportRunsListSpec reads the runs the person made of their reports.
func reportRunsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "report_list_runs",
		entityPlural: "report runs",
		summary: "List recent runs the person you are acting for made of a report, newest " +
			"first, so you can name the two worth comparing; another person's runs are never " +
			"listed. Use this before report_compare_runs, which needs run ids and " +
			"has no other way to learn them. Each row says whether it can be compared " +
			"(comparable) and why not — a run whose rows were not stored cannot be, and " +
			"there is no way to recover them short of running the report again.",
		queryNote: "Words to find in a run's status, format or built-in report key.",
		notes: map[string]string{
			"cannedKey": "A built-in report's reportKey, from report_list.",
			fieldRunStatus: "queued, running, succeeded, failed, canceled or expired; only " +
				"a succeeded run can be compared.",
		},
		references: map[string]idReference{paramReportDefinition: savedReportReference},
		unoffered: []string{
			"artifactExpiresAt", "byteSize", "cacheHit", "cannedVersion", "durationMs",
			fieldRunQueuedAt, fieldRunStartedAt, "revisionId",
		},
		resource: permission.ResourceReport,
		shape:    runComparable,
	}
}

// reportSchedulesListSpec reads the reports on a schedule.
func reportSchedulesListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "report_list_schedules",
		entityPlural: "report schedules",
		summary: "List reports on a schedule: when each runs, in what formats, who it is " +
			"emailed to, whether it is on and when it runs next. Only the person who set a " +
			"schedule up can change it with report_update_schedule or remove it with " +
			"report_delete_schedule; mine says which are theirs.",
		searchTerms: []string{"scheduled", "emailed", "recurring", "subscription"},
		references:  map[string]idReference{paramReportDefinition: savedReportReference},
		defaults:    listDefaults,
		resource:    permission.ResourceReport,
	}
}

// runComparable says whether a run can be compared with another, and why
// not: only a run that succeeded and whose rows were stored can be. A run
// whose status or storage was withheld is left unjudged rather than called
// incomparable.
func runComparable(row map[string]any, withheld map[string]struct{}, _ clock) {
	if _, hidden := withheld[fieldRunStatus]; hidden {
		return
	}
	if _, hidden := withheld[fieldRunRowsStored]; hidden {
		return
	}

	status := report.RunStatus(stringOf(row[fieldRunStatus]))
	stored, _ := row[fieldRunRowsStored].(bool)
	row[fieldRunComparable] = false
	switch {
	case !status.IsTerminal():
		row[fieldRunReason] = "This run is still " + string(status) +
			"; it can be compared once it succeeds."
	case status != report.RunStatusSucceeded:
		row[fieldRunReason] = "This run ended as " + string(status) + ", so it has no rows."
	case !stored:
		row[fieldRunReason] = "This run was generated before its rows were stored, so it " +
			"cannot be compared."
	default:
		row[fieldRunComparable] = true
	}
}

// tableChangeAlertsListSpec reads the change alerts the person set up; the
// repository keeps the read to the reader's own alerts.
func tableChangeAlertsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "report_list_table_change_alerts",
		entityPlural: "table change alerts",
		summary: "List the change alerts the person you are working for has set up: for " +
			"which changes each fires and whether it is Active or Paused. Only their own " +
			"alerts are returned. Use it to find one to change, pause, resume or delete. An " +
			"alert's name, the table it watches and its conditions show only when your data " +
			"access reaches them, and query searches the name and table, so it is refused " +
			"below that; narrow by status instead.",
		queryNote: "Words to find in the alert's name or the table it watches.",
		notes: map[string]string{
			paramStatus:      "Active or Paused.",
			"conditionMatch": "all when every condition must hold, any when one is enough.",
		},
		unoffered: []string{
			"customTitle", "customMessage", "topic", fieldAlertPriority, fieldAlertRecordID,
			fieldAlertUpdatedAt,
		},
		resource: permission.ResourceTableChangeAlert,
		shape:    alertConditionCount,
	}
}

// alertConditionCount says how many conditions an alert checks rather than
// carrying each one, and only when the conditions themselves were shown: a
// count of withheld conditions would say something about them. An alert
// with none reaches here without the field, which the read leaves out empty.
func alertConditionCount(row map[string]any, withheld map[string]struct{}, _ clock) {
	if _, hidden := withheld[fieldAlertConditions]; hidden {
		return
	}
	conditions, _ := row[fieldAlertConditions].([]any)
	row["conditionCount"] = len(conditions)
	delete(row, fieldAlertConditions)
}
