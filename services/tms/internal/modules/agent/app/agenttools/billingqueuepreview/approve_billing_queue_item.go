package billingqueuepreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type ApprovalInvoicePlanner interface {
	PreviewApprovalInvoice(
		ctx context.Context,
		req *invoiceports.CreateInvoiceFromBillingQueueRequest,
	) (*invoiceports.CreateInvoiceFromBillingQueueResult, error)
}

// draftInvoiceChange is the draft approval creates, with its totals.
func DraftInvoiceChange(draft *invoice.Invoice) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceInvoice,
			Label:    invoicepreview.InvoiceLabel(draft),
		},
		draft,
		toolpreview.Only(
			"number",
			changepreview.FieldStatus,
			"billType",
			"billToName",
			"invoiceDate",
			"dueDate",
			"paymentTerm",
			"currencyCode",
		),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		draft.CurrencyCode,
		agent.MoneyLine{
			Label: MoneyLineFreight,
			After: changepreview.KnownAmount(draft.SubtotalAmount),
		},
		agent.MoneyLine{
			Label: MoneyLineAccessorials,
			After: changepreview.KnownAmount(draft.OtherAmount),
		},
	), toolpreview.SensitiveAs("totalAmount", "subtotalAmount", "otherAmount"))

	return change, nil
}
