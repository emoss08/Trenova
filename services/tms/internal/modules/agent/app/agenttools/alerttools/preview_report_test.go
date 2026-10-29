package alerttools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportscheduleservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type previewingSchedules struct {
	scheduleWriter

	previewed *reportscheduleservice.SaveScheduleRequest
	created   *reportscheduleservice.SaveScheduleRequest
}

func (f *previewingSchedules) PreviewSchedule(
	_ context.Context,
	req *reportscheduleservice.SaveScheduleRequest,
) (*reportscheduleservice.SchedulePreview, error) {
	f.previewed = req

	return &reportscheduleservice.SchedulePreview{
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
	req *reportscheduleservice.SaveScheduleRequest,
) (*report.ReportSchedule, error) {
	f.created = req

	return &report.ReportSchedule{}, nil
}

func TestScheduleReportPreview_ShowsTheRecurringEmail(t *testing.T) {
	t.Parallel()

	schedules := &previewingSchedules{}
	tool := &ScheduleReportTool{schedules: schedules}
	params := agenttoolstest.ExecuteParams(map[string]any{
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

	created := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationCreate)
	assert.Equal(t, "0 7 * * 1", agenttoolstest.FindField(t, created, "cronExpression").After)

	send := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationSend)
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
	tool := &ScheduleReportTool{schedules: schedules}
	params := agenttoolstest.ExecuteParams(map[string]any{
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

	tool := &ScheduleReportTool{}

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
