package common

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

// invoicePaymentChange is one invoice the payment pays: what has been applied
// to it and its open balance, before and after.
func InvoicePaymentChange(before, after *invoice.Invoice) (*agent.RecordChange, error) {
	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInvoice,
			ID:       before.ID,
			Label:    InvoiceLabel(before),
			Version:  PinnedVersion(before.Version),
		},
		before,
		after,
		toolpreview.Only("appliedAmount", "settlementStatus"),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(before.CurrencyCode, agent.MoneyLine{
		Label:  "Open balance",
		Before: KnownAmount(money.DecimalFromMinor(before.OpenBalanceMinor())),
		After:  KnownAmount(money.DecimalFromMinor(after.OpenBalanceMinor())),
	}), toolpreview.SensitiveAs("totalAmount", "appliedAmount"))

	return change, nil
}

func InvoiceLabel(inv *invoice.Invoice) string {
	if number := strings.TrimSpace(inv.Number); number != "" {
		return "Invoice " + number
	}

	return "Invoice"
}
