package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverPayTools_AreAlwaysAPersonsDecision(t *testing.T) {
	t.Parallel()

	providers := []any{
		provideIssuePayAdvanceTool,
		provideWriteOffPayAdvanceTool,
		provideOpenEscrowAccountTool,
		provideUpdateEscrowAccountTool,
		provideAdjustEscrowAccountTool,
		provideCloseEscrowAccountTool,
		provideAssignPayProfileTool,
		provideEndPayAssignmentTool,
		provideCreateRecurringDeductionTool,
		provideUpdateRecurringDeductionTool,
		provideCreateRecurringEarningTool,
		provideUpdateRecurringEarningTool,
		provideAcceptCarrierInvoiceMatchTool,
		provideAcceptCarrierInvoiceMatchWithVarianceTool,
		providePayWorkerNowTool,
	}
	for _, provider := range providers {
		tool := buildTool(t, provider)
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.True(t, policy.HasEgress(agent.EgressMoney), tool.Name())
		_, previews := tool.(serviceports.ToolPreviewer)
		assert.True(t, previews, tool.Name())
	}
}

type fakeAdvances struct {
	advance     *driverpay.PayAdvance
	issued      *driverpay.PayAdvance
	writtenOff  pulid.ID
	writeReason string
}

func (f *fakeAdvances) IssueAdvance(
	_ context.Context,
	entity *driverpay.PayAdvance,
	_ *serviceports.RequestActor,
) (*driverpay.PayAdvance, error) {
	f.issued = entity

	return entity, nil
}

func (f *fakeAdvances) GetAdvance(
	context.Context,
	repositories.GetPayAdvanceByIDRequest,
) (*driverpay.PayAdvance, error) {
	return f.advance, nil
}

func (f *fakeAdvances) WriteOffAdvance(
	_ context.Context,
	_ pagination.TenantInfo,
	advanceID pulid.ID,
	reason string,
	_ *serviceports.RequestActor,
) (*driverpay.PayAdvance, error) {
	f.writtenOff, f.writeReason = advanceID, reason

	return f.advance, nil
}

func TestIssuePayAdvance_RecordsOnlyOnceAPersonApproves(t *testing.T) {
	t.Parallel()

	advances := &fakeAdvances{}
	tool := &issuePayAdvanceTool{advances: advances}
	workerID := pulid.MustNew("wrk_")
	raw := map[string]any{
		paramWorkerID:        workerID.String(),
		paramDriverPayAmount: "300.00",
		paramAdvanceSource:   "EFSMoneyCode",
		paramAdvanceRef:      "MC-55012",
		paramIssuedDate:      "2026-09-20",
	}

	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	assert.Contains(
		t,
		preview.Summary,
		"Would record an advance of 300.00 USD given as EFSMoneyCode",
	)
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourcePayAdvance, change.Resource)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(300)))

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(raw)), ErrDriverPayNeedsAPerson)
	assert.Nil(t, advances.issued)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	require.NotNil(t, advances.issued)
	assert.Equal(t, int64(30000), advances.issued.AmountMinor)
	assert.Equal(t, driverpay.AdvanceSourceEFSMoneyCode, advances.issued.Source)
	assert.Equal(t, workerID, advances.issued.WorkerID)

	raw[paramAdvanceSource] = "Venmo"
	require.Error(t, tool.Validate(t.Context(), executeParams(raw)))
	raw[paramAdvanceSource] = "Cash"
	raw[paramDriverPayAmount] = "-5"
	require.Error(t, tool.Validate(t.Context(), executeParams(raw)))
}

func TestWriteOffPayAdvance_ShowsWhatIsForgiven(t *testing.T) {
	t.Parallel()

	advance := &driverpay.PayAdvance{
		ID:             pulid.MustNew("padv_"),
		Status:         driverpay.AdvanceStatusPartiallyRecovered,
		AmountMinor:    30000,
		RecoveredMinor: 10000,
		CurrencyCode:   "USD",
		Reference:      "MC-55012",
		Version:        3,
	}
	advances := &fakeAdvances{advance: advance}
	tool := &writeOffPayAdvanceTool{advances: advances}
	raw := map[string]any{
		paramAdvanceID:      advance.ID.String(),
		paramWriteOffReason: "Driver left the company",
	}

	target, ok := tool.Target(raw)
	require.True(t, ok)
	assert.Equal(t, permission.ResourcePayAdvance, target.Resource)

	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	assert.Contains(
		t,
		preview.Summary,
		"write off the 200.00 USD still owed on a 300.00 USD advance",
	)
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Advance MC-55012", change.Label)
	assert.Equal(
		t,
		string(driverpay.AdvanceStatusWrittenOff),
		fieldByPath(t, change, fieldStatus).After,
	)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.NewFromInt(-200)))
	assert.Equal(t, driverpay.AdvanceStatusPartiallyRecovered, advance.Status)

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(raw)), ErrDriverPayNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, advance.ID, advances.writtenOff)
	assert.Equal(t, "Driver left the company", advances.writeReason)

	advance.RecoveredMinor = 30000
	advance.Status = driverpay.AdvanceStatusRecovered
	preview, err = tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

type fakeEscrow struct {
	account  *driverpay.EscrowAccount
	active   bool
	opened   *driverpay.EscrowAccount
	updated  *driverpay.EscrowAccount
	adjusted *driverpayservice.EscrowAdjustmentRequest
	closed   pulid.ID
}

func (f *fakeEscrow) GetEscrowAccount(
	context.Context,
	repositories.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	return f.account, nil
}

func (f *fakeEscrow) PlanOpenEscrowAccount(
	_ context.Context,
	entity *driverpay.EscrowAccount,
	now int64,
) error {
	entity.Status = driverpay.EscrowAccountStatusActive
	if entity.OpenedDate == 0 {
		entity.OpenedDate = now
	}
	if f.active {
		return errortypes.NewValidationError(
			"workerId",
			errortypes.ErrDuplicate,
			"Worker already has an active escrow account",
		)
	}

	return nil
}

func (f *fakeEscrow) OpenEscrowAccount(
	_ context.Context,
	entity *driverpay.EscrowAccount,
	_ *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	f.opened = entity

	return entity, nil
}

func (f *fakeEscrow) PlanUpdateEscrowAccount(
	context.Context,
	*driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	return f.account, nil
}

func (f *fakeEscrow) UpdateEscrowAccount(
	_ context.Context,
	entity *driverpay.EscrowAccount,
	_ *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	f.updated = entity

	return entity, nil
}

func (f *fakeEscrow) AdjustEscrowAccount(
	_ context.Context,
	req *driverpayservice.EscrowAdjustmentRequest,
	_ *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	f.adjusted = req

	return f.account, nil
}

func (f *fakeEscrow) CloseEscrowAccount(
	_ context.Context,
	_ pagination.TenantInfo,
	accountID pulid.ID,
	_ *serviceports.RequestActor,
) (*driverpay.EscrowAccount, error) {
	f.closed = accountID

	return f.account, nil
}

func activeEscrow() *driverpay.EscrowAccount {
	return &driverpay.EscrowAccount{
		ID:                 pulid.MustNew("escr_"),
		WorkerID:           pulid.MustNew("wrk_"),
		Status:             driverpay.EscrowAccountStatusActive,
		BalanceMinor:       80000,
		TargetAmountMinor:  250000,
		AnnualInterestRate: decimal.RequireFromString("2"),
		CurrencyCode:       "USD",
		OpenedDate:         1_780_000_000,
		Version:            5,
		Worker:             &worker.Worker{FirstName: "Ana", LastName: "Ruiz"},
	}
}

func TestOpenEscrowAccount_OneActiveAccountPerDriver(t *testing.T) {
	t.Parallel()

	escrow := &fakeEscrow{}
	tool := &openEscrowAccountTool{escrow: escrow}
	raw := map[string]any{
		paramWorkerID:           pulid.MustNew("wrk_").String(),
		paramEscrowTarget:       "2500",
		paramEscrowInterestRate: "2.5",
	}

	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "a 2500.00 USD target, earning 2.5% a year")
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, int64(250000), escrow.opened.TargetAmountMinor)

	raw[paramEscrowInterestRate] = "140"
	require.Error(t, tool.Validate(t.Context(), executeParams(raw)))

	raw[paramEscrowInterestRate] = "2.5"
	escrow.active = true
	preview, err = tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestUpdateEscrowAccount_ChangesOnlyTheTerms(t *testing.T) {
	t.Parallel()

	escrow := &fakeEscrow{account: activeEscrow()}
	tool := &updateEscrowAccountTool{escrow: escrow}
	id := escrow.account.ID.String()

	require.ErrorIs(t, tool.Validate(t.Context(),
		executeParams(map[string]any{paramEscrowAccountID: id})), errNothingToChange)

	raw := map[string]any{paramEscrowAccountID: id, paramEscrowTarget: "3000"}
	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Escrow account for Ana Ruiz", change.Label)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.NewFromInt(500)))

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, int64(300000), escrow.updated.TargetAmountMinor)
	assert.Equal(t, int64(80000), escrow.updated.BalanceMinor)
	assert.Equal(t, int64(250000), escrow.account.TargetAmountMinor)
}

func TestAdjustEscrowAccount_NeverDrawsBelowZero(t *testing.T) {
	t.Parallel()

	escrow := &fakeEscrow{account: activeEscrow()}
	tool := &adjustEscrowAccountTool{escrow: escrow}
	raw := map[string]any{
		paramEscrowAccountID:   escrow.account.ID.String(),
		paramDriverPayAmount:   "-900.00",
		paramEscrowDescription: "Trailer damage under the lease",
	}

	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)

	raw[paramDriverPayAmount] = "-300.00"
	preview, err = tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(500)))
	assert.Equal(t, int64(80000), escrow.account.BalanceMinor)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, int64(-30000), escrow.adjusted.AmountMinor)
	assert.Equal(t, "Trailer damage under the lease", escrow.adjusted.Description)

	raw[paramDriverPayAmount] = "0"
	require.Error(t, tool.Validate(t.Context(), executeParams(raw)))
}

func TestCloseEscrowAccount_RefundsWhatRemains(t *testing.T) {
	t.Parallel()

	escrow := &fakeEscrow{account: activeEscrow()}
	tool := &closeEscrowAccountTool{escrow: escrow}
	raw := map[string]any{paramEscrowAccountID: escrow.account.ID.String()}

	assert.Equal(t, permission.OpClose, tool.Policy().Operation)

	preview, err := tool.Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "The remaining 800.00 USD is refunded to the driver.")
	change := previewChange(t, preview, 0)
	assert.Equal(
		t,
		string(driverpay.EscrowAccountStatusClosed),
		fieldByPath(t, change, fieldStatus).After,
	)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.TotalAfter.Decimal.IsZero())

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(raw)), ErrDriverPayNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, escrow.account.ID, escrow.closed)

	escrow.account.Status = driverpay.EscrowAccountStatusClosed
	require.Error(t, tool.Validate(t.Context(), executeParams(raw)))
}
