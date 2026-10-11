package agenttoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

const (
	moneyLineApplied   = "Applied to invoices"
	moneyLineUnapplied = "Unapplied cash on account"
	moneyLineCredit    = "Credit remaining"
)

func paymentLabel(payment *customerpayment.Payment) string {
	return "Payment " + payment.DocumentLabel()
}

func paymentRecord(payment *customerpayment.Payment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceCustomerPayment,
		ID:       payment.ID,
		Label:    paymentLabel(payment),
		Version:  pinnedVersion(payment.Version),
	}
}

func paymentChange(
	plan *customerpaymentports.CustomerPaymentChangePreview,
) (*agent.RecordChange, error) {
	before, after := plan.PaymentBefore, plan.PaymentAfter
	change, err := toolpreview.Changed(
		paymentRecord(before),
		before,
		after,
		toolpreview.Only(fieldStatus, "reversalReason", "reversedById", "reversedAt"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"reversedById": permission.ResourceUser,
		}),
		toolpreview.Volatile("reversedAt"),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		before.CurrencyCode,
		agent.MoneyLine{
			Label:  moneyLineApplied,
			Before: knownAmount(money.DecimalFromMinor(before.AppliedAmountMinor)),
			After:  knownAmount(money.DecimalFromMinor(after.AppliedAmountMinor)),
		},
		agent.MoneyLine{
			Label:  moneyLineUnapplied,
			Before: knownAmount(money.DecimalFromMinor(before.UnappliedAmountMinor)),
			After:  knownAmount(money.DecimalFromMinor(after.UnappliedAmountMinor)),
		},
	))

	return change, nil
}

func invoiceBalanceChanges(before, after []*invoice.Invoice) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, len(after))
	for idx, invAfter := range after {
		if invAfter == nil || idx >= len(before) || before[idx] == nil {
			continue
		}
		change, err := invoicePaymentChange(before[idx], invAfter)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
}

func paymentPreview(
	plan *customerpaymentports.CustomerPaymentChangePreview,
	summary string,
) (*agent.ToolPreview, error) {
	payment, err := paymentChange(plan)
	if err != nil {
		return nil, err
	}
	invoices, err := invoiceBalanceChanges(plan.InvoicesBefore, plan.InvoicesAfter)
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(invoices)+2)
	changes = append(changes, payment)
	changes = append(changes, invoices...)
	if plan.Journal != nil {
		journal, journalErr := journalEntryChange(
			plan.Journal,
			plan.PaymentBefore.CurrencyCode,
			"Journal entry for "+paymentLabel(plan.PaymentBefore),
		)
		if journalErr != nil {
			return nil, journalErr
		}
		changes = append(changes, journal)
	}

	return toolpreview.Build(summary, changes...), nil
}

func renderApplyPayment(
	req *customerpaymentports.ApplyCustomerPaymentRequest,
	plan *customerpaymentports.CustomerPaymentChangePreview,
) (*agent.ToolPreview, error) {
	currency := plan.PaymentBefore.CurrencyCode
	return paymentPreview(plan, fmt.Sprintf(
		"Would apply %s of payment %s to %s, leaving %s unapplied on the customer's account.",
		money.FormatMinor(
			plan.PaymentAfter.AppliedAmountMinor-plan.PaymentBefore.AppliedAmountMinor,
			currency,
		),
		plan.PaymentBefore.DocumentLabel(),
		countOf(len(req.Applications), "invoice"),
		money.FormatMinor(plan.PaymentAfter.UnappliedAmountMinor, currency),
	))
}

func renderReversePayment(
	_ *customerpaymentports.ReverseCustomerPaymentRequest,
	plan *customerpaymentports.CustomerPaymentChangePreview,
) (*agent.ToolPreview, error) {
	return paymentPreview(plan, fmt.Sprintf(
		"Would reverse payment %s for %s, reopening %s by what it paid and booking a "+
			"reversing entry.",
		plan.PaymentBefore.DocumentLabel(),
		money.FormatMinor(plan.PaymentBefore.AmountMinor, plan.PaymentBefore.CurrencyCode),
		countOf(len(plan.InvoicesAfter), "invoice"),
	))
}

func creditMemoChange(
	plan *customerpaymentports.CreditMemoApplicationPreview,
) (*agent.RecordChange, error) {
	before, after := plan.CreditMemoBefore, plan.CreditMemoAfter
	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInvoice,
			ID:       before.ID,
			Label:    creditMemoLabel(before),
			Version:  pinnedVersion(before.Version),
		},
		before,
		after,
		toolpreview.Only("appliedAmount", "settlementStatus"),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(before.CurrencyCode, agent.MoneyLine{
		Label:  moneyLineCredit,
		Before: knownAmount(money.DecimalFromMinor(before.CreditRemainingMinor())),
		After:  knownAmount(money.DecimalFromMinor(after.CreditRemainingMinor())),
	}), toolpreview.SensitiveAs("totalAmount", "appliedAmount"))

	return change, nil
}

func creditMemoLabel(memo *invoice.Invoice) string {
	return "Credit memo " + memo.Number
}

var creditApplicationRefs = map[string]permission.Resource{
	"creditMemoInvoiceId": permission.ResourceInvoice,
	paramInvoiceID:        permission.ResourceInvoice,
	"unappliedById":       permission.ResourceUser,
}

func creditApplicationChange(
	plan *customerpaymentports.CreditMemoApplicationPreview,
	application *customerpayment.CreditMemoApplication,
) (*agent.RecordChange, error) {
	record := toolpreview.Record{
		Resource: permission.ResourceCustomerPayment,
		Label:    "Credit applied from " + creditMemoLabel(plan.CreditMemoBefore),
	}
	if plan.ApplicationBefore != nil {
		record.ID = plan.ApplicationBefore.ID
		change, err := toolpreview.Changed(
			record,
			plan.ApplicationBefore,
			application,
			toolpreview.Only(fieldStatus, "unappliedReason", "unappliedById", "unappliedAt"),
			toolpreview.WithRefs(creditApplicationRefs),
			toolpreview.Volatile("unappliedAt"),
		)
		if err != nil {
			return nil, err
		}
		change.Operation = agent.PreviewOperationArchive

		return change, nil
	}

	change, err := toolpreview.Create(
		record,
		application,
		toolpreview.Only(fieldStatus, "creditMemoInvoiceId", paramInvoiceID, paramAccountingDate),
		toolpreview.WithRefs(creditApplicationRefs),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			paramAccountingDate: assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		plan.CreditMemoBefore.CurrencyCode,
		agent.MoneyLine{
			Label: "Applied",
			After: knownAmount(money.DecimalFromMinor(application.AppliedAmountMinor)),
		},
	))

	return change, nil
}

func creditPreview(
	plan *customerpaymentports.CreditMemoApplicationPreview,
	summary string,
) (*agent.ToolPreview, error) {
	memo, err := creditMemoChange(plan)
	if err != nil {
		return nil, err
	}
	invoices, err := invoiceBalanceChanges(plan.InvoicesBefore, plan.InvoicesAfter)
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(invoices)+len(plan.Applications)+1)
	changes = append(changes, memo)
	changes = append(changes, invoices...)
	for _, application := range plan.Applications {
		change, changeErr := creditApplicationChange(plan, application)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(summary, changes...), nil
}

func creditAppliedMinor(plan *customerpaymentports.CreditMemoApplicationPreview) int64 {
	var total int64
	for _, application := range plan.Applications {
		total += application.AppliedAmountMinor
	}

	return total
}

func renderApplyCredit(
	req *customerpaymentports.ApplyCreditMemoRequest,
	plan *customerpaymentports.CreditMemoApplicationPreview,
) (*agent.ToolPreview, error) {
	currency := plan.CreditMemoBefore.CurrencyCode
	return creditPreview(plan, fmt.Sprintf(
		"Would apply %s of credit memo %s to %s, leaving %s of credit.",
		money.FormatMinor(creditAppliedMinor(plan), currency),
		plan.CreditMemoBefore.Number,
		countOf(len(req.Applications), "invoice"),
		money.FormatMinor(plan.CreditMemoAfter.CreditRemainingMinor(), currency),
	))
}

func renderUnapplyCredit(
	_ *customerpaymentports.UnapplyCreditMemoApplicationRequest,
	plan *customerpaymentports.CreditMemoApplicationPreview,
) (*agent.ToolPreview, error) {
	target := "the invoice"
	if len(plan.InvoicesBefore) > 0 && plan.InvoicesBefore[0] != nil {
		target = invoiceLabel(plan.InvoicesBefore[0])
	}

	return creditPreview(plan, fmt.Sprintf(
		"Would take back %s of credit memo %s from %s, reopening its balance by that much.",
		money.FormatMinor(creditAppliedMinor(plan), plan.CreditMemoBefore.CurrencyCode),
		plan.CreditMemoBefore.Number,
		target,
	))
}
