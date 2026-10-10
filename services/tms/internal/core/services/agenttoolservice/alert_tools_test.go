package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A trigger nobody parsed is a schedule that silently never fires.

The failure is the worst shape available: the person asks for the unbilled
aging every Monday, the card says it was scheduled, and weeks later they
mention they never got it. Nothing errored, nothing logged, and the report
has been silent the whole time. So the expression is parsed when the card is
built, by the same function the service uses when it runs.
*/
func TestScheduleReport_RefusesAnExpressionThatWillNeverFire(t *testing.T) {
	t.Parallel()

	tool := &scheduleReportTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "every monday morning",
		"emailRecipients": []any{"ops@example.com"},
	}))

	require.Error(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "99 99 * * *",
		"emailRecipients": []any{"ops@example.com"},
	}))
}

func TestScheduleReport_AcceptsARealSchedule(t *testing.T) {
	t.Parallel()

	tool := &scheduleReportTool{}

	require.NoError(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "0 7 * * 1",
		"timezone":        "America/New_York",
		"emailRecipients": []any{"ops@example.com"},
	}))
}

// A schedule with nowhere to send runs forever and reaches nobody.
func TestScheduleReport_RefusesASendWithNoRecipients(t *testing.T) {
	t.Parallel()

	tool := &scheduleReportTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"definitionId":   "rdef_1",
		"cronExpression": "0 7 * * 1",
	}))
	require.Error(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "0 7 * * 1",
		"emailRecipients": []any{"", "   "},
	}))
}

// It sends, so a retry that creates a second schedule mails the same report
// twice a week until somebody goes looking for the duplicate.
func TestScheduleReport_NeedsAnIdempotencyKey(t *testing.T) {
	t.Parallel()

	require.True(t, (&scheduleReportTool{}).Policy().Idempotent)
}

/*
An alert is only as good as its trigger, and both halves can be wrong in ways
that read as working: an event type nothing emits, or an operator the matcher
does not know. Either produces an alert that sits there and never fires.
*/
func TestCreateTableChangeAlert_RefusesAnEventNothingEmits(t *testing.T) {
	t.Parallel()

	tool := &createTableChangeAlertTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"name":       "Delayed shipments",
		"tableName":  "shipments",
		"eventTypes": []any{"MODIFIED"},
	}))
}

func TestCreateTableChangeAlert_RefusesAnOperatorTheMatcherDoesNotKnow(t *testing.T) {
	t.Parallel()

	tool := &createTableChangeAlertTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"name":       "Delayed shipments",
		"tableName":  "shipments",
		"eventTypes": []any{"UPDATE"},
		"conditions": []any{
			map[string]any{"field": "status", "operator": "becomes", "value": "Delayed"},
		},
	}))
}

func TestCreateTableChangeAlert_RefusesAConditionOnNoColumn(t *testing.T) {
	t.Parallel()

	tool := &createTableChangeAlertTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"name":       "Delayed shipments",
		"tableName":  "shipments",
		"eventTypes": []any{"UPDATE"},
		"conditions": []any{map[string]any{"operator": "changed_to", "value": "Delayed"}},
	}))
}

func TestCreateTableChangeAlert_AcceptsAnAlertThatCanFire(t *testing.T) {
	t.Parallel()

	tool := &createTableChangeAlertTool{}

	require.NoError(t, tool.validateArgs(map[string]any{
		"name":           "Delayed shipments",
		"tableName":      "shipments",
		"eventTypes":     []any{"UPDATE"},
		"watchedColumns": []any{"status"},
		"conditions": []any{
			map[string]any{"field": "status", "operator": "changed_to", "value": "Delayed"},
		},
	}))
}

func TestCreateTableChangeAlert_NeedsSomethingToWatch(t *testing.T) {
	t.Parallel()

	tool := &createTableChangeAlertTool{}

	require.Error(t, tool.validateArgs(map[string]any{"name": "x", "tableName": "shipments"}))
	require.Error(t, tool.validateArgs(map[string]any{"name": "x", "eventTypes": []any{"UPDATE"}}))
	require.Error(t, tool.validateArgs(map[string]any{
		"tableName": "shipments", "eventTypes": []any{"UPDATE"},
	}))
}

type fakeScheduleReports struct {
	scheduleWriter
	known    map[pulid.ID]bool
	existing []*report.ReportSchedule
}

func (f *fakeScheduleReports) ListSchedules(
	_ context.Context,
	req *reporting.ListSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	matched := make([]*report.ReportSchedule, 0, len(f.existing))
	for _, schedule := range f.existing {
		if schedule.DefinitionID == req.DefinitionID {
			matched = append(matched, schedule)
		}
	}

	return matched, nil
}

func (f *fakeScheduleReports) GetDefinition(
	_ context.Context,
	req *reporting.GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	if !f.known[req.DefinitionID] {
		return nil, errortypes.NewNotFoundError("Report not found")
	}

	return &report.ReportDefinition{ID: req.DefinitionID}, nil
}

// A schedule on a report that does not exist is refused on the card, naming
// the tool that has the real id, rather than after somebody approves it.
func TestScheduleReport_RefusesAReportThatDoesNotExistBeforeProposing(t *testing.T) {
	t.Parallel()

	known := pulid.MustNew("rdef_")
	tool := &scheduleReportTool{
		schedules: &fakeScheduleReports{known: map[pulid.ID]bool{known: true}},
	}
	args := func(id string) map[string]any {
		return map[string]any{
			"definitionId":    id,
			"cronExpression":  "0 7 * * 1",
			"emailRecipients": []any{"ops@example.com"},
		}
	}

	require.NoError(t, tool.Validate(t.Context(), executeParams(args(known.String()))))

	err := tool.Validate(t.Context(), executeParams(args("unbilled_aging")))
	assert.Contains(t, fieldErrors(t, err)["definitionId"], "list_reports")

	err = tool.Validate(t.Context(), executeParams(args(pulid.MustNew("rdef_").String())))
	assert.Contains(t, fieldErrors(t, err)["definitionId"], "list_reports")
}

// Approving the same schedule twice sent every email twice; three identical
// Monday schedules piled up on one report. A repeat is refused, naming the one
// already there; a different time or recipient is not a repeat.
func TestScheduleReport_RefusesAScheduleTheReportAlreadyHas(t *testing.T) {
	t.Parallel()

	definition := pulid.MustNew("rdef_")
	standing := &report.ReportSchedule{
		ID:             pulid.MustNew("rsch_"),
		DefinitionID:   definition,
		CronExpression: "0 7 * * 1",
		Timezone:       "America/Los_Angeles",
		Delivery:       &report.ScheduleDelivery{EmailRecipients: []string{"Admin@Trenova.app"}},
	}
	tool := &scheduleReportTool{schedules: &fakeScheduleReports{
		known:    map[pulid.ID]bool{definition: true},
		existing: []*report.ReportSchedule{standing},
	}}
	args := func(cron, recipient string) map[string]any {
		return map[string]any{
			"definitionId":    definition.String(),
			"cronExpression":  cron,
			"timezone":        "America/Los_Angeles",
			"emailRecipients": []any{recipient},
		}
	}

	err := tool.Validate(t.Context(), executeParams(args("0 7 * * 1", "admin@trenova.app")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), standing.ID.String())

	require.NoError(t, tool.Validate(t.Context(), executeParams(args("0 7 * * 2", "admin@trenova.app"))))
	require.NoError(t, tool.Validate(t.Context(), executeParams(args("0 7 * * 1", "ops@trenova.app"))))
}
