package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramPayrollExportID = "payrollExportId"
	paramFirstDay        = "firstDay"
	paramLastDay         = "lastDay"
	kindPayrollExport    = "payroll run"
	maxPayrollNoteChars  = 1000
)

var payrollExportFields = []string{
	fieldStatus, fieldPeriodStart, fieldPeriodEnd, "timesheetCount", "regularMinutes",
	"overtimeMinutes", "paidLeaveMinutes", "generatedById", "generatedAt", fieldVoidedAt,
	"voidReason", fieldNote,
}

type payrollExporter interface {
	PlanGenerateExport(
		ctx context.Context,
		req *timesheetservice.GenerateExportRequest,
	) (*timesheetservice.ExportPlan, error)
	GenerateExport(
		ctx context.Context,
		req *timesheetservice.GenerateExportRequest,
	) (*worker.PayrollExport, error)
	PlanVoidExport(
		ctx context.Context,
		req *timesheetservice.VoidExportRequest,
	) (*timesheetservice.ExportChange, error)
	VoidExport(
		ctx context.Context,
		req *timesheetservice.VoidExportRequest,
	) (*worker.PayrollExport, error)
}

var _ payrollExporter = (*timesheetservice.Service)(nil)

func payrollToolProviders() []any {
	return []any{provideGeneratePayrollExportTool, provideVoidPayrollExportTool}
}

func payrollExportRecord(export *worker.PayrollExport) toolpreview.Record {
	return wfRecord(permission.ResourceTimesheet, export.ID,
		fmt.Sprintf("Payroll run %s to %s", dayText(export.PeriodStart),
			dayText(export.PeriodEnd-secondsPerDay)), export.Version)
}

func payrollExportResult(action string, export *worker.PayrollExport) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   kindPayrollExport,
		IDs:    map[string]string{paramPayrollExportID: export.ID.String()},
	}
}

func payrollSpec(name, description, rationale string) *receivableSpec {
	spec := personOnly(wfSpec(name, description, rationale, permission.ResourceTimesheet,
		permission.OpExport))
	spec.egress = agent.EgressMoney
	spec.reversible = false
	spec.artifact = ""
	spec.searchTerms = []string{"payroll", "payroll export", "payroll run", "timesheets"}
	return spec
}

func newGeneratePayrollExportTool(exporter payrollExporter) serviceports.AgentTool {
	spec := withSchema(payrollSpec(
		"generate_payroll_export",
		"Draft a payroll run over a pay period for a person to approve. Every approved "+
			"timesheet week starting between the first and last day given that has not gone "+
			"to payroll yet is totalled and locked into the run. Refused when no approved week "+
			"is waiting.",
		"Locks approved hours into a payroll run that pays people, so a person approves it "+
			"and it runs as them; void_payroll_export takes it back.",
	), map[string]any{
		paramFirstDay: dayProperty("The first day of the pay period."),
		paramLastDay:  dayProperty("The last day of the pay period, included."),
		fieldNote:     stringProperty("A note kept on the run.", maxPayrollNoteChars),
	}, paramFirstDay, paramLastDay)

	return newReceivableTool(spec, receivablePlan[
		*timesheetservice.GenerateExportRequest, *timesheetservice.ExportPlan,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*timesheetservice.GenerateExportRequest, error) {
			first, err := requireScheduleDay(params.Params, paramFirstDay)
			if err != nil {
				return nil, err
			}
			last, err := requireScheduleDay(params.Params, paramLastDay)
			if err != nil {
				return nil, err
			}
			if last < first {
				return nil, fmt.Errorf("parameter %q cannot be before %q", paramLastDay,
					paramFirstDay)
			}
			note, err := boundedText(params.Params, fieldNote, maxPayrollNoteChars)
			if err != nil {
				return nil, err
			}
			return &timesheetservice.GenerateExportRequest{
				PeriodStart: first,
				PeriodEnd:   last + secondsPerDay,
				Note:        note,
				TenantInfo:  tenantFrom(*params),
				UserID:      params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *timesheetservice.GenerateExportRequest,
			_ *serviceports.ToolExecuteParams,
		) (*timesheetservice.ExportPlan, error) {
			return exporter.PlanGenerateExport(ctx, req)
		},
		refused: func(*timesheetservice.GenerateExportRequest) string {
			return "Would generate a payroll run."
		},
		render: func(
			_ *timesheetservice.GenerateExportRequest,
			plan *timesheetservice.ExportPlan,
		) (*agent.ToolPreview, error) {
			record := payrollExportRecord(plan.Export)
			record.ID = pulid.Nil
			record.Version = nil
			created, err := toolpreview.Create(record, plan.Export,
				append(wfOptions(payrollExportFields...),
					toolpreview.Volatile("generatedAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would lock %d approved timesheet weeks into a payroll run.",
				len(plan.TimesheetIDs)), created), nil
		},
		run: func(
			ctx context.Context,
			req *timesheetservice.GenerateExportRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			export, err := exporter.GenerateExport(ctx, req)
			if err != nil {
				return nil, err
			}
			return payrollExportResult("generated", export), nil
		},
	})
}

func newVoidPayrollExportTool(exporter payrollExporter) serviceports.AgentTool {
	spec := targeting(withSchema(payrollSpec(
		"void_payroll_export",
		"Draft voiding a payroll run for a person to approve, with the reason: the run is "+
			"taken back and every week in it returns to approved, ready for another run. "+
			"A run already voided is refused.",
		"Takes back a payroll run that payroll may already have paid from, so a person "+
			"approves it and it runs as them; a voided run stays voided.",
	), map[string]any{
		paramPayrollExportID: idProperty("The payroll run, from list_payroll_exports. " +
			"Never guess one."),
		fieldReason: stringProperty("Why the run is voided, kept on it.", maxPayrollNoteChars),
	}, paramPayrollExportID, fieldReason), paramPayrollExportID, permission.ResourceTimesheet)

	return newReceivableTool(spec, receivablePlan[
		*timesheetservice.VoidExportRequest, *timesheetservice.ExportChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*timesheetservice.VoidExportRequest, error) {
			id, err := requirePulid(params.Params, paramPayrollExportID)
			if err != nil {
				return nil, err
			}
			reason, err := requireBoundedText(params.Params, fieldReason, maxPayrollNoteChars)
			if err != nil {
				return nil, err
			}
			return &timesheetservice.VoidExportRequest{
				ID:         id,
				Reason:     reason,
				TenantInfo: tenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *timesheetservice.VoidExportRequest,
			_ *serviceports.ToolExecuteParams,
		) (*timesheetservice.ExportChange, error) {
			return exporter.PlanVoidExport(ctx, req)
		},
		refused: func(*timesheetservice.VoidExportRequest) string {
			return "Would void a payroll run."
		},
		render: func(
			_ *timesheetservice.VoidExportRequest,
			change *timesheetservice.ExportChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(payrollExportRecord(change.Before),
				change.Before, change.After,
				append(wfOptions(payrollExportFields...), toolpreview.Volatile(fieldVoidedAt))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would void the payroll run and return its %d timesheet weeks to approved.",
				change.Before.TimesheetCount), recorded), nil
		},
		run: func(
			ctx context.Context,
			req *timesheetservice.VoidExportRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			export, err := exporter.VoidExport(ctx, req)
			if err != nil {
				return nil, err
			}
			return payrollExportResult("voided", export), nil
		},
	})
}

func provideGeneratePayrollExportTool(s *timesheetservice.Service) serviceports.AgentTool {
	return newGeneratePayrollExportTool(s)
}

func provideVoidPayrollExportTool(s *timesheetservice.Service) serviceports.AgentTool {
	return newVoidPayrollExportTool(s)
}
