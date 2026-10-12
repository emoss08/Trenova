package receivabletools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceivables struct {
	change      *customerpaymentports.CustomerPaymentChangePreview
	credit      *customerpaymentports.CreditMemoApplicationPreview
	refusal     error
	guard       *agenttoolstest.WriteGuard
	applied     *customerpaymentports.ApplyCustomerPaymentRequest
	reversed    *customerpaymentports.ReverseCustomerPaymentRequest
	creditApply *customerpaymentports.ApplyCreditMemoRequest
	unapplied   *customerpaymentports.UnapplyCreditMemoApplicationRequest
}

func (f *fakeReceivables) changePlan() (*customerpaymentports.CustomerPaymentChangePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.change, nil
}

func (f *fakeReceivables) creditPlan() (*customerpaymentports.CreditMemoApplicationPreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.credit, nil
}

func (f *fakeReceivables) ApplyUnapplied(
	_ context.Context,
	req *customerpaymentports.ApplyCustomerPaymentRequest,
	_ *permissionports.RequestActor,
) (*customerpayment.Payment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change.PaymentAfter, nil
}

func (f *fakeReceivables) Reverse(
	_ context.Context,
	req *customerpaymentports.ReverseCustomerPaymentRequest,
	_ *permissionports.RequestActor,
) (*customerpayment.Payment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reversed = req

	return f.change.PaymentAfter, nil
}

func (f *fakeReceivables) ApplyCreditMemo(
	_ context.Context,
	req *customerpaymentports.ApplyCreditMemoRequest,
	_ *permissionports.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.creditApply = req

	return f.credit.Applications, nil
}

func (f *fakeReceivables) UnapplyCreditMemoApplication(
	_ context.Context,
	req *customerpaymentports.UnapplyCreditMemoApplicationRequest,
	_ *permissionports.RequestActor,
) (*customerpayment.CreditMemoApplication, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.unapplied = req

	return f.credit.Applications[0], nil
}

func (f *fakeReceivables) PreviewApplyUnapplied(
	context.Context,
	*customerpaymentports.ApplyCustomerPaymentRequest,
	*permissionports.RequestActor,
) (*customerpaymentports.CustomerPaymentChangePreview, error) {
	return f.changePlan()
}

func (f *fakeReceivables) PreviewReverse(
	context.Context,
	*customerpaymentports.ReverseCustomerPaymentRequest,
	*permissionports.RequestActor,
) (*customerpaymentports.CustomerPaymentChangePreview, error) {
	return f.changePlan()
}

func (f *fakeReceivables) PreviewApplyCreditMemo(
	context.Context,
	*customerpaymentports.ApplyCreditMemoRequest,
	*permissionports.RequestActor,
) (*customerpaymentports.CreditMemoApplicationPreview, error) {
	return f.creditPlan()
}

func (f *fakeReceivables) PreviewUnapplyCreditMemoApplication(
	context.Context,
	*customerpaymentports.UnapplyCreditMemoApplicationRequest,
	*permissionports.RequestActor,
) (*customerpaymentports.CreditMemoApplicationPreview, error) {
	return f.creditPlan()
}

func applyChangePlan() *customerpaymentports.CustomerPaymentChangePreview {
	before := &customerpayment.Payment{
		ID:                   pulid.MustNew("cpay_"),
		AmountMinor:          15000,
		AppliedAmountMinor:   10000,
		UnappliedAmountMinor: 5000,
		Status:               customerpayment.StatusPosted,
		ReferenceNumber:      "ACH-9001",
		CurrencyCode:         "USD",
		Version:              2,
	}
	after := *before
	after.AppliedAmountMinor = 15000
	after.UnappliedAmountMinor = 0
	inv := agenttoolstest.ReceivableInvoice("INV-77", 5000, 0)
	invAfter := *inv
	invAfter.AppliedAmountMinor = 5000
	invAfter.SettlementStatus = invoice.SettlementStatusPaid

	return &customerpaymentports.CustomerPaymentChangePreview{
		PaymentBefore:  before,
		PaymentAfter:   &after,
		InvoicesBefore: []*invoice.Invoice{inv},
		InvoicesAfter:  []*invoice.Invoice{&invAfter},
		Journal: &kernelports.JournalPreview{
			AccountingDate: 1_790_000_000,
			FiscalPeriodID: pulid.MustNew("fp_"),
			EntryStatus:    "Posted",
			Lines: []kernelports.JournalLinePreview{
				{GLAccountID: pulid.MustNew("gla_"), DebitMinor: 5000},
				{GLAccountID: pulid.MustNew("gla_"), CreditMinor: 5000},
			},
		},
	}
}

func applyPaymentParams(paymentID, invoiceID pulid.ID) agentports.ToolExecuteParams {
	return agenttoolstest.ExecuteParams(map[string]any{
		ParamCustomerPaymentID:     paymentID.String(),
		common.ParamAccountingDate: "2026-09-20",
		common.ParamApplications: []any{
			map[string]any{common.ParamInvoiceID: invoiceID.String(), "amount": "50.00"},
		},
	})
}

func TestApplyCustomerPayment_PreviewShowsThePaymentTheInvoicesAndTheEntry(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{change: applyChangePlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewApplyCustomerPaymentTool(receivables).(agentports.ToolPreviewer)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	preview := agenttoolstest.PreviewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Contains(t, preview.Summary, "Would apply 50.00 USD of payment ACH-9001 to 1 invoice")
	assert.Contains(t, preview.Summary, "leaving 0.00 USD unapplied")
	require.Len(t, preview.Changes, 3)
	payment := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceCustomerPayment, payment.Resource)
	require.NotNil(t, payment.Money)
	inv := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceInvoice, inv.Resource)
	assert.Equal(t, "Paid", agenttoolstest.FieldByPath(t, inv, "settlementStatus").After)
	journal := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceJournalEntry, journal.Resource)
}

func TestApplyCustomerPayment_OnlyAPersonApplies(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{change: applyChangePlan()}
	tool := NewApplyCustomerPaymentTool(receivables)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	assert.Nil(t, receivables.applied)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.applied)
	require.Len(t, receivables.applied.Applications, 1)
	assert.Equal(t, int64(5000), receivables.applied.Applications[0].AppliedAmountMinor)
	want := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, want, receivables.applied.AccountingDate)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceCustomerPayment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCustomerPayment, target.Resource)
}

func TestApplyCustomerPayment_RefusesBadApplications(t *testing.T) {
	t.Parallel()

	tool := NewApplyCustomerPaymentTool(&fakeReceivables{change: applyChangePlan()})
	paymentID := pulid.MustNew("cpay_").String()
	invoiceID := pulid.MustNew("inv_").String()
	cases := map[string]map[string]any{
		"no applications": {
			ParamCustomerPaymentID: paymentID, common.ParamAccountingDate: "2026-09-20",
		},
		"a repeated invoice": {
			ParamCustomerPaymentID: paymentID, common.ParamAccountingDate: "2026-09-20",
			common.ParamApplications: []any{
				map[string]any{common.ParamInvoiceID: invoiceID, "amount": "10"},
				map[string]any{common.ParamInvoiceID: invoiceID, "amount": "10"},
			},
		},
		"a bad date": {
			ParamCustomerPaymentID: paymentID, common.ParamAccountingDate: "20 Sept",
			common.ParamApplications: []any{map[string]any{common.ParamInvoiceID: invoiceID, "amount": "10"}},
		},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
				agenttoolstest.ExecuteParams(raw)))
		})
	}
}

func TestApplyCustomerPayment_ARefusalIsAWouldFail(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{
		change: applyChangePlan(),
		refusal: errortypes.NewValidationError(
			"paymentId",
			errortypes.ErrInvalidOperation,
			"Customer payment has no unapplied amount remaining",
		),
	}
	tool := NewApplyCustomerPaymentTool(receivables)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
}

func reverseChangePlan() *customerpaymentports.CustomerPaymentChangePreview {
	plan := applyChangePlan()
	before := *plan.PaymentAfter
	after := before
	after.Status = customerpayment.StatusReversed
	after.ReversalReason = "Chargeback"
	plan.PaymentBefore = &before
	plan.PaymentAfter = &after
	plan.InvoicesBefore, plan.InvoicesAfter = plan.InvoicesAfter, plan.InvoicesBefore

	return plan
}

func TestReverseCustomerPayment_PreviewAndPersonOnly(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{change: reverseChangePlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewReverseCustomerPaymentTool(receivables)
	params := agenttoolstest.ExecuteParams(map[string]any{
		ParamCustomerPaymentID:     receivables.change.PaymentBefore.ID.String(),
		common.ParamAccountingDate: "2026-09-21",
		common.ParamReason:         "Chargeback",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would reverse payment ACH-9001")
	assert.Equal(t, "Reversed", agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.reversed)
	assert.Equal(t, "Chargeback", receivables.reversed.Reason)
	assert.False(t, tool.Policy().Reversible)
}

func creditApplicationPlan() *customerpaymentports.CreditMemoApplicationPreview {
	memo := agenttoolstest.ReceivableInvoice("CM-12", -8000, 0)
	memo.BillType = billingqueue.BillTypeCreditMemo
	memoAfter := *memo
	memoAfter.AppliedAmountMinor = 3000
	target := agenttoolstest.ReceivableInvoice("INV-78", 10000, 0)
	targetAfter := *target
	targetAfter.AppliedAmountMinor = 3000
	targetAfter.SettlementStatus = invoice.SettlementStatusPartiallyPaid

	return &customerpaymentports.CreditMemoApplicationPreview{
		CreditMemoBefore: memo,
		CreditMemoAfter:  &memoAfter,
		InvoicesBefore:   []*invoice.Invoice{target},
		InvoicesAfter:    []*invoice.Invoice{&targetAfter},
		Applications: []*customerpayment.CreditMemoApplication{{
			CreditMemoInvoiceID: memo.ID,
			InvoiceID:           target.ID,
			AppliedAmountMinor:  3000,
			AccountingDate:      1_790_000_000,
			Status:              customerpayment.CreditApplicationStatusApplied,
		}},
	}
}

func TestApplyCreditMemo_PreviewAndPersonOnly(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{credit: creditApplicationPlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewApplyCreditMemoTool(receivables)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramCreditMemoID:          receivables.credit.CreditMemoBefore.ID.String(),
		common.ParamAccountingDate: "2026-09-21",
		common.ParamApplications: []any{map[string]any{
			common.ParamInvoiceID: receivables.credit.InvoicesBefore[0].ID.String(),
			"amount":              "30.00",
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would apply 30.00 USD of credit memo CM-12 to 1 invoice")
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, "PartiallyPaid",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 1), "settlementStatus").After)
	assert.Equal(t, agent.PreviewOperationCreate, agenttoolstest.PreviewChange(t, preview, 2).Operation)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.creditApply)
	assert.Equal(t, int64(3000), receivables.creditApply.Applications[0].AppliedAmountMinor)

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoice, target.Resource)
}

func TestUnapplyCreditMemo_PreviewAndPersonOnly(t *testing.T) {
	t.Parallel()

	plan := creditApplicationPlan()
	plan.CreditMemoBefore, plan.CreditMemoAfter = plan.CreditMemoAfter, plan.CreditMemoBefore
	plan.InvoicesBefore, plan.InvoicesAfter = plan.InvoicesAfter, plan.InvoicesBefore
	applied := *plan.Applications[0]
	applied.ID = pulid.MustNew("cmapp_")
	plan.ApplicationBefore = &applied
	unapplied := applied
	unapplied.Status = customerpayment.CreditApplicationStatusUnapplied
	unapplied.UnappliedReason = "Wrong invoice"
	plan.Applications = []*customerpayment.CreditMemoApplication{&unapplied}

	receivables := &fakeReceivables{credit: plan, guard: &agenttoolstest.WriteGuard{}}
	tool := NewUnapplyCreditMemoTool(receivables)
	params := agenttoolstest.ExecuteParams(map[string]any{
		ParamCreditMemoApplicationID: applied.ID.String(),
		common.ParamReason:           "Wrong invoice",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would take back 30.00 USD of credit memo CM-12")
	application := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, "Unapplied", agenttoolstest.FieldByPath(t, application, "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.unapplied)
	assert.Equal(t, "Wrong invoice", receivables.unapplied.Reason)
}
