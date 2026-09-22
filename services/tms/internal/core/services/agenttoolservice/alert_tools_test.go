package agenttoolservice

import (
	"testing"

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

	require.True(t, (&scheduleReportTool{}).RequiresIdempotencyKey())
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
