package reporttools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dashboardtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/recurringshipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportscheduleservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramReportRunID = "runId"

const paramDefinitionID = "definitionId"

const paramDashboardID = "dashboardId"

const paramReportScheduleID = "scheduleId"

const paramFormats = "formats"

const paramEmailRecipients = "emailRecipients"

const paramEmailAttach = "emailAttach"

const paramEmailInline = "emailInline"

const maxScheduleRecipients = 20

const reportRunSupplier = "The report run, from report_list_runs or the result of report_run. " +
	"Never guess one."

const reportDefinitionSupplier = "The saved report, from report_list. Never guess one."

const reportDashboardSupplier = "The dashboard, from report_list_dashboards. Never guess one."

const reportScheduleSupplier = "The schedule, from report_list_schedules. Never guess one."

type ReportAdministrator interface {
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
}

type DashboardAdministrator interface {
	PlanDeleteDashboard(
		ctx context.Context,
		req *reportdashboardservice.GetDashboardRequest,
	) (*report.Dashboard, error)
	DeleteDashboard(ctx context.Context, req *reportdashboardservice.GetDashboardRequest) error
}

type ScheduleAdministrator interface {
	GetSchedule(
		ctx context.Context,
		req *reportscheduleservice.GetScheduleRequest,
	) (*report.ReportSchedule, error)
	PlanUpdateSchedule(
		ctx context.Context,
		req *reportscheduleservice.SaveScheduleRequest,
	) (*reportscheduleservice.ScheduleChange, error)
	UpdateSchedule(
		ctx context.Context,
		req *reportscheduleservice.SaveScheduleRequest,
	) (*report.ReportSchedule, error)
	PlanDeleteSchedule(
		ctx context.Context,
		req *reportscheduleservice.GetScheduleRequest,
	) (*report.ReportSchedule, error)
	DeleteSchedule(ctx context.Context, req *reportscheduleservice.GetScheduleRequest) error
}

func reportTarget(key string) func(map[string]any) (agenttoolports.ToolTarget, bool) {
	return func(params map[string]any) (agenttoolports.ToolTarget, bool) {
		return toolparams.TargetOf(params, key, permission.ResourceReport)
	}
}

func reportInternalSpec(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.Resource = permission.ResourceReport

	return toolspec.FuelInternalSpec(spec)
}

type reportRunView struct {
	Status string `json:"status"`
}

func NewCancelReportRunTool(reports ReportAdministrator) agenttoolports.AgentTool {
	request := func(params *agenttoolports.ToolExecuteParams) (*reporting.GetRunRequest, error) {
		id, err := toolparams.RequirePulid(params.Params, paramReportRunID)
		if err != nil {
			return nil, err
		}

		return &reporting.GetRunRequest{Request: reportingRequestFrom(*params), RunID: id}, nil
	}

	return toolspec.NewReceivableTool(reportInternalSpec(&toolspec.ReceivableSpec{
		Name: "report_cancel_run",
		Description: "Cancel a report run the person started that is still queued or " +
			"running, such as one asked for with the wrong parameters. A finished run is " +
			"kept as it is.",
		Operation:   permission.OpRead,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Rationale: "Stops a report the person asked for from running; nothing is changed or " +
			"sent, and running the report again gives the same rows.",
		Properties: map[string]any{
			paramReportRunID: agenttoolschema.KindID(reportRunSupplier, recordkind.ReportRun),
		},
		Required: []string{paramReportRunID},
		Target:   reportTarget(paramReportRunID),
	}), toolspec.ReceivablePlan[*reporting.GetRunRequest, *reporting.RunChange]{
		Request: request,
		Plan: func(
			ctx context.Context,
			req *reporting.GetRunRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*reporting.RunChange, error) {
			return reports.PlanCancelRun(ctx, req)
		},
		Refused: func(*reporting.GetRunRequest) string {
			return "Would cancel a report run."
		},
		Render: func(_ *reporting.GetRunRequest, plan *reporting.RunChange) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{
					Resource: permission.ResourceReport,
					ID:       plan.Before.ID,
					Label:    "Report run",
					Version:  changepreview.PinnedVersion(plan.Before.Version),
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
		Run: func(
			ctx context.Context,
			req *reporting.GetRunRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := reports.CancelRun(ctx, req)

			return nil, err
		},
	})
}

func definitionRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*reporting.GetDefinitionRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramDefinitionID)
	if err != nil {
		return nil, err
	}

	return &reporting.GetDefinitionRequest{
		Request:      reportingRequestFrom(*params),
		DefinitionID: id,
	}, nil
}

func NewDeleteReportTool(reports ReportAdministrator) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(reportInternalSpec(&toolspec.ReceivableSpec{
		Name: "report_delete",
		Description: "Propose deleting a saved report the person owns. Its schedules stop " +
			"with it. Built-in reports cannot be deleted; a copy of one made with report_fork " +
			"can.",
		Operation: permission.OpDelete,
		MaxTier:   agent.TierPropose,
		Rationale: "Removes a saved report colleagues may run and nothing brings it back, so " +
			"a person always decides.",
		Properties: map[string]any{
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				reportDefinitionSupplier,
			),
		},
		Required: []string{paramDefinitionID},
		Target:   reportTarget(paramDefinitionID),
	}), toolspec.ReceivablePlan[*reporting.GetDefinitionRequest, *report.ReportDefinition]{
		Request: definitionRequestFrom,
		Plan: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*report.ReportDefinition, error) {
			return reports.PlanDeleteDefinition(ctx, req)
		},
		Refused: func(*reporting.GetDefinitionRequest) string {
			return "Would delete a saved report."
		},
		Render: func(
			_ *reporting.GetDefinitionRequest,
			existing *report.ReportDefinition,
		) (*agentpreview.ToolPreview, error) {
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
		Run: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			return nil, reports.DeleteDefinition(ctx, req)
		},
	})
}

func NewResetReportForkTool(reports ReportAdministrator) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(reportInternalSpec(&toolspec.ReceivableSpec{
		Name: "report_reset_fork",
		Description: "Put a report copied from a built-in one with report_fork back to the " +
			"built-in's current definition, dropping the person's changes. Its columns, " +
			"filters and parameters become the built-in's again, as when the built-in improved.",
		Operation: permission.OpUpdate,
		Rationale: "Rewrites a report the person owns from the built-in catalog; nothing is " +
			"sent, but the person's changes to it are lost.",
		Properties: map[string]any{
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				reportDefinitionSupplier+" It must be a "+
					"copy of a built-in report.",
			),
		},
		Required: []string{paramDefinitionID},
		Target:   reportTarget(paramDefinitionID),
	}), toolspec.ReceivablePlan[*reporting.GetDefinitionRequest, *reporting.DefinitionChange]{
		Request: definitionRequestFrom,
		Plan: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*reporting.DefinitionChange, error) {
			return reports.PlanResetCannedFork(ctx, req)
		},
		Refused: func(*reporting.GetDefinitionRequest) string {
			return "Would reset a report to its built-in definition."
		},
		Render: func(
			_ *reporting.GetDefinitionRequest,
			plan *reporting.DefinitionChange,
		) (*agentpreview.ToolPreview, error) {
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
		Run: func(
			ctx context.Context,
			req *reporting.GetDefinitionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := reports.ResetCannedFork(ctx, req)

			return nil, err
		},
	})
}

func NewDeleteDashboardTool(dashboards DashboardAdministrator) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(reportInternalSpec(&toolspec.ReceivableSpec{
		Name:        "report_delete_dashboard",
		SearchTerms: []string{"remove dashboard", "delete dashboard and its tiles"},
		Description: "Propose deleting a dashboard the person owns. The reports its tiles " +
			"show are kept.",
		Operation: permission.OpDelete,
		MaxTier:   agent.TierPropose,
		Rationale: "Removes a dashboard colleagues may open and nothing brings it back, so a " +
			"person always decides.",
		Properties: map[string]any{
			paramDashboardID: agenttoolschema.RecordIDText(
				permission.ResourceDashboard,
				reportDashboardSupplier,
			),
		},
		Required: []string{paramDashboardID},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramDashboardID, permission.ResourceDashboard)
		},
	}), toolspec.ReceivablePlan[*reportdashboardservice.GetDashboardRequest, *report.Dashboard]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*reportdashboardservice.GetDashboardRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramDashboardID)
			if err != nil {
				return nil, err
			}

			return &reportdashboardservice.GetDashboardRequest{
				Request:     reportingRequestFrom(*params),
				DashboardID: id,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *reportdashboardservice.GetDashboardRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*report.Dashboard, error) {
			return dashboards.PlanDeleteDashboard(ctx, req)
		},
		Refused: func(*reportdashboardservice.GetDashboardRequest) string {
			return "Would delete a dashboard."
		},
		Render: func(_ *reportdashboardservice.GetDashboardRequest, existing *report.Dashboard) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Delete(
				toolpreview.Record{
					Resource: permission.ResourceDashboard,
					ID:       existing.ID,
					Label:    existing.Name,
					Version:  changepreview.PinnedVersion(existing.Version),
				},
				dashboardtools.DashboardViewOf(existing),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the dashboard %q; the reports it shows are kept.", existing.Name,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *reportdashboardservice.GetDashboardRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			return nil, dashboards.DeleteDashboard(ctx, req)
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
		Version:  changepreview.PinnedVersion(schedule.Version),
	}
}

type scheduleEdit struct {
	id     pulid.ID
	values map[string]any
}

func (e scheduleEdit) request(
	ctx context.Context,
	schedules ScheduleAdministrator,
	params *agenttoolports.ToolExecuteParams,
) (*reportscheduleservice.SaveScheduleRequest, error) {
	base := reporting.Request{TenantInfo: toolparams.TenantFrom(*params)}
	stored, err := schedules.GetSchedule(ctx, &reportscheduleservice.GetScheduleRequest{
		Request:    base,
		ScheduleID: e.id,
	})
	if err != nil {
		return nil, err
	}

	req := &reportscheduleservice.SaveScheduleRequest{
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

func (e scheduleEdit) apply(req *reportscheduleservice.SaveScheduleRequest) error {
	if _, given := e.values[paramDefinitionID]; given {
		id, err := toolparams.RequirePulid(e.values, paramDefinitionID)
		if err != nil {
			return err
		}
		req.DefinitionID = id
	}
	for key, target := range map[string]*string{
		recurringshipmenttools.ParamCronExpression: &req.CronExpression,
		recurringshipmenttools.ParamTimezone:       &req.Timezone,
	} {
		if _, given := e.values[key]; given {
			text, err := toolparams.RequireString(e.values, key)
			if err != nil {
				return err
			}
			*target = strings.TrimSpace(text)
		}
	}
	if _, given := e.values[paramFormats]; given {
		req.Formats = toolparams.ScheduleFormats(e.values)
		for idx, format := range req.Formats {
			if !report.Format(format).IsValid() {
				return fmt.Errorf("%s[%d]: %q is not a report format", paramFormats, idx, format)
			}
		}
	}
	if _, given := e.values[paramEmailRecipients]; given {
		recipients := toolparams.StringSliceParam(e.values, paramEmailRecipients)
		if len(recipients) > maxScheduleRecipients {
			return fmt.Errorf("%s holds %d addresses; a schedule takes at most %d",
				paramEmailRecipients, len(recipients), maxScheduleRecipients)
		}
		req.EmailRecipients = recipients
	}
	for key, target := range map[string]*bool{
		paramEmailAttach:          &req.EmailAttach,
		paramEmailInline:          &req.EmailInline,
		carriertools.ParamEnabled: &req.Enabled,
	} {
		flag, err := toolparams.OptionalBoolPointer(e.values, key)
		if err != nil {
			return err
		}
		if flag != nil {
			*target = *flag
		}
	}

	return nil
}

func scheduleIDFrom(params *agenttoolports.ToolExecuteParams) (scheduleEdit, error) {
	id, err := toolparams.RequirePulid(params.Params, paramReportScheduleID)
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

func NewUpdateReportScheduleTool(schedules ScheduleAdministrator) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "report_update_schedule",
		Description: "Change a report schedule the person set up: when it runs, its zone, " +
			"the formats, who it is emailed to, or switch it on or off. Fields left out keep " +
			"their value. The report must exist; report_list finds its id.",
		Resource:    permission.ResourceReport,
		Operation:   permission.OpExport,
		Egress:      agent.EgressExternalRecipient,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes where and when a report is emailed, which may be to addresses " +
			"outside the organization.",
		Properties: map[string]any{
			paramReportScheduleID: agenttoolschema.KindID(reportScheduleSupplier,
				recordkind.ReportSchedule),
			paramDefinitionID: agenttoolschema.RecordIDText(
				permission.ResourceReport,
				"The saved report it runs, from report_list. "+
					"Leave it out to keep it.",
			),
			recurringshipmenttools.ParamCronExpression: toolparams.StringProperty(
				"When it runs, as five cron fields, such as "+
					"\"0 7 * * 1\" for 07:00 every Monday.",
				0,
			),
			recurringshipmenttools.ParamTimezone: toolparams.StringProperty(
				"The zone the schedule is read in, as an IANA name "+
					"such as America/New_York.",
				0,
			),
			paramFormats: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Which formats to attach.",
				toolschema.KeyItems:       agenttoolschema.Enum("", toolparams.ReportFormats),
			},
			paramEmailRecipients: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The email addresses it goes to, replacing the list.",
				toolschema.KeyMaxItems:    maxScheduleRecipients,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
			},
			paramEmailAttach: toolparams.BooleanProperty(
				"Attach the file rather than only linking to it.",
			),
			paramEmailInline: toolparams.BooleanProperty(
				"Put the rows in the email body.",
			),
			carriertools.ParamEnabled: toolparams.BooleanProperty("Whether the schedule runs."),
		},
		Required:    []string{paramReportScheduleID},
		Target:      reportTarget(paramReportScheduleID),
		SearchTerms: []string{"scheduled", "recipients", "emailed", "subscription"},
	}, toolspec.ReceivablePlan[scheduleEdit, *reportscheduleservice.ScheduleChange]{
		Request: scheduleIDFrom,
		Plan: func(
			ctx context.Context,
			edit scheduleEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*reportscheduleservice.ScheduleChange, error) {
			req, err := edit.request(ctx, schedules, params)
			if err != nil {
				return nil, err
			}

			return schedules.PlanUpdateSchedule(ctx, req)
		},
		Refused: func(scheduleEdit) string {
			return "Would change a report schedule."
		},
		Render: renderReportScheduleUpdate,
		Run: func(
			ctx context.Context,
			edit scheduleEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := edit.request(ctx, schedules, params)
			if err != nil {
				return nil, err
			}
			_, err = schedules.UpdateSchedule(ctx, req)

			return nil, err
		},
	})
}

func renderReportScheduleUpdate(
	_ scheduleEdit,
	plan *reportscheduleservice.ScheduleChange,
) (*agentpreview.ToolPreview, error) {
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
		changepreview.CountOf(recipients, "recipient"),
	), change), nil
}

func NewDeleteReportScheduleTool(schedules ScheduleAdministrator) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(reportInternalSpec(&toolspec.ReceivableSpec{
		Name: "report_delete_schedule",
		Description: "Propose removing a report schedule the person set up, so the report " +
			"stops being emailed. The report itself is kept.",
		Operation: permission.OpExport,
		MaxTier:   agent.TierPropose,
		Rationale: "Stops a report reaching the people it was scheduled for; nothing brings " +
			"the schedule back but setting it up again, so a person always decides.",
		Properties: map[string]any{
			paramReportScheduleID: agenttoolschema.KindID(reportScheduleSupplier,
				recordkind.ReportSchedule),
		},
		Required:    []string{paramReportScheduleID},
		Target:      reportTarget(paramReportScheduleID),
		SearchTerms: []string{"scheduled", "unsubscribe", "emailing"},
	}), toolspec.ReceivablePlan[*reportscheduleservice.GetScheduleRequest, *report.ReportSchedule]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*reportscheduleservice.GetScheduleRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramReportScheduleID)
			if err != nil {
				return nil, err
			}

			return &reportscheduleservice.GetScheduleRequest{
				Request:    reporting.Request{TenantInfo: toolparams.TenantFrom(*params)},
				ScheduleID: id,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *reportscheduleservice.GetScheduleRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*report.ReportSchedule, error) {
			return schedules.PlanDeleteSchedule(ctx, req)
		},
		Refused: func(*reportscheduleservice.GetScheduleRequest) string {
			return "Would remove a report schedule."
		},
		Render: func(
			_ *reportscheduleservice.GetScheduleRequest,
			existing *report.ReportSchedule,
		) (*agentpreview.ToolPreview, error) {
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
		Run: func(
			ctx context.Context,
			req *reportscheduleservice.GetScheduleRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			return nil, schedules.DeleteSchedule(ctx, req)
		},
	})
}
