package payrolltools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/app/timesheetservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPayrollExportID = "payrollExportId"

const paramFirstDay = "firstDay"

const paramLastDay = "lastDay"

const kindPayrollExport = "payroll run"

const maxPayrollNoteChars = 1000

var payrollExportFields = []string{
	changepreview.FieldStatus,
	changepreview.FieldPeriodStart,
	changepreview.FieldPeriodEnd,
	"timesheetCount",
	"regularMinutes",
	"overtimeMinutes",
	"paidLeaveMinutes",
	"generatedById",
	"generatedAt",
	changepreview.FieldVoidedAt,
	"voidReason",
	changepreview.FieldNote,
}

type PayrollExporter interface {
	PlanGenerateExport(
		ctx context.Context,
		req *timesheetservice.GenerateExportRequest,
	) (*timesheetservice.ExportPlan, error)
	GenerateExport(
		ctx context.Context,
		req *timesheetservice.GenerateExportRequest,
	) (*timesheet.PayrollExport, error)
	PlanVoidExport(
		ctx context.Context,
		req *timesheetservice.VoidExportRequest,
	) (*timesheetservice.ExportChange, error)
	VoidExport(
		ctx context.Context,
		req *timesheetservice.VoidExportRequest,
	) (*timesheet.PayrollExport, error)
}

func PayrollToolProviders() []any {
	return []any{provideGeneratePayrollExportTool, provideVoidPayrollExportTool}
}

func payrollExportRecord(export *timesheet.PayrollExport) toolpreview.Record {
	return workforcekit.Record(permission.ResourceTimesheet, export.ID,
		fmt.Sprintf("Payroll run %s to %s", changepreview.DayText(export.PeriodStart),
			changepreview.DayText(export.PeriodEnd-toolparams.SecondsPerDay)), export.Version)
}

func payrollExportResult(
	action string,
	export *timesheet.PayrollExport,
) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   kindPayrollExport,
		IDs:    map[string]string{paramPayrollExportID: export.ID.String()},
	}
}

func payrollSpec(name, description, rationale string) *toolspec.ReceivableSpec {
	spec := workforcekit.PersonOnly(
		workforcekit.Spec(name, description, rationale, permission.ResourceTimesheet,
			permission.OpExport),
	)
	spec.Egress = agent.EgressMoney
	spec.Reversible = false
	spec.Artifact = ""
	spec.SearchTerms = []string{"payroll", "payroll export", "payroll run", "timesheets"}
	return spec
}

func newGeneratePayrollExportTool(exporter PayrollExporter) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(payrollSpec(
		"settlement_generate_payroll_export",
		"Draft a payroll run over a pay period for a person to approve. Every approved "+
			"timesheet week starting between the first and last day given that has not gone "+
			"to payroll yet is totalled and locked into the run. Refused when no approved week "+
			"is waiting.",
		"Locks approved hours into a payroll run that pays people, so a person approves it "+
			"and it runs as them; settlement_void_payroll_export takes it back.",
	), map[string]any{
		paramFirstDay: agenttoolschema.Date("The first day of the pay period."),
		paramLastDay:  agenttoolschema.Date("The last day of the pay period, included."),
		changepreview.FieldNote: toolparams.StringProperty(
			"A note kept on the run.",
			maxPayrollNoteChars,
		),
	}, paramFirstDay, paramLastDay)

	return toolspec.NewReceivableTool(spec, toolspec.ReceivablePlan[
		*timesheetservice.GenerateExportRequest, *timesheetservice.ExportPlan,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*timesheetservice.GenerateExportRequest, error) {
			first, err := toolparams.RequireScheduleDay(params.Params, paramFirstDay)
			if err != nil {
				return nil, err
			}
			last, err := toolparams.RequireScheduleDay(params.Params, paramLastDay)
			if err != nil {
				return nil, err
			}
			if last < first {
				return nil, fmt.Errorf("parameter %q cannot be before %q", paramLastDay,
					paramFirstDay)
			}
			note, err := toolparams.BoundedText(
				params.Params,
				changepreview.FieldNote,
				maxPayrollNoteChars,
			)
			if err != nil {
				return nil, err
			}
			return &timesheetservice.GenerateExportRequest{
				PeriodStart: first,
				PeriodEnd:   last + toolparams.SecondsPerDay,
				Note:        note,
				TenantInfo:  toolparams.TenantFrom(*params),
				UserID:      params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *timesheetservice.GenerateExportRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*timesheetservice.ExportPlan, error) {
			return exporter.PlanGenerateExport(ctx, req)
		},
		Refused: func(*timesheetservice.GenerateExportRequest) string {
			return "Would generate a payroll run."
		},
		Render: func(
			_ *timesheetservice.GenerateExportRequest,
			plan *timesheetservice.ExportPlan,
		) (*agentpreview.ToolPreview, error) {
			record := payrollExportRecord(plan.Export)
			record.ID = pulid.Nil
			record.Version = nil
			created, err := toolpreview.Create(record, plan.Export,
				append(workforcekit.Options(payrollExportFields...),
					toolpreview.Volatile("generatedAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would lock %d approved timesheet weeks into a payroll run.",
				len(plan.TimesheetIDs)), created), nil
		},
		Run: func(
			ctx context.Context,
			req *timesheetservice.GenerateExportRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			export, err := exporter.GenerateExport(ctx, req)
			if err != nil {
				return nil, err
			}
			return payrollExportResult("generated", export), nil
		},
	})
}

func newVoidPayrollExportTool(exporter PayrollExporter) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(payrollSpec(
		"settlement_void_payroll_export",
		"Draft voiding a payroll run for a person to approve, with the reason: the run is "+
			"taken back and every week in it returns to approved, ready for another run. "+
			"A run already voided is refused.",
		"Takes back a payroll run that payroll may already have paid from, so a person "+
			"approves it and it runs as them; a voided run stays voided.",
	), map[string]any{
		paramPayrollExportID: agenttoolschema.KindID(
			"The payroll run, from settlement_list_payroll_exports. "+
				"Never guess one.",
			recordkind.PayrollExport,
		),
		changepreview.FieldReason: toolparams.StringProperty(
			"Why the run is voided, kept on it.",
			maxPayrollNoteChars,
		),
	}, paramPayrollExportID, changepreview.FieldReason), paramPayrollExportID, permission.ResourceTimesheet)
	spec.SearchTerms = append(spec.SearchTerms, "undo payroll run", "redo payroll", "rerun payroll")

	return toolspec.NewReceivableTool(spec, toolspec.ReceivablePlan[
		*timesheetservice.VoidExportRequest, *timesheetservice.ExportChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*timesheetservice.VoidExportRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramPayrollExportID)
			if err != nil {
				return nil, err
			}
			reason, err := toolparams.RequireBoundedText(
				params.Params,
				changepreview.FieldReason,
				maxPayrollNoteChars,
			)
			if err != nil {
				return nil, err
			}
			return &timesheetservice.VoidExportRequest{
				ID:         id,
				Reason:     reason,
				TenantInfo: toolparams.TenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *timesheetservice.VoidExportRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*timesheetservice.ExportChange, error) {
			return exporter.PlanVoidExport(ctx, req)
		},
		Refused: func(*timesheetservice.VoidExportRequest) string {
			return "Would void a payroll run."
		},
		Render: func(
			_ *timesheetservice.VoidExportRequest,
			change *timesheetservice.ExportChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(
				payrollExportRecord(change.Before),
				change.Before,
				change.After,
				append(
					workforcekit.Options(payrollExportFields...),
					toolpreview.Volatile(changepreview.FieldVoidedAt),
				)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would void the payroll run and return its %d timesheet weeks to approved.",
				change.Before.TimesheetCount), recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *timesheetservice.VoidExportRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			export, err := exporter.VoidExport(ctx, req)
			if err != nil {
				return nil, err
			}
			return payrollExportResult("voided", export), nil
		},
	})
}

func provideGeneratePayrollExportTool(s *timesheetservice.Service) agenttoolports.AgentTool {
	return newGeneratePayrollExportTool(s)
}

func provideVoidPayrollExportTool(s *timesheetservice.Service) agenttoolports.AgentTool {
	return newVoidPayrollExportTool(s)
}
