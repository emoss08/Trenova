package invoicetools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

const moneyLineApplied = "Applied to invoices"

const moneyLineUnapplied = "Unapplied cash on account"

func paymentLabel(payment *customerpayment.Payment) string {
	return "Payment " + payment.DocumentLabel()
}

func paymentRecord(payment *customerpayment.Payment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceCustomerPayment,
		ID:       payment.ID,
		Label:    paymentLabel(payment),
		Version:  common.PinnedVersion(payment.Version),
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
		toolpreview.Only(common.FieldStatus, "reversalReason", "reversedById", "reversedAt"),
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
			Before: common.KnownAmount(money.DecimalFromMinor(before.AppliedAmountMinor)),
			After:  common.KnownAmount(money.DecimalFromMinor(after.AppliedAmountMinor)),
		},
		agent.MoneyLine{
			Label:  moneyLineUnapplied,
			Before: common.KnownAmount(money.DecimalFromMinor(before.UnappliedAmountMinor)),
			After:  common.KnownAmount(money.DecimalFromMinor(after.UnappliedAmountMinor)),
		},
	))

	return change, nil
}

func paymentPreview(
	plan *customerpaymentports.CustomerPaymentChangePreview,
	summary string,
) (*agent.ToolPreview, error) {
	payment, err := paymentChange(plan)
	if err != nil {
		return nil, err
	}
	invoices, err := common.InvoiceBalanceChanges(plan.InvoicesBefore, plan.InvoicesAfter)
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

func RenderApplyPayment(
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
		common.CountOf(len(req.Applications), "invoice"),
		money.FormatMinor(plan.PaymentAfter.UnappliedAmountMinor, currency),
	))
}

func RenderReversePayment(
	_ *customerpaymentports.ReverseCustomerPaymentRequest,
	plan *customerpaymentports.CustomerPaymentChangePreview,
) (*agent.ToolPreview, error) {
	return paymentPreview(plan, fmt.Sprintf(
		"Would reverse payment %s for %s, reopening %s by what it paid and booking a "+
			"reversing entry.",
		plan.PaymentBefore.DocumentLabel(),
		money.FormatMinor(plan.PaymentBefore.AmountMinor, plan.PaymentBefore.CurrencyCode),
		common.CountOf(len(plan.InvoicesAfter), "invoice"),
	))
}
