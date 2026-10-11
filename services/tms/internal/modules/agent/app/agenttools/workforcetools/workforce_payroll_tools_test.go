package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePayroll struct {
	guard  *agenttoolstest.WriteGuard
	export *worker.PayrollExport
	ready  int

	generated *timesheetservice.GenerateExportRequest
	voided    *timesheetservice.VoidExportRequest
}

func newFakePayroll() *fakePayroll {
	return &fakePayroll{
		guard: &agenttoolstest.WriteGuard{},
		ready: 3,
		export: &worker.PayrollExport{
			ID:             pulid.MustNew("pxb_"),
			Status:         worker.PayrollExportGenerated,
			PeriodStart:    1_789_516_800,
			PeriodEnd:      1_790_726_400,
			TimesheetCount: 12,
			Version:        1,
		},
	}
}

func (f *fakePayroll) PlanGenerateExport(
	_ context.Context,
	req *timesheetservice.GenerateExportRequest,
) (*timesheetservice.ExportPlan, error) {
	if f.ready == 0 {
		return nil, errortypes.NewValidationError("periodStart", errortypes.ErrInvalidOperation,
			"There are no approved timesheets waiting in that period")
	}
	ids := make([]pulid.ID, 0, f.ready)
	for range f.ready {
		ids = append(ids, pulid.MustNew("wts_"))
	}
	return &timesheetservice.ExportPlan{
		Export: &worker.PayrollExport{
			Status:         worker.PayrollExportGenerated,
			PeriodStart:    req.PeriodStart,
			PeriodEnd:      req.PeriodEnd,
			TimesheetCount: int32(f.ready), //nolint:gosec // a test's handful of weeks
		},
		TimesheetIDs: ids,
	}, nil
}

func (f *fakePayroll) GenerateExport(
	_ context.Context,
	req *timesheetservice.GenerateExportRequest,
) (*worker.PayrollExport, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.generated = req
	return f.export, nil
}

func (f *fakePayroll) PlanVoidExport(
	context.Context,
	*timesheetservice.VoidExportRequest,
) (*timesheetservice.ExportChange, error) {
	if f.export.Status == worker.PayrollExportVoided {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"That run has already been voided")
	}
	after := *f.export
	after.Status = worker.PayrollExportVoided
	return &timesheetservice.ExportChange{Before: f.export, After: &after}, nil
}

func (f *fakePayroll) VoidExport(
	_ context.Context,
	req *timesheetservice.VoidExportRequest,
) (*worker.PayrollExport, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.voided = req
	return f.export, nil
}

func TestGeneratePayrollExport_TakesTheLastDayInclusive(t *testing.T) {
	t.Parallel()

	payroll := newFakePayroll()
	tool := newGeneratePayrollExportTool(payroll)
	raw := map[string]any{
		paramFirstDay:    "2026-09-14",
		paramLastDay:     "2026-09-27",
		common.FieldNote: "Biweekly run",
	}

	preview := agenttoolstest.PreviewWithoutWrites(t, payroll.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	})
	assert.Equal(t, "Would lock 3 approved timesheet weeks into a payroll run.", preview.Summary)

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)), common.ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	require.NotNil(t, payroll.generated)
	assert.Equal(t, int64(1_789_344_000), payroll.generated.PeriodStart)
	assert.Equal(t, int64(1_790_553_600), payroll.generated.PeriodEnd)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceTimesheet, policy.Resource)
	assert.Equal(t, permission.OpExport, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)

	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{paramFirstDay: "2026-09-27", paramLastDay: "2026-09-14"})))
	payroll.ready = 0
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(raw)), "validation runs the service's plan")
}

func TestVoidPayrollExport_KeepsTheReason(t *testing.T) {
	t.Parallel()

	payroll := newFakePayroll()
	tool := newVoidPayrollExportTool(payroll)
	raw := map[string]any{
		paramPayrollExportID: payroll.export.ID.String(),
		common.FieldReason:   "Two weeks were approved in error",
	}

	preview := agenttoolstest.PreviewWithoutWrites(t, payroll.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	})
	assert.Contains(t, preview.Summary, "return its 12 timesheet weeks to approved")
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	require.NotNil(t, payroll.voided)
	assert.Equal(t, "Two weeks were approved in error", payroll.voided.Reason)

	payroll.export.Status = worker.PayrollExportVoided
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(raw)))
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{paramPayrollExportID: payroll.export.ID.String()})))
}
