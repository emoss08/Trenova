package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type previewingSchedules struct {
	scheduleWriter

	previewed *reporting.SaveScheduleRequest
	created   *reporting.SaveScheduleRequest
}

func (f *previewingSchedules) PreviewSchedule(
	_ context.Context,
	req *reporting.SaveScheduleRequest,
) (*reporting.SchedulePreview, error) {
	f.previewed = req

	return &reporting.SchedulePreview{
		Schedule: &report.ReportSchedule{
			DefinitionID:   req.DefinitionID,
			CronExpression: req.CronExpression,
			Timezone:       "America/Chicago",
			Formats:        req.Formats,
			Enabled:        req.Enabled,
			NextRunAt:      1_790_000_000,
			Delivery: &report.ScheduleDelivery{
				EmailRecipients: req.EmailRecipients,
				EmailAttach:     req.EmailAttach,
			},
		},
		Definition: &report.ReportDefinition{ID: req.DefinitionID, Name: "Unbilled aging"},
	}, nil
}

func (f *previewingSchedules) CreateSchedule(
	_ context.Context,
	req *reporting.SaveScheduleRequest,
) (*report.ReportSchedule, error) {
	f.created = req

	return &report.ReportSchedule{}, nil
}

func TestScheduleReportPreview_ShowsTheRecurringEmail(t *testing.T) {
	t.Parallel()

	schedules := &previewingSchedules{}
	tool := &scheduleReportTool{schedules: schedules}
	params := executeParams(map[string]any{
		"definitionId":    pulid.MustNew("rdef_").String(),
		"cronExpression":  "0 7 * * 1",
		"emailRecipients": []any{"ops@carrier.example"},
		"formats":         []any{"csv"},
	})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, schedules.created, "a preview must not schedule")
	assert.Contains(t, preview.Summary, "Unbilled aging")

	created := findChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, "0 7 * * 1", findField(t, created, "cronExpression").After)

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, []string{"ops@carrier.example"}, send.Message.To)
	assert.Equal(t, `"0 7 * * 1" (America/Chicago)`, send.Message.Cadence)
	assert.Equal(t, "The latest Unbilled aging report, attached as CSV.", send.Message.Body)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *schedules.previewed, *schedules.created)
}

// The schedule form in the app starts every schedule on Excel, and a schedule
// with no format is refused. The agent is told formats are optional, so a call
// that leaves them out gets what a person filling in the form would get,
// rather than a proposal that can only fail when it is approved.
func TestScheduleReport_DefaultsToExcelLikeTheScheduleForm(t *testing.T) {
	t.Parallel()

	schedules := &previewingSchedules{}
	tool := &scheduleReportTool{schedules: schedules}
	params := executeParams(map[string]any{
		"definitionId":    pulid.MustNew("rdef_").String(),
		"cronExpression":  "0 7 * * 1",
		"emailRecipients": []any{"ops@carrier.example"},
	})
	params.IdempotencyKey = "idem-1"

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.created)
	assert.Equal(t, []string{string(report.FormatXLSX)}, schedules.created.Formats)
}

func TestScheduleReport_RefusesAFormatReportsCannotBeWrittenIn(t *testing.T) {
	t.Parallel()

	tool := &scheduleReportTool{}

	require.Error(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "0 7 * * 1",
		"emailRecipients": []any{"ops@example.com"},
		"formats":         []any{"docx"},
	}))
	require.NoError(t, tool.validateArgs(map[string]any{
		"definitionId":    "rdef_1",
		"cronExpression":  "0 7 * * 1",
		"emailRecipients": []any{"ops@example.com"},
		"formats":         []any{"csv", "pdf"},
	}))
}
