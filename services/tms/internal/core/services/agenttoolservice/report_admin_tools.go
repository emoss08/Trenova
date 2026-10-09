package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramReportRunID      = "runId"
	paramDefinitionID     = "definitionId"
	paramDashboardID      = "dashboardId"
	paramReportScheduleID = "scheduleId"
	paramFormats          = "formats"
	paramEmailRecipients  = "emailRecipients"
	paramEmailAttach      = "emailAttach"
	paramEmailInline      = "emailInline"
	maxScheduleRecipients = 20

	reportRunSupplier = "The report run, from list_report_runs or the result of run_report. " +
		"Never guess one."
	reportDefinitionSupplier = "The saved report, from list_reports. Never guess one."
	reportDashboardSupplier  = "The dashboard, from list_dashboards. Never guess one."
	reportScheduleSupplier   = "The schedule, from list_report_schedules. Never guess one."
)

type reportAdministrator interface {
	PlanCancelRun(ctx context.Context, req *reporting.GetRunRequest) (*reporting.RunChange, error)
	CancelRun(ctx context.Context, req *reporting.GetRunRequest) (*report.ReportRun, error)
	PlanDeleteDefinition(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
	DeleteDefinition(ctx context.Context, req *reporting.GetDefinitionRequest) error
	PlanResetCannedFork(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*reporting.DefinitionChange, error)
	ResetCannedFork(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
	PlanDeleteDashboard(
		ctx context.Context,
		req *reporting.GetDashboardRequest,
	) (*report.Dashboard, error)
	DeleteDashboard(ctx context.Context, req *reporting.GetDashboardRequest) error
	GetSchedule(
		ctx context.Context,
		req *reporting.GetScheduleRequest,
	) (*report.ReportSchedule, error)
	PlanUpdateSchedule(
		ctx context.Context,
		req *reporting.SaveScheduleRequest,
	) (*reporting.ScheduleChange, error)
	UpdateSchedule(
		ctx context.Context,
		req *reporting.SaveScheduleRequest,
	) (*report.ReportSchedule, error)
	PlanDeleteSchedule(
		ctx context.Context,
		req *reporting.GetScheduleRequest,
	) (*report.ReportSchedule, error)
	DeleteSchedule(ctx context.Context, req *reporting.GetScheduleRequest) error
}

var _ reportAdministrator = (*reporting.Service)(nil)

func reportTarget(key string) func(map[string]any) (serviceports.ToolTarget, bool) {
	return func(params map[string]any) (serviceports.ToolTarget, bool) {
		return targetOf(params, key, permission.ResourceReport)
	}
}

func reportInternalSpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceReport

	return fuelInternalSpec(spec)
}

type reportRunView struct {
	Status string `json:"status"`
}

func newCancelReportRunTool(reports reportAdministrator) serviceports.AgentTool {
	request := func(params *serviceports.ToolExecuteParams) (*reporting.GetRunRequest, error) {
		id, err := requirePulid(params.Params, paramReportRunID)
		if err != nil {
			return nil, err
		}

		return &reporting.GetRunRequest{Request: reportingRequestFrom(*params), RunID: id}, nil
	}

	return newReceivableTool(reportInternalSpec(&receivableSpec{
		name: "cancel_report_run",
		description: "Cancel a report run the person started that is still queued or " +
			"running, such as one asked for with the wrong parameters. A finished run is " +
			"kept as it is.",
		operation:   permission.OpRead,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		rationale: "Stops a report the person asked for from running; nothing is changed or " +
			"sent, and running the report again gives the same rows.",
		properties: map[string]any{
			paramReportRunID: agenttoolschema.KindID(reportRunSupplier, permission.KindReportRun),
		},
		required: []string{paramReportRunID},
		target:   reportTarget(paramReportRunID),
	}), receivablePlan[*reporting.GetRunRequest, *reporting.RunChange]{
		request: request,
		plan: func(
			ctx context.Context,
			req *reporting.GetRunRequest,
			_ *serviceports.ToolExecuteParams,
		) (*reporting.RunChange, error) {
			return reports.PlanCancelRun(ctx, req)
		},
		refused: func(*reporting.GetRunRequest) string {
			return "Would cancel a report run."
		},
		render: func(_ *reporting.GetRunRequest, plan *reporting.RunChange) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{
					Resource: permission.ResourceReport,
					ID:       plan.Before.ID,
					Label:    "Report run",
					Version:  pinnedVersion(plan.Before.Version),
				},
				&reportRunView{Status: string(plan.Before.Status)},
				&reportRunView{Status: string(plan.After.Status)},
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would cancel the report run that is %s.", plan.Before.Status,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *reporting.GetRunRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := reports.CancelRun(ctx, req)

			return nil, err
		},
	})
}

func definitionRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*reporting.GetDefinitionRequest, error) {
	id, err := requirePulid(params.Params, paramDefinitionID)
	if err != nil {
		return nil, err
	}

	return &reporting.GetDefinitionRequest{
		Request:      reportingRequestFrom(*params),
		DefinitionID: id,
	}, nil
}

func newDeleteReportTool(reports reportAdministrator) serviceports.AgentTool {
	return newReceivableTool(reportInternalSpec(&receivableSpec{
		name: "delete_report",
		description: "Propose deleting a saved report the person owns. Its schedules stop " +
			"with it. Built-in reports cannot be deleted; a copy of one made with fork_report " +
			"can.",
		operation: permission.OpDelete,
		maxTier:   agent.TierPropose,
		rationale: "Removes a saved report colleagues may run and nothing brings it back, so " +
			"a person always decides.",
		properties: map[string]any{
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				reportDefinitionSupplier,
			),
		},
		required: []string{paramDefinitionID},
		target:   reportTarget(paramDefinitionID),
	}), receivablePlan[*reporting.GetDefinitionRequest, *report.ReportDefinition]{
		request: definitionRequestFrom,
		plan: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*report.ReportDefinition, error) {
			return reports.PlanDeleteDefinition(ctx, req)
		},
		refused: func(*reporting.GetDefinitionRequest) string {
			return "Would delete a saved report."
		},
		render: func(
			_ *reporting.GetDefinitionRequest,
			existing *report.ReportDefinition,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(
				reportRecord(existing),
				reportViewOf(existing),
				toolpreview.Labels(reportViewLabels),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the saved report %q, which shows %s.",
				existing.Name, reportAudience(existing.Visibility),
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, reports.DeleteDefinition(ctx, req)
		},
	})
}

func newResetReportForkTool(reports reportAdministrator) serviceports.AgentTool {
	return newReceivableTool(reportInternalSpec(&receivableSpec{
		name: "reset_report_fork",
		description: "Put a report copied from a built-in one with fork_report back to the " +
			"built-in's current definition, dropping the person's changes. Its columns, " +
			"filters and parameters become the built-in's again, as when the built-in improved.",
		operation: permission.OpUpdate,
		rationale: "Rewrites a report the person owns from the built-in catalog; nothing is " +
			"sent, but the person's changes to it are lost.",
		properties: map[string]any{
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				reportDefinitionSupplier+" It must be a "+
					"copy of a built-in report.",
			),
		},
		required: []string{paramDefinitionID},
		target:   reportTarget(paramDefinitionID),
	}), receivablePlan[*reporting.GetDefinitionRequest, *reporting.DefinitionChange]{
		request: definitionRequestFrom,
		plan: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*reporting.DefinitionChange, error) {
			return reports.PlanResetCannedFork(ctx, req)
		},
		refused: func(*reporting.GetDefinitionRequest) string {
			return "Would reset a report to its built-in definition."
		},
		render: func(
			_ *reporting.GetDefinitionRequest,
			plan *reporting.DefinitionChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				reportRecord(plan.Before),
				reportViewOf(plan.Before),
				reportViewOf(plan.After),
				toolpreview.Labels(reportViewLabels),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would put the report %q back to the built-in report it was copied from, "+
					"dropping the changes made to it.",
				plan.Before.Name,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := reports.ResetCannedFork(ctx, req)

			return nil, err
		},
	})
}

func newDeleteDashboardTool(reports reportAdministrator) serviceports.AgentTool {
	return newReceivableTool(reportInternalSpec(&receivableSpec{
		name:        "delete_dashboard",
		searchTerms: []string{"remove dashboard", "delete dashboard and its tiles"},
		description: "Propose deleting a dashboard the person owns. The reports its tiles " +
			"show are kept.",
		operation: permission.OpDelete,
		maxTier:   agent.TierPropose,
		rationale: "Removes a dashboard colleagues may open and nothing brings it back, so a " +
			"person always decides.",
		properties: map[string]any{
			paramDashboardID: agenttoolschema.RecordIDText(
				permission.ResourceDashboard,
				reportDashboardSupplier,
			),
		},
		required: []string{paramDashboardID},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramDashboardID, permission.ResourceDashboard)
		},
	}), receivablePlan[*reporting.GetDashboardRequest, *report.Dashboard]{
		request: func(params *serviceports.ToolExecuteParams) (*reporting.GetDashboardRequest, error) {
			id, err := requirePulid(params.Params, paramDashboardID)
			if err != nil {
				return nil, err
			}

			return &reporting.GetDashboardRequest{
				Request:     reportingRequestFrom(*params),
				DashboardID: id,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *reporting.GetDashboardRequest,
			_ *serviceports.ToolExecuteParams,
		) (*report.Dashboard, error) {
			return reports.PlanDeleteDashboard(ctx, req)
		},
		refused: func(*reporting.GetDashboardRequest) string {
			return "Would delete a dashboard."
		},
		render: func(_ *reporting.GetDashboardRequest, existing *report.Dashboard) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(
				toolpreview.Record{
					Resource: permission.ResourceDashboard,
					ID:       existing.ID,
					Label:    existing.Name,
					Version:  pinnedVersion(existing.Version),
				},
				dashboardViewOf(existing),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the dashboard %q; the reports it shows are kept.", existing.Name,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *reporting.GetDashboardRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, reports.DeleteDashboard(ctx, req)
		},
	})
}

type reportScheduleView struct {
	DefinitionID    string   `json:"definitionId"`
	CronExpression  string   `json:"cronExpression"`
	Timezone        string   `json:"timezone"`
	Formats         []string `json:"formats"`
	EmailRecipients []string `json:"emailRecipients"`
	EmailAttach     bool     `json:"emailAttach"`
	EmailInline     bool     `json:"emailInline"`
	Enabled         bool     `json:"enabled"`
}

func reportScheduleViewOf(schedule *report.ReportSchedule) *reportScheduleView {
	view := &reportScheduleView{
		DefinitionID:   schedule.DefinitionID.String(),
		CronExpression: schedule.CronExpression,
		Timezone:       schedule.Timezone,
		Formats:        schedule.Formats,
		Enabled:        schedule.Enabled,
	}
	if schedule.Delivery != nil {
		view.EmailRecipients = schedule.Delivery.EmailRecipients
		view.EmailAttach = schedule.Delivery.EmailAttach
		view.EmailInline = schedule.Delivery.EmailInline
	}

	return view
}

func reportScheduleRecord(schedule *report.ReportSchedule) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceReport,
		ID:       schedule.ID,
		Label:    "Report schedule " + schedule.CronExpression,
		Version:  pinnedVersion(schedule.Version),
	}
}

type scheduleEdit struct {
	id     pulid.ID
	values map[string]any
}

func (e scheduleEdit) request(
	ctx context.Context,
	reports reportAdministrator,
	params *serviceports.ToolExecuteParams,
) (*reporting.SaveScheduleRequest, error) {
	base := reporting.Request{TenantInfo: tenantFrom(*params)}
	stored, err := reports.GetSchedule(ctx, &reporting.GetScheduleRequest{
		Request:    base,
		ScheduleID: e.id,
	})
	if err != nil {
		return nil, err
	}

	req := &reporting.SaveScheduleRequest{
		Request:        base,
		ScheduleID:     stored.ID,
		DefinitionID:   stored.DefinitionID,
		CronExpression: stored.CronExpression,
		Timezone:       stored.Timezone,
		Formats:        slices.Clone(stored.Formats),
		Alert:          stored.Alert,
		Enabled:        stored.Enabled,
		Version:        stored.Version,
	}
	if stored.Delivery != nil {
		req.EmailRecipients = slices.Clone(stored.Delivery.EmailRecipients)
		req.EmailAttach = stored.Delivery.EmailAttach
		req.EmailInline = stored.Delivery.EmailInline
		req.NotifyUserIDs = slices.Clone(stored.Delivery.NotifyUserIDs)
	}

	return req, e.apply(req)
}

func (e scheduleEdit) apply(req *reporting.SaveScheduleRequest) error {
	if _, given := e.values[paramDefinitionID]; given {
		id, err := requirePulid(e.values, paramDefinitionID)
		if err != nil {
			return err
		}
		req.DefinitionID = id
	}
	for key, target := range map[string]*string{
		paramCronExpression: &req.CronExpression,
		paramTimezone:       &req.Timezone,
	} {
		if _, given := e.values[key]; given {
			text, err := requireString(e.values, key)
			if err != nil {
				return err
			}
			*target = strings.TrimSpace(text)
		}
	}
	if _, given := e.values[paramFormats]; given {
		req.Formats = scheduleFormats(e.values)
		for idx, format := range req.Formats {
			if !report.Format(format).IsValid() {
				return fmt.Errorf("%s[%d]: %q is not a report format", paramFormats, idx, format)
			}
		}
	}
	if _, given := e.values[paramEmailRecipients]; given {
		recipients := stringSliceParam(e.values, paramEmailRecipients)
		if len(recipients) > maxScheduleRecipients {
			return fmt.Errorf("%s holds %d addresses; a schedule takes at most %d",
				paramEmailRecipients, len(recipients), maxScheduleRecipients)
		}
		req.EmailRecipients = recipients
	}
	for key, target := range map[string]*bool{
		paramEmailAttach: &req.EmailAttach,
		paramEmailInline: &req.EmailInline,
		paramEnabled:     &req.Enabled,
	} {
		flag, err := optionalBoolPointer(e.values, key)
		if err != nil {
			return err
		}
		if flag != nil {
			*target = *flag
		}
	}

	return nil
}

func scheduleIDFrom(params *serviceports.ToolExecuteParams) (scheduleEdit, error) {
	id, err := requirePulid(params.Params, paramReportScheduleID)
	if err != nil {
		return scheduleEdit{}, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != paramReportScheduleID {
			values[key] = value
		}
	}

	return scheduleEdit{id: id, values: values}, nil
}

func newUpdateReportScheduleTool(reports reportAdministrator) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "update_report_schedule",
		description: "Change a report schedule the person set up: when it runs, its zone, " +
			"the formats, who it is emailed to, or switch it on or off. Fields left out keep " +
			"their value. The report must exist; list_reports finds its id.",
		resource:    permission.ResourceReport,
		operation:   permission.OpExport,
		egress:      agent.EgressExternalRecipient,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes where and when a report is emailed, which may be to addresses " +
			"outside the organization.",
		properties: map[string]any{
			paramReportScheduleID: agenttoolschema.KindID(reportScheduleSupplier,
				permission.KindReportSchedule),
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				"The saved report it runs, from list_reports. "+
					"Leave it out to keep it.",
			),
			paramCronExpression: stringProperty("When it runs, as five cron fields, such as "+
				"\"0 7 * * 1\" for 07:00 every Monday.", 0),
			paramTimezone: stringProperty("The zone the schedule is read in, as an IANA name "+
				"such as America/New_York.", 0),
			paramFormats: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Which formats to attach.",
				toolschema.KeyItems:       agenttoolschema.Enum("", reportFormats),
			},
			paramEmailRecipients: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The email addresses it goes to, replacing the list.",
				toolschema.KeyMaxItems:    maxScheduleRecipients,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
			},
			paramEmailAttach: booleanProperty("Attach the file rather than only linking to it."),
			paramEmailInline: booleanProperty("Put the rows in the email body."),
			paramEnabled:     booleanProperty("Whether the schedule runs."),
		},
		required:    []string{paramReportScheduleID},
		target:      reportTarget(paramReportScheduleID),
		searchTerms: []string{"scheduled", "recipients", "emailed", "subscription"},
	}, receivablePlan[scheduleEdit, *reporting.ScheduleChange]{
		request: scheduleIDFrom,
		plan: func(
			ctx context.Context,
			edit scheduleEdit,
			params *serviceports.ToolExecuteParams,
		) (*reporting.ScheduleChange, error) {
			req, err := edit.request(ctx, reports, params)
			if err != nil {
				return nil, err
			}

			return reports.PlanUpdateSchedule(ctx, req)
		},
		refused: func(scheduleEdit) string {
			return "Would change a report schedule."
		},
		render: func(_ scheduleEdit, plan *reporting.ScheduleChange) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				reportScheduleRecord(plan.Before),
				reportScheduleViewOf(plan.Before),
				reportScheduleViewOf(plan.After),
			)
			if err != nil {
				return nil, err
			}

			recipients := 0
			if plan.After.Delivery != nil {
				recipients = len(plan.After.Delivery.EmailRecipients)
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would change the schedule of the report %q to run %s (%s) and email it to %s.",
				plan.Definition.Name, plan.After.CronExpression, plan.After.Timezone,
				countOf(recipients, "recipient"),
			), change), nil
		},
		run: func(
			ctx context.Context,
			edit scheduleEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := edit.request(ctx, reports, params)
			if err != nil {
				return nil, err
			}
			_, err = reports.UpdateSchedule(ctx, req)

			return nil, err
		},
	})
}

func newDeleteReportScheduleTool(reports reportAdministrator) serviceports.AgentTool {
	return newReceivableTool(reportInternalSpec(&receivableSpec{
		name: "delete_report_schedule",
		description: "Propose removing a report schedule the person set up, so the report " +
			"stops being emailed. The report itself is kept.",
		operation: permission.OpExport,
		maxTier:   agent.TierPropose,
		rationale: "Stops a report reaching the people it was scheduled for; nothing brings " +
			"the schedule back but setting it up again, so a person always decides.",
		properties: map[string]any{
			paramReportScheduleID: agenttoolschema.KindID(reportScheduleSupplier,
				permission.KindReportSchedule),
		},
		required:    []string{paramReportScheduleID},
		target:      reportTarget(paramReportScheduleID),
		searchTerms: []string{"scheduled", "unsubscribe", "emailing"},
	}), receivablePlan[*reporting.GetScheduleRequest, *report.ReportSchedule]{
		request: func(params *serviceports.ToolExecuteParams) (*reporting.GetScheduleRequest, error) {
			id, err := requirePulid(params.Params, paramReportScheduleID)
			if err != nil {
				return nil, err
			}

			return &reporting.GetScheduleRequest{
				Request:    reporting.Request{TenantInfo: tenantFrom(*params)},
				ScheduleID: id,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *reporting.GetScheduleRequest,
			_ *serviceports.ToolExecuteParams,
		) (*report.ReportSchedule, error) {
			return reports.PlanDeleteSchedule(ctx, req)
		},
		refused: func(*reporting.GetScheduleRequest) string {
			return "Would remove a report schedule."
		},
		render: func(
			_ *reporting.GetScheduleRequest,
			existing *report.ReportSchedule,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(
				reportScheduleRecord(existing),
				reportScheduleViewOf(existing),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would remove the report schedule that runs %s; the report is kept.",
				existing.CronExpression,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *reporting.GetScheduleRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, reports.DeleteSchedule(ctx, req)
		},
	})
}
