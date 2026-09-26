package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceiptMatcher struct {
	receipt       *bankreceipt.BankReceipt
	paymentAmount int64
	matched       *serviceports.MatchBankReceiptRequest
	previewed     *serviceports.MatchBankReceiptRequest
	actor         *serviceports.RequestActor
}

func (f *fakeReceiptMatcher) Get(
	_ context.Context,
	_ *serviceports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return f.receipt, nil
}

func (f *fakeReceiptMatcher) PreviewMatch(
	_ context.Context,
	req *serviceports.MatchBankReceiptRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptservice.MatchPreview, error) {
	f.previewed = req

	return f.previewMatch(&customerpayment.Payment{
		ID:           req.PaymentID,
		AmountMinor:  f.paymentAmount,
		Status:       customerpayment.StatusPosted,
		CurrencyCode: "USD",
	}, actor)
}

func (f *fakeReceiptMatcher) PreviewMatchPayment(
	_ context.Context,
	req *bankreceiptservice.PreviewMatchPaymentRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptservice.MatchPreview, error) {
	return f.previewMatch(req.Payment, actor)
}

func (f *fakeReceiptMatcher) previewMatch(
	payment *customerpayment.Payment,
	actor *serviceports.RequestActor,
) (*bankreceiptservice.MatchPreview, error) {
	if f.receipt.AmountMinor != payment.AmountMinor {
		return nil, errors.New("Bank receipt amount must match customer payment amount")
	}
	before := *f.receipt
	after := before
	after.Status = bankreceipt.StatusMatched
	after.MatchedCustomerPaymentID = payment.ID
	after.MatchedByID = actor.UserID
	item := &bankreceiptworkitem.WorkItem{
		ID:     pulid.MustNew("brwi_"),
		Status: bankreceiptworkitem.StatusOpen,
	}
	resolved := *item
	resolved.Status = bankreceiptworkitem.StatusResolved
	resolved.ResolutionType = bankreceiptworkitem.ResolutionMatchedToPayment

	return &bankreceiptservice.MatchPreview{
		ReceiptBefore:  &before,
		ReceiptAfter:   &after,
		Payment:        payment,
		WorkItemBefore: item,
		WorkItemAfter:  &resolved,
	}, nil
}

func (f *fakeReceiptMatcher) Match(
	_ context.Context,
	req *serviceports.MatchBankReceiptRequest,
	actor *serviceports.RequestActor,
) (*bankreceipt.BankReceipt, error) {
	f.matched = req
	f.actor = actor

	return &bankreceipt.BankReceipt{ID: req.ReceiptID, Status: bankreceipt.StatusMatched}, nil
}

type fakePaymentPoster struct {
	posted    *serviceports.PostCustomerPaymentRequest
	previewed *serviceports.PostCustomerPaymentRequest
	id        pulid.ID
}

func (f *fakePaymentPoster) PreviewPostAndApply(
	_ context.Context,
	req *serviceports.PostCustomerPaymentRequest,
	_ *serviceports.RequestActor,
) (*serviceports.CustomerPaymentPostPreview, error) {
	f.previewed = req
	payment := &customerpayment.Payment{
		CustomerID:      req.CustomerID,
		AmountMinor:     req.AmountMinor,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		PaymentDate:     req.PaymentDate,
		Status:          customerpayment.StatusPosted,
		CurrencyCode:    "USD",
	}
	before := make([]*invoice.Invoice, 0, len(req.Applications))
	after := make([]*invoice.Invoice, 0, len(req.Applications))
	for idx, application := range req.Applications {
		payment.Applications = append(payment.Applications, &customerpayment.Application{
			InvoiceID:          application.InvoiceID,
			AppliedAmountMinor: application.AppliedAmountMinor,
		})
		open := &invoice.Invoice{
			ID:               application.InvoiceID,
			Number:           fmt.Sprintf("INV-%d", idx+1),
			TotalAmountMinor: 150_000,
			CurrencyCode:     "USD",
		}
		paid := *open
		paid.ApplyPaymentMinor(application.AppliedAmountMinor + application.ShortPayAmountMinor)
		before = append(before, open)
		after = append(after, &paid)
	}
	payment.SyncAmounts()

	return &serviceports.CustomerPaymentPostPreview{
		Payment:        payment,
		InvoicesBefore: before,
		InvoicesAfter:  after,
	}, nil
}

func (f *fakePaymentPoster) PostAndApply(
	_ context.Context,
	req *serviceports.PostCustomerPaymentRequest,
	_ *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	f.posted = req
	if f.id.IsNil() {
		f.id = pulid.MustNew("cpay_")
	}

	return &customerpayment.Payment{ID: f.id, AmountMinor: req.AmountMinor}, nil
}

type fakeWorkItemResolver struct {
	item      *bankreceiptworkitem.WorkItem
	resolved  *serviceports.ResolveBankReceiptWorkItemRequest
	dismissed *serviceports.DismissBankReceiptWorkItemRequest
	guard     writeGuard
}

func (f *fakeWorkItemResolver) Get(
	_ context.Context,
	_ *serviceports.GetBankReceiptWorkItemRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	if f.item == nil {
		return nil, errortypes.NewNotFoundError("Bank receipt work item not found")
	}
	copied := *f.item

	return &copied, nil
}

func (f *fakeWorkItemResolver) Resolve(
	_ context.Context,
	req *serviceports.ResolveBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.resolved = req
	if f.item == nil {
		return &bankreceiptworkitem.WorkItem{
			ID:     req.WorkItemID,
			Status: bankreceiptworkitem.StatusResolved,
		}, nil
	}
	if err := bankreceiptworkitemservice.ResolveWorkItem(
		f.item, req, actor.UserID, timeutils.NowUnix(),
	); err != nil {
		return nil, err
	}

	return f.item, nil
}

func (f *fakeWorkItemResolver) Dismiss(
	_ context.Context,
	req *serviceports.DismissBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.dismissed = req
	if f.item == nil {
		return &bankreceiptworkitem.WorkItem{
			ID:     req.WorkItemID,
			Status: bankreceiptworkitem.StatusDismissed,
		}, nil
	}
	if err := bankreceiptworkitemservice.DismissWorkItem(
		f.item, req, actor.UserID, timeutils.NowUnix(),
	); err != nil {
		return nil, err
	}

	return f.item, nil
}

type allowingPermissions struct {
	serviceports.PermissionEngine

	allowed  bool
	lastReq  *serviceports.PermissionCheckRequest
	requests int
}

func (p *allowingPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	p.lastReq = req
	p.requests++

	return &serviceports.PermissionCheckResult{Allowed: p.allowed}, nil
}

func unmatchedReceipt(amountMinor int64) *bankreceipt.BankReceipt {
	return &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		AmountMinor:     amountMinor,
		ReferenceNumber: "ACH 4471",
		Status:          bankreceipt.StatusException,
	}
}

func TestMatchBankReceipt_MatchesAsTheApprover(t *testing.T) {
	t.Parallel()

	receipts := &fakeReceiptMatcher{}
	tool := newMatchBankReceiptTool(receipts)
	receiptID, paymentID := pulid.MustNew("brcpt_"), pulid.MustNew("cpay_")
	params := memoryParams(map[string]any{
		"bankReceiptId":     receiptID.String(),
		"customerPaymentId": paymentID.String(),
	})

	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, receipts.matched)
	assert.Equal(t, receiptID, receipts.matched.ReceiptID)
	assert.Equal(t, paymentID, receipts.matched.PaymentID)
	assert.Equal(t, params.OrganizationID, receipts.matched.TenantInfo.OrgID)
	assert.Same(t, params.Actor, receipts.actor)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceBankReceipt, target.Resource)
	assert.Equal(t, receiptID, target.ID)
}

// The preview reads both records and refuses what the service would refuse,
// so an approver sees "the amounts differ" before approving rather than an
// execution failure after.
func TestMatchBankReceipt_PreviewRefusesAnAmountMismatch(t *testing.T) {
	t.Parallel()

	receipt := unmatchedReceipt(125_000)
	receipts := &fakeReceiptMatcher{receipt: receipt, paymentAmount: 120_000}
	tool := newMatchBankReceiptTool(receipts)
	previewer := tool.(serviceports.ToolPreviewer)
	params := memoryParams(map[string]any{
		"bankReceiptId":     receipt.ID.String(),
		"customerPaymentId": pulid.MustNew("cpay_").String(),
	})

	_, err := previewer.Preview(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match")

	receipts.paymentAmount = 125_000
	preview, err := previewer.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Nil(t, receipts.matched, "a preview must not match")
	assert.Contains(t, preview.Summary, "bank receipt ACH 4471")

	matched := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceBankReceipt, matched.Resource)
	assert.Equal(t, "Matched", findField(t, matched, "status").After)
	payment := findField(t, matched, "matchedCustomerPaymentId")
	require.NotNil(t, payment.AfterRef)
	assert.Equal(t, permission.ResourceCustomerPayment, payment.AfterRef.Resource)
	require.NotNil(t, matched.Money)
	assert.True(t, matched.Money.Delta.Decimal.Equal(decimal.NewFromInt(-1250)))

	closed := findChange(t, preview, agent.PreviewOperationArchive)
	assert.Equal(t, permission.ResourceBankReceiptWorkItem, closed.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *receipts.previewed, *receipts.matched,
		"the preview and the write must make the same match")
}

func TestPostCustomerPayment_PostsAndAppliesInMinorUnits(t *testing.T) {
	t.Parallel()

	poster := &fakePaymentPoster{}
	tool := newPostCustomerPaymentTool(
		poster,
		&fakeReceiptMatcher{},
		&allowingPermissions{allowed: true},
	)
	customerID, invoiceID := pulid.MustNew("cus_"), pulid.MustNew("inv_")

	require.NoError(t, tool.Execute(t.Context(), memoryParams(map[string]any{
		"customerId":      customerID.String(),
		"amount":          "1250.00",
		"paymentDate":     "2026-09-18",
		"paymentMethod":   "Check",
		"referenceNumber": "CHK 1002",
		"applications": []any{
			map[string]any{
				"invoiceId":      invoiceID.String(),
				"amount":         "1000.00",
				"shortPayAmount": "12.50",
			},
		},
	})))

	require.NotNil(t, poster.posted)
	assert.Equal(t, customerID, poster.posted.CustomerID)
	assert.Equal(t, int64(125_000), poster.posted.AmountMinor)
	assert.Equal(t, customerpayment.MethodCheck, poster.posted.PaymentMethod)
	assert.Equal(t, int64(1_789_689_600), poster.posted.PaymentDate)
	assert.Equal(t, poster.posted.PaymentDate, poster.posted.AccountingDate)
	require.Len(t, poster.posted.Applications, 1)
	assert.Equal(t, int64(100_000), poster.posted.Applications[0].AppliedAmountMinor)
	assert.Equal(t, int64(1_250), poster.posted.Applications[0].ShortPayAmountMinor)
}

func TestPostCustomerPayment_RefusesApplicationsBeyondTheAmount(t *testing.T) {
	t.Parallel()

	poster := &fakePaymentPoster{}
	tool := newPostCustomerPaymentTool(
		poster,
		&fakeReceiptMatcher{},
		&allowingPermissions{allowed: true},
	)

	err := tool.Execute(t.Context(), memoryParams(map[string]any{
		"customerId":  pulid.MustNew("cus_").String(),
		"amount":      "100.00",
		"paymentDate": "2026-09-18",
		"applications": []any{
			map[string]any{"invoiceId": pulid.MustNew("inv_").String(), "amount": "60.00"},
			map[string]any{"invoiceId": pulid.MustNew("inv_").String(), "amount": "60.00"},
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "120.00")
	assert.Nil(t, poster.posted)
}

// A receipt is matched by the payment that records it, in the same step,
// once the approver's right to match is confirmed. The receipt's reference
// becomes the payment's when none was given, so the payment can be found
// by what the bank called it.
func TestPostCustomerPayment_MatchesTheReceiptItRecords(t *testing.T) {
	t.Parallel()

	receipt := unmatchedReceipt(125_000)
	receipts := &fakeReceiptMatcher{receipt: receipt}
	poster := &fakePaymentPoster{}
	perms := &allowingPermissions{allowed: true}
	tool := newPostCustomerPaymentTool(poster, receipts, perms)
	params := memoryParams(map[string]any{
		"customerId":    pulid.MustNew("cus_").String(),
		"amount":        "1250.00",
		"paymentDate":   "2026-09-18",
		"bankReceiptId": receipt.ID.String(),
	})

	require.NoError(t, tool.Execute(t.Context(), params))

	assert.Equal(t, "ACH 4471", poster.posted.ReferenceNumber)
	require.NotNil(t, receipts.matched)
	assert.Equal(t, receipt.ID, receipts.matched.ReceiptID)
	assert.Equal(t, poster.id, receipts.matched.PaymentID)
	require.NotNil(t, perms.lastReq)
	assert.Equal(t, permission.ResourceBankReceipt.String(), perms.lastReq.Resource)
	assert.Equal(t, permission.OpUpdate, perms.lastReq.Operation)
	assert.Equal(t, params.Actor.UserID, perms.lastReq.UserID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, receipt.ID, target.ID)
}

func TestPostCustomerPayment_RefusesAReceiptOfADifferentAmountOrWithoutTheRight(t *testing.T) {
	t.Parallel()

	receipt := unmatchedReceipt(125_000)
	poster := &fakePaymentPoster{}
	tool := newPostCustomerPaymentTool(
		poster,
		&fakeReceiptMatcher{receipt: receipt},
		&allowingPermissions{allowed: true},
	)

	err := tool.Execute(t.Context(), memoryParams(map[string]any{
		"customerId":    pulid.MustNew("cus_").String(),
		"amount":        "1200.00",
		"paymentDate":   "2026-09-18",
		"bankReceiptId": receipt.ID.String(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same amount")
	assert.Nil(t, poster.posted, "nothing is posted when the receipt could not be matched")

	denied := newPostCustomerPaymentTool(
		poster,
		&fakeReceiptMatcher{receipt: receipt},
		&allowingPermissions{allowed: false},
	)
	err = denied.Execute(t.Context(), memoryParams(map[string]any{
		"customerId":    pulid.MustNew("cus_").String(),
		"amount":        "1250.00",
		"paymentDate":   "2026-09-18",
		"bankReceiptId": receipt.ID.String(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "may not match bank receipts")
	assert.Nil(t, poster.posted)
}

func TestPostCustomerPayment_PreviewShowsWhatStaysUnapplied(t *testing.T) {
	t.Parallel()

	receipt := unmatchedReceipt(125_000)
	poster := &fakePaymentPoster{}
	receipts := &fakeReceiptMatcher{receipt: receipt}
	tool := newPostCustomerPaymentTool(poster, receipts, &allowingPermissions{allowed: true})
	params := memoryParams(map[string]any{
		"customerId":    pulid.MustNew("cus_").String(),
		"amount":        "1250.00",
		"paymentDate":   "2026-09-18",
		"bankReceiptId": receipt.ID.String(),
		"applications": []any{
			map[string]any{"invoiceId": pulid.MustNew("inv_").String(), "amount": "1000.00"},
		},
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Nil(t, poster.posted, "a preview must not post")
	assert.Nil(t, receipts.matched, "a preview must not match")
	assert.Contains(t, preview.Summary, "leaving 250.00 USD unapplied")
	assert.Contains(t, preview.Summary, "Bank receipt ACH 4471 would be matched")

	created := findChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceCustomerPayment, created.Resource)
	assert.Equal(t, "ACH 4471", created.Label)
	require.NotNil(t, created.Money)
	assert.True(t, created.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(1250)))
	require.Len(t, created.Money.Lines, 2)
	assert.True(t, created.Money.Lines[1].After.Decimal.Equal(decimal.NewFromInt(250)))

	var invoiceChange *agent.RecordChange
	for i := range preview.Changes {
		if preview.Changes[i].Resource == permission.ResourceInvoice {
			invoiceChange = &preview.Changes[i]
		}
	}
	require.NotNil(t, invoiceChange)
	assert.Equal(t, "Invoice INV-1", invoiceChange.Label)
	require.NotNil(t, invoiceChange.Money)
	assert.True(t, invoiceChange.Money.Lines[0].Before.Decimal.Equal(decimal.NewFromInt(1500)))
	assert.True(t, invoiceChange.Money.Lines[0].After.Decimal.Equal(decimal.NewFromInt(500)))

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *poster.previewed, *poster.posted,
		"the preview and the write must post the same payment")
}

func TestResolveBankReceiptWorkItem_DismissesAFalsePositiveAndResolvesTheRest(t *testing.T) {
	t.Parallel()

	items := &fakeWorkItemResolver{}
	tool := newResolveBankReceiptWorkItemTool(items)
	id := pulid.MustNew("brwi_")

	require.NoError(t, tool.Execute(t.Context(), memoryParams(map[string]any{
		"workItemId": id.String(),
		"resolution": "MarkedFalsePositive",
		"note":       "Interest credit from the bank, not a customer payment.",
	})))
	require.NotNil(t, items.dismissed)
	assert.Equal(t, id, items.dismissed.WorkItemID)
	assert.Contains(t, items.dismissed.ResolutionNote, "Interest credit")

	require.NoError(t, tool.Execute(t.Context(), memoryParams(map[string]any{
		"workItemId": id.String(),
		"resolution": "RequiresExternalFollowUp",
		"note":       "Payer unknown; ask the bank for the remittance advice.",
	})))
	require.NotNil(t, items.resolved)
	assert.Equal(
		t,
		bankreceiptworkitem.ResolutionRequiresExternalFollowUp,
		items.resolved.ResolutionType,
	)
}

func TestResolveBankReceiptWorkItem_RefusesAMatchResolutionAndAnOverlongNote(t *testing.T) {
	t.Parallel()

	items := &fakeWorkItemResolver{}
	tool := newResolveBankReceiptWorkItemTool(items)
	id := pulid.MustNew("brwi_")

	err := tool.Execute(t.Context(), memoryParams(map[string]any{
		"workItemId": id.String(),
		"resolution": "MatchedToPayment",
		"note":       "x",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "match_bank_receipt")

	err = tool.Execute(t.Context(), memoryParams(map[string]any{
		"workItemId": id.String(),
		"resolution": "MarkedFalsePositive",
		"note":       strings.Repeat("x", maxResolutionNoteChars+1),
	}))
	require.Error(t, err)
	assert.Nil(t, items.dismissed)
	assert.Nil(t, items.resolved)
}
