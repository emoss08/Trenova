package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReportAdmin struct {
	guard      *writeGuard
	run        *report.ReportRun
	definition *report.ReportDefinition
	dashboard  *report.Dashboard
	schedule   *report.ReportSchedule
	canceled   *reporting.GetRunRequest
	deleted    []pulid.ID
	reset      *reporting.GetDefinitionRequest
	saved      *reporting.SaveScheduleRequest
}

func (f *fakeReportAdmin) PlanCancelRun(
	context.Context,
	*reporting.GetRunRequest,
) (*reporting.RunChange, error) {
	after := *f.run
	after.Status = report.RunStatusCanceled

	return &reporting.RunChange{Before: f.run, After: &after}, nil
}

func (f *fakeReportAdmin) CancelRun(
	_ context.Context,
	req *reporting.GetRunRequest,
) (*report.ReportRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.canceled = req

	return f.run, nil
}

func (f *fakeReportAdmin) PlanDeleteDefinition(
	context.Context,
	*reporting.GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	return f.definition, nil
}

func (f *fakeReportAdmin) DeleteDefinition(
	_ context.Context,
	req *reporting.GetDefinitionRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, req.DefinitionID)

	return nil
}

func (f *fakeReportAdmin) PlanResetCannedFork(
	context.Context,
	*reporting.GetDefinitionRequest,
) (*reporting.DefinitionChange, error) {
	after := *f.definition
	after.Description = "The built-in description"

	return &reporting.DefinitionChange{Before: f.definition, After: &after}, nil
}

func (f *fakeReportAdmin) ResetCannedFork(
	_ context.Context,
	req *reporting.GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reset = req

	return f.definition, nil
}

func (f *fakeReportAdmin) PlanDeleteDashboard(
	context.Context,
	*reporting.GetDashboardRequest,
) (*report.Dashboard, error) {
	return f.dashboard, nil
}

func (f *fakeReportAdmin) DeleteDashboard(
	_ context.Context,
	req *reporting.GetDashboardRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, req.DashboardID)

	return nil
}

func (f *fakeReportAdmin) GetSchedule(
	context.Context,
	*reporting.GetScheduleRequest,
) (*report.ReportSchedule, error) {
	copied := *f.schedule

	return &copied, nil
}

func scheduleFrom(req *reporting.SaveScheduleRequest) *report.ReportSchedule {
	return &report.ReportSchedule{
		ID:             req.ScheduleID,
		DefinitionID:   req.DefinitionID,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Formats:        req.Formats,
		Enabled:        req.Enabled,
		Delivery: &report.ScheduleDelivery{
			EmailRecipients: req.EmailRecipients,
			EmailAttach:     req.EmailAttach,
			EmailInline:     req.EmailInline,
		},
		Version: req.Version,
	}
}

func (f *fakeReportAdmin) PlanUpdateSchedule(
	_ context.Context,
	req *reporting.SaveScheduleRequest,
) (*reporting.ScheduleChange, error) {
	return &reporting.ScheduleChange{
		Before:     f.schedule,
		After:      scheduleFrom(req),
		Definition: f.definition,
	}, nil
}

func (f *fakeReportAdmin) UpdateSchedule(
	_ context.Context,
	req *reporting.SaveScheduleRequest,
) (*report.ReportSchedule, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.saved = req

	return scheduleFrom(req), nil
}

func (f *fakeReportAdmin) PlanDeleteSchedule(
	context.Context,
	*reporting.GetScheduleRequest,
) (*report.ReportSchedule, error) {
	return f.schedule, nil
}

func (f *fakeReportAdmin) DeleteSchedule(
	_ context.Context,
	req *reporting.GetScheduleRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, req.ScheduleID)

	return nil
}

func adminReports() *fakeReportAdmin {
	definition := &report.ReportDefinition{
		ID:          pulid.MustNew("rd_"),
		Name:        "Late deliveries",
		Description: "My late deliveries",
		Visibility:  report.VisibilityShared,
		Version:     3,
	}

	return &fakeReportAdmin{
		guard:      &writeGuard{},
		run:        &report.ReportRun{ID: pulid.MustNew("rrun_"), Status: report.RunStatusRunning},
		definition: definition,
		dashboard:  &report.Dashboard{ID: pulid.MustNew("dash_"), Name: "Ops", Version: 1},
		schedule: &report.ReportSchedule{
			ID:             pulid.MustNew("rsch_"),
			DefinitionID:   definition.ID,
			CronExpression: "0 7 * * 1",
			Timezone:       "America/Chicago",
			Formats:        []string{"csv"},
			Enabled:        true,
			Delivery: &report.ScheduleDelivery{
				EmailRecipients: []string{"ops@example.com"},
				NotifyUserIDs:   []pulid.ID{pulid.MustNew("usr_")},
			},
			Version: 9,
		},
	}
}

func TestCancelReportRun_StopsARunWithoutChangingAnything(t *testing.T) {
	t.Parallel()

	reports := adminReports()
	tool := newCancelReportRunTool(reports)
	params := executeParams(map[string]any{paramReportRunID: reports.run.ID.String()})

	preview := previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would cancel the report run that is running.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, reports.run.ID, reports.canceled.RunID)
	assert.Equal(t, params.Actor.UserID, reports.canceled.Principal.UserID)
	assert.Equal(t, agent.TierAutoExecute, tool.Policy().MaxTier)
}

func TestReportDeletions_AreOnlyEverProposed(t *testing.T) {
	t.Parallel()

	reports := adminReports()
	report := newDeleteReportTool(reports)
	params := executeParams(map[string]any{paramDefinitionID: reports.definition.ID.String()})
	preview := previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return report.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, `Would delete the saved report "Late deliveries", which shows on every `+
		"colleague's Reports page.", preview.Summary)
	require.NoError(t, report.Execute(t.Context(), params))

	dashboard := newDeleteDashboardTool(reports)
	board := executeParams(map[string]any{paramDashboardID: reports.dashboard.ID.String()})
	preview = previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return dashboard.(serviceports.ToolPreviewer).Preview(t.Context(), board)
	})
	assert.Equal(t, permission.ResourceDashboard, previewChange(t, preview, 0).Resource)
	require.NoError(t, dashboard.Execute(t.Context(), board))

	schedule := newDeleteReportScheduleTool(reports)
	stop := executeParams(map[string]any{paramReportScheduleID: reports.schedule.ID.String()})
	preview = previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return schedule.(serviceports.ToolPreviewer).Preview(t.Context(), stop)
	})
	assert.Contains(t, preview.Summary, "runs 0 7 * * 1; the report is kept")
	require.NoError(t, schedule.Execute(t.Context(), stop))

	assert.Equal(t, []pulid.ID{
		reports.definition.ID,
		reports.dashboard.ID,
		reports.schedule.ID,
	}, reports.deleted)
	for _, tool := range []serviceports.AgentTool{report, dashboard, schedule} {
		assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier, tool.Name())
	}
}

func TestResetReportFork_ShowsWhatTheBuiltInPutsBack(t *testing.T) {
	t.Parallel()

	reports := adminReports()
	tool := newResetReportForkTool(reports)
	params := executeParams(map[string]any{paramDefinitionID: reports.definition.ID.String()})

	preview := previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "dropping the changes made to it")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, reports.definition.ID, reports.reset.DefinitionID)
}

func TestUpdateReportSchedule_KeepsWhatItIsNotGivenAndNeverLeavesUnasked(t *testing.T) {
	t.Parallel()

	reports := adminReports()
	tool := newUpdateReportScheduleTool(reports)
	params := executeParams(map[string]any{
		paramReportScheduleID: reports.schedule.ID.String(),
		paramEmailRecipients:  []any{"ops@example.com", "controller@customer.example"},
		paramFormats:          []any{"xlsx"},
	})

	preview := previewWithoutWrites(t, reports.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, `Would change the schedule of the report "Late deliveries" to run `+
		"0 7 * * 1 (America/Chicago) and email it to 2 recipients.", preview.Summary)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "0 7 * * 1", reports.saved.CronExpression)
	assert.Equal(t, []string{"xlsx"}, reports.saved.Formats)
	assert.Len(t, reports.saved.EmailRecipients, 2)
	assert.Equal(t, reports.schedule.Delivery.NotifyUserIDs, reports.saved.NotifyUserIDs)
	assert.Equal(t, reports.schedule.Version, reports.saved.Version)
	assert.True(t, reports.saved.Enabled)

	policy := tool.Policy()
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, permission.OpExport, policy.Operation)

	params.Params[paramFormats] = []any{"docx"}
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}
