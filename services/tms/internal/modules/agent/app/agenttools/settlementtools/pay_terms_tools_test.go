package settlementtools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPayStoreDown = errors.New("driver pay store is unavailable")

type fakeAssigner struct {
	existing []*driverpay.WorkerPayAssignment
	failure  error
	assigned *driverpay.WorkerPayAssignment
}

func (f *fakeAssigner) PlanAssignment(
	_ context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpayservice.AssignmentPlan, error) {
	if f.failure != nil {
		return nil, f.failure
	}
	ended, err := driverpayservice.PlanEndOverlapping(entity, f.existing)
	if err != nil {
		return nil, err
	}

	return &driverpayservice.AssignmentPlan{
		Profile: &driverpay.PayProfile{
			Name:           "Linehaul OO",
			Classification: driverpay.PayeeClassificationOwnerOperator,
		},
		Ended: ended,
	}, nil
}

func (f *fakeAssigner) AssignProfileToWorker(
	_ context.Context,
	entity *driverpay.WorkerPayAssignment,
	_ *permissionports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	f.assigned = entity

	return entity, nil
}

func TestAssignPayProfile_EndsTheAssignmentInForce(t *testing.T) {
	t.Parallel()

	current := &driverpay.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		EffectiveFrom: 1_700_000_000,
		SplitPercent:  decimal.NewFromInt(100),
		Version:       2,
	}
	assigner := &fakeAssigner{existing: []*driverpay.WorkerPayAssignment{current}}
	tool := &paytools.AssignPayProfileTool{Assignments: assigner}
	raw := map[string]any{
		common.ParamWorkerID:       pulid.MustNew("wrk_").String(),
		paytools.ParamPayProfileID: pulid.MustNew("dpp_").String(),
		common.ParamEffectiveFrom:  "2026-10-01",
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary,
		"Would pay the driver under Linehaul OO (OwnerOperator) from Oct 1, 2026 at a 100% share.")
	assert.Contains(t, preview.Summary, "1 assignment in force ends that day.")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, agenttoolstest.PreviewChange(t, preview, 0).Operation)
	assert.Equal(t, current.ID, agenttoolstest.PreviewChange(t, preview, 1).EntityID)
	assert.Nil(t, current.EffectiveTo)

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)), common.ErrDriverPayNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.True(t, assigner.assigned.SplitPercent.Equal(decimal.NewFromInt(100)))

	raw[common.ParamEffectiveFrom] = "2023-01-01"
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)

	assigner.failure = errPayStoreDown
	_, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.ErrorIs(t, err, errPayStoreDown)
}

type fakeAssignmentEnder struct {
	assignment *driverpay.WorkerPayAssignment
	ended      pulid.ID
	endDate    int64
}

func (f *fakeAssignmentEnder) GetAssignment(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*driverpay.WorkerPayAssignment, error) {
	return f.assignment, nil
}

func (f *fakeAssignmentEnder) EndAssignment(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
	endDate int64,
	_ *permissionports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	f.ended, f.endDate = assignmentID, endDate

	return f.assignment, nil
}

func TestEndWorkerPayAssignment_EndsAfterItStarts(t *testing.T) {
	t.Parallel()

	ender := &fakeAssignmentEnder{assignment: &driverpay.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		EffectiveFrom: periodStartUnix,
		PayProfile:    &driverpay.PayProfile{Name: "Company solo"},
	}}
	tool := &paytools.EndPayAssignmentTool{Assignments: ender}
	assert.Equal(t, permission.OpAssign, tool.Policy().Operation)

	early := map[string]any{
		paytools.ParamAssignmentID: ender.assignment.ID.String(),
		paytools.ParamEndDate:      "2020-01-01",
	}
	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(early))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)

	raw := map[string]any{
		paytools.ParamAssignmentID: ender.assignment.ID.String(),
		paytools.ParamEndDate:      "2026-12-31",
	}
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would end Pay assignment: Company solo on Dec 31, 2026.")
	assert.Nil(t, ender.assignment.EffectiveTo)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.Equal(t, ender.assignment.ID, ender.ended)
	assert.NotZero(t, ender.endDate)
}

type fakeRecurringBook struct {
	existing *driverpay.RecurringDeduction
	refusal  error
	escrow   bool
	created  *driverpay.RecurringDeduction
	updated  *driverpay.RecurringDeduction
}

func (f *fakeRecurringBook) get(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*driverpay.RecurringDeduction, error) {
	return f.existing, nil
}

func (f *fakeRecurringBook) check(
	_ context.Context,
	_ *driverpay.RecurringDeduction,
	escrow bool,
) error {
	f.escrow = escrow

	return f.refusal
}

func (f *fakeRecurringBook) create(
	_ context.Context,
	entity *driverpay.RecurringDeduction,
	_ bool,
	_ *permissionports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	f.created = entity

	return entity, nil
}

func (f *fakeRecurringBook) update(
	_ context.Context,
	entity *driverpay.RecurringDeduction,
	_ *permissionports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	f.updated = entity

	return entity, nil
}

func TestCreateRecurringDeduction_CanPayIntoEscrow(t *testing.T) {
	t.Parallel()

	book := &fakeRecurringBook{}
	tool := &RecurringPayTool[driverpay.RecurringDeduction]{kind: deductionKind(), book: book}
	_, targeted := tool.Target(map[string]any{"deductionId": pulid.MustNew("rded_").String()})
	assert.False(t, targeted)

	raw := map[string]any{
		common.ParamWorkerID:        pulid.MustNew("wrk_").String(),
		common.ParamPayCodeID:       pulid.MustNew("payc_").String(),
		paramRecurringDescription:   "Escrow contribution",
		common.ParamDriverPayAmount: "100.00",
		paramRecurringStart:         "2026-10-01",
		paramRecurringCap:           "2500",
		paramEscrowContribution:     true,
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.True(t, book.escrow)
	assert.Contains(t, preview.Summary,
		"Would set up a recurring deduction of 100.00 USD for the driver: Escrow contribution.")
	assert.Contains(t, preview.Summary, "paid into the driver's escrow account")

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)), common.ErrDriverPayNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	require.NotNil(t, book.created)
	assert.Equal(t, int64(10000), book.created.AmountMinor)
	require.NotNil(t, book.created.TotalCapMinor)
	assert.Equal(t, int64(250000), *book.created.TotalCapMinor)
	assert.Equal(t, driverpay.DeductionFrequencyEverySettlement, book.created.Frequency)

	delete(raw, common.ParamDriverPayAmount)
	require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(raw)))

	raw[common.ParamDriverPayAmount] = "100.00"
	book.refusal = errortypes.NewNotFoundError("Pay code not found")
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)

	book.refusal = errPayStoreDown
	require.ErrorIs(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(raw)), errPayStoreDown)
}

func TestUpdateRecurringDeduction_ChangesOnlyWhatIsGiven(t *testing.T) {
	t.Parallel()

	existing := &driverpay.RecurringDeduction{
		ID:           pulid.MustNew("rded_"),
		Status:       driverpay.DeductionStatusActive,
		Frequency:    driverpay.DeductionFrequencyEverySettlement,
		Description:  "Occupational accident insurance",
		AmountMinor:  4500,
		CurrencyCode: "USD",
		Version:      7,
	}
	book := &fakeRecurringBook{existing: existing}
	tool := &RecurringPayTool[driverpay.RecurringDeduction]{
		kind:   deductionKind(),
		book:   book,
		update: true,
	}
	id := existing.ID.String()

	target, ok := tool.Target(map[string]any{"deductionId": id})
	require.True(t, ok)
	assert.Equal(t, permission.ResourceRecurringDeduction, target.Resource)

	require.ErrorIs(t, tool.Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{"deductionId": id})), common.ErrNothingToChange)
	require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"deductionId":        id,
		paramRecurringStatus: "Cancelled",
	})))

	raw := map[string]any{"deductionId": id, paramRecurringStatus: "Paused"}
	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary,
		`Would change the driver's recurring deduction "Occupational accident insurance".`)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Paused", agenttoolstest.FieldByPath(t, change, common.FieldStatus).After)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.Equal(t, driverpay.DeductionStatusPaused, book.updated.Status)
	assert.Equal(t, int64(4500), book.updated.AmountMinor)
	assert.Equal(t, driverpay.DeductionStatusActive, existing.Status)
}
