package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceivables struct {
	change      *serviceports.CustomerPaymentChangePreview
	credit      *serviceports.CreditMemoApplicationPreview
	refusal     error
	guard       *writeGuard
	applied     *serviceports.ApplyCustomerPaymentRequest
	reversed    *serviceports.ReverseCustomerPaymentRequest
	creditApply *serviceports.ApplyCreditMemoRequest
	unapplied   *serviceports.UnapplyCreditMemoApplicationRequest
}

func (f *fakeReceivables) changePlan() (*serviceports.CustomerPaymentChangePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.change, nil
}

func (f *fakeReceivables) creditPlan() (*serviceports.CreditMemoApplicationPreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.credit, nil
}

func (f *fakeReceivables) ApplyUnapplied(
	_ context.Context,
	req *serviceports.ApplyCustomerPaymentRequest,
	_ *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change.PaymentAfter, nil
}

func (f *fakeReceivables) Reverse(
	_ context.Context,
	req *serviceports.ReverseCustomerPaymentRequest,
	_ *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reversed = req

	return f.change.PaymentAfter, nil
}

func (f *fakeReceivables) ApplyCreditMemo(
	_ context.Context,
	req *serviceports.ApplyCreditMemoRequest,
	_ *serviceports.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.creditApply = req

	return f.credit.Applications, nil
}

func (f *fakeReceivables) UnapplyCreditMemoApplication(
	_ context.Context,
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	_ *serviceports.RequestActor,
) (*customerpayment.CreditMemoApplication, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.unapplied = req

	return f.credit.Applications[0], nil
}

func (f *fakeReceivables) PreviewApplyUnapplied(
	context.Context,
	*serviceports.ApplyCustomerPaymentRequest,
	*serviceports.RequestActor,
) (*serviceports.CustomerPaymentChangePreview, error) {
	return f.changePlan()
}

func (f *fakeReceivables) PreviewReverse(
	context.Context,
	*serviceports.ReverseCustomerPaymentRequest,
	*serviceports.RequestActor,
) (*serviceports.CustomerPaymentChangePreview, error) {
	return f.changePlan()
}

func (f *fakeReceivables) PreviewApplyCreditMemo(
	context.Context,
	*serviceports.ApplyCreditMemoRequest,
	*serviceports.RequestActor,
) (*serviceports.CreditMemoApplicationPreview, error) {
	return f.creditPlan()
}

func (f *fakeReceivables) PreviewUnapplyCreditMemoApplication(
	context.Context,
	*serviceports.UnapplyCreditMemoApplicationRequest,
	*serviceports.RequestActor,
) (*serviceports.CreditMemoApplicationPreview, error) {
	return f.creditPlan()
}

func receivableInvoice(number string, totalMinor, appliedMinor int64) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		Number:             number,
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		CurrencyCode:       "USD",
		TotalAmount:        decimal.NewFromInt(totalMinor / 100),
		TotalAmountMinor:   totalMinor,
		AppliedAmount:      decimal.NewFromInt(appliedMinor / 100),
		AppliedAmountMinor: appliedMinor,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		Version:            4,
	}
}

func applyChangePlan() *serviceports.CustomerPaymentChangePreview {
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
	inv := receivableInvoice("INV-77", 5000, 0)
	invAfter := *inv
	invAfter.AppliedAmountMinor = 5000
	invAfter.SettlementStatus = invoice.SettlementStatusPaid

	return &serviceports.CustomerPaymentChangePreview{
		PaymentBefore:  before,
		PaymentAfter:   &after,
		InvoicesBefore: []*invoice.Invoice{inv},
		InvoicesAfter:  []*invoice.Invoice{&invAfter},
		Journal: &serviceports.JournalPreview{
			AccountingDate: 1_790_000_000,
			FiscalPeriodID: pulid.MustNew("fp_"),
			EntryStatus:    "Posted",
			Lines: []serviceports.JournalLinePreview{
				{GLAccountID: pulid.MustNew("gla_"), DebitMinor: 5000},
				{GLAccountID: pulid.MustNew("gla_"), CreditMinor: 5000},
			},
		},
	}
}

func applyPaymentParams(paymentID, invoiceID pulid.ID) serviceports.ToolExecuteParams {
	return executeParams(map[string]any{
		paramCustomerPaymentID: paymentID.String(),
		paramAccountingDate:    "2026-09-20",
		paramApplications: []any{
			map[string]any{paramInvoiceID: invoiceID.String(), "amount": "50.00"},
		},
	})
}

func TestApplyCustomerPayment_PreviewShowsThePaymentTheInvoicesAndTheEntry(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{change: applyChangePlan(), guard: &writeGuard{}}
	tool := newApplyCustomerPaymentTool(receivables).(serviceports.ToolPreviewer)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	preview := previewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Contains(t, preview.Summary, "Would apply 50.00 USD of payment ACH-9001 to 1 invoice")
	assert.Contains(t, preview.Summary, "leaving 0.00 USD unapplied")
	require.Len(t, preview.Changes, 3)
	payment := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceCustomerPayment, payment.Resource)
	require.NotNil(t, payment.Money)
	inv := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceInvoice, inv.Resource)
	assert.Equal(t, "Paid", fieldByPath(t, inv, "settlementStatus").After)
	journal := previewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceJournalEntry, journal.Resource)
}

func TestApplyCustomerPayment_OnlyAPersonApplies(t *testing.T) {
	t.Parallel()

	receivables := &fakeReceivables{change: applyChangePlan()}
	tool := newApplyCustomerPaymentTool(receivables)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
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
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCustomerPayment, target.Resource)
}

func TestApplyCustomerPayment_RefusesBadApplications(t *testing.T) {
	t.Parallel()

	tool := newApplyCustomerPaymentTool(&fakeReceivables{change: applyChangePlan()})
	paymentID := pulid.MustNew("cpay_").String()
	invoiceID := pulid.MustNew("inv_").String()
	cases := map[string]map[string]any{
		"no applications": {
			paramCustomerPaymentID: paymentID, paramAccountingDate: "2026-09-20",
		},
		"a repeated invoice": {
			paramCustomerPaymentID: paymentID, paramAccountingDate: "2026-09-20",
			paramApplications: []any{
				map[string]any{paramInvoiceID: invoiceID, "amount": "10"},
				map[string]any{paramInvoiceID: invoiceID, "amount": "10"},
			},
		},
		"a bad date": {
			paramCustomerPaymentID: paymentID, paramAccountingDate: "20 Sept",
			paramApplications: []any{map[string]any{paramInvoiceID: invoiceID, "amount": "10"}},
		},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
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
	tool := newApplyCustomerPaymentTool(receivables)
	params := applyPaymentParams(
		receivables.change.PaymentBefore.ID,
		receivables.change.InvoicesBefore[0].ID,
	)

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func reverseChangePlan() *serviceports.CustomerPaymentChangePreview {
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

	receivables := &fakeReceivables{change: reverseChangePlan(), guard: &writeGuard{}}
	tool := newReverseCustomerPaymentTool(receivables)
	params := executeParams(map[string]any{
		paramCustomerPaymentID: receivables.change.PaymentBefore.ID.String(),
		paramAccountingDate:    "2026-09-21",
		paramReason:            "Chargeback",
	})

	preview := previewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would reverse payment ACH-9001")
	assert.Equal(t, "Reversed", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.reversed)
	assert.Equal(t, "Chargeback", receivables.reversed.Reason)
	assert.False(t, tool.Policy().Reversible)
}

func creditApplicationPlan() *serviceports.CreditMemoApplicationPreview {
	memo := receivableInvoice("CM-12", -8000, 0)
	memo.BillType = billingqueue.BillTypeCreditMemo
	memoAfter := *memo
	memoAfter.AppliedAmountMinor = 3000
	target := receivableInvoice("INV-78", 10000, 0)
	targetAfter := *target
	targetAfter.AppliedAmountMinor = 3000
	targetAfter.SettlementStatus = invoice.SettlementStatusPartiallyPaid

	return &serviceports.CreditMemoApplicationPreview{
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

	receivables := &fakeReceivables{credit: creditApplicationPlan(), guard: &writeGuard{}}
	tool := newApplyCreditMemoTool(receivables)
	params := executeParams(map[string]any{
		paramCreditMemoID:   receivables.credit.CreditMemoBefore.ID.String(),
		paramAccountingDate: "2026-09-21",
		paramApplications: []any{map[string]any{
			paramInvoiceID: receivables.credit.InvoicesBefore[0].ID.String(),
			"amount":       "30.00",
		}},
	})

	preview := previewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would apply 30.00 USD of credit memo CM-12 to 1 invoice")
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, "PartiallyPaid",
		fieldByPath(t, previewChange(t, preview, 1), "settlementStatus").After)
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 2).Operation)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.creditApply)
	assert.Equal(t, int64(3000), receivables.creditApply.Applications[0].AppliedAmountMinor)

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
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

	receivables := &fakeReceivables{credit: plan, guard: &writeGuard{}}
	tool := newUnapplyCreditMemoTool(receivables)
	params := executeParams(map[string]any{
		paramCreditMemoApplicationID: applied.ID.String(),
		paramReason:                  "Wrong invoice",
	})

	preview := previewWithoutWrites(t, receivables.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would take back 30.00 USD of credit memo CM-12")
	application := previewChange(t, preview, 2)
	assert.Equal(t, "Unapplied", fieldByPath(t, application, "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, receivables.unapplied)
	assert.Equal(t, "Wrong invoice", receivables.unapplied.Reason)
}
