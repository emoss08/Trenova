package invoicetools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func (t *MatchBankReceiptTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	match, err := t.receipts.PreviewMatch(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	changes, err := bankMatchChanges(match)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would match %s for %s to the posted payment %s, closing its reconciliation.",
		bankReceiptLabel(match.ReceiptBefore),
		money.FormatMinor(match.ReceiptBefore.AmountMinor, match.Payment.CurrencyCode),
		match.Payment.DocumentLabel(),
	)

	return toolpreview.Build(summary, changes...), nil
}

func (t *PostCustomerPaymentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	args, err := t.arguments(ctx, params)
	if err != nil {
		return nil, err
	}

	posted, err := t.payments.PreviewPostAndApply(ctx, args.request, params.Actor)
	if err != nil {
		return nil, err
	}

	payment := posted.Payment
	changes := make([]*agent.RecordChange, 0, len(posted.InvoicesAfter)+3)
	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceCustomerPayment,
			Label:    payment.DocumentLabel(),
		},
		payment,
		toolpreview.Only(
			common.PreviewFieldCustomerID,
			common.PreviewFieldPaymentDate,
			"paymentMethod",
			"referenceNumber",
			"memo",
			"currencyCode",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.PreviewFieldCustomerID: permission.ResourceCustomer,
		}),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			common.PreviewFieldPaymentDate: assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(created, paymentMoney(payment))
	changes = append(changes, created)

	for idx, after := range posted.InvoicesAfter {
		change, iErr := common.InvoicePaymentChange(posted.InvoicesBefore[idx], after)
		if iErr != nil {
			return nil, iErr
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf(
		"Would record a %s %s payment and apply %s of it to %s, leaving %s unapplied on "+
			"the customer's account.",
		money.FormatMinor(payment.AmountMinor, payment.CurrencyCode),
		payment.PaymentMethod,
		money.FormatMinor(payment.AppliedAmountMinor, payment.CurrencyCode),
		common.CountOf(len(posted.InvoicesAfter), "invoice"),
		money.FormatMinor(payment.UnappliedAmountMinor, payment.CurrencyCode),
	)

	if args.receiptID.IsNotNil() {
		match, mErr := t.receipts.PreviewMatchPayment(
			ctx,
			&bankreceiptservice.PreviewMatchPaymentRequest{
				ReceiptID:  args.receiptID,
				TenantInfo: args.request.TenantInfo,
				Payment:    payment,
			},
			params.Actor,
		)
		if mErr != nil {
			return nil, mErr
		}
		matched, mErr := bankMatchChanges(match)
		if mErr != nil {
			return nil, mErr
		}
		changes = append(changes, matched...)
		summary += fmt.Sprintf(
			" %s would be matched to it.",
			stringutils.CapitalizeFirst(bankReceiptLabel(match.ReceiptBefore)),
		)
	}

	return toolpreview.Build(summary, changes...), nil
}

// bankMatchChanges is a receipt matched to a payment, and the reconciliation
// work item the match closes.
func bankMatchChanges(match *bankreceiptservice.MatchPreview) ([]*agent.RecordChange, error) {
	receipt, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceBankReceipt,
			ID:       match.ReceiptBefore.ID,
			Label:    bankReceiptLabel(match.ReceiptBefore),
			Version:  common.PinnedVersion(match.ReceiptBefore.Version),
		},
		match.ReceiptBefore,
		match.ReceiptAfter,
		toolpreview.WithRefs(map[string]permission.Resource{
			"matchedCustomerPaymentId": permission.ResourceCustomerPayment,
			"matchedById":              permission.ResourceUser,
		}),
		toolpreview.Volatile("matchedAt"),
	)
	if err != nil {
		return nil, err
	}

	currency := match.Payment.CurrencyCode
	toolpreview.AttachMoney(receipt, toolpreview.MoneyBlock(currency, agent.MoneyLine{
		Label:  "Unreconciled cash",
		Before: common.KnownAmount(money.DecimalFromMinor(match.ReceiptBefore.AmountMinor)),
		After:  common.KnownAmount(decimal.Zero),
	}))

	changes := []*agent.RecordChange{receipt}
	if match.WorkItemAfter == nil {
		return changes, nil
	}

	item, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceBankReceiptWorkItem,
			ID:       match.WorkItemBefore.ID,
			Label:    "Reconciliation of " + bankReceiptLabel(match.ReceiptBefore),
			Version:  common.PinnedVersion(match.WorkItemBefore.Version),
		},
		match.WorkItemBefore,
		match.WorkItemAfter,
		toolpreview.WithRefs(map[string]permission.Resource{
			"resolvedByUserId": permission.ResourceUser,
		}),
		toolpreview.Volatile("resolvedAt"),
	)
	if err != nil {
		return nil, err
	}
	item.Operation = agent.PreviewOperationArchive

	return append(changes, item), nil
}

// paymentMoney splits a new payment into what it pays and what stays on the
// customer's account; together they are the payment.
func paymentMoney(payment *customerpayment.Payment) *agent.MoneyPreview {
	lines := []agent.MoneyLine{
		{
			Label: "Applied to invoices",
			After: common.KnownAmount(money.DecimalFromMinor(payment.AppliedAmountMinor)),
		},
		{
			Label: "Unapplied cash on account",
			After: common.KnownAmount(money.DecimalFromMinor(payment.UnappliedAmountMinor)),
		},
	}

	return toolpreview.MoneyBlock(payment.CurrencyCode, lines...)
}

func bankReceiptLabel(receipt *bankreceipt.BankReceipt) string {
	if reference := strings.TrimSpace(receipt.ReferenceNumber); reference != "" {
		return "bank receipt " + reference
	}

	return "the bank receipt of " + time.Unix(receipt.ReceiptDate, 0).UTC().Format("2006-01-02")
}
