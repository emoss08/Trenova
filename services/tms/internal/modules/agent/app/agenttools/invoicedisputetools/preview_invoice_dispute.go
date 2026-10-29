package invoicedisputetools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

const fieldDisputeStatus = "disputeStatus"

var disputeFields = []string{
	changepreview.FieldStatus,
	paramDisputeReasonCode,
	paramDisputedAmount,
	fieldDisputeNotes,
	paramDisputeResolution,
	paramResolutionNotes,
	changepreview.FieldResolvedByID,
	changepreview.FieldResolvedAt,
	changepreview.FieldOpenedByID,
	toolparams.ParamInvoiceID,
	changepreview.FieldCustomerID,
}

var disputeRefs = map[string]permission.Resource{
	toolparams.ParamInvoiceID:       permission.ResourceInvoice,
	changepreview.FieldCustomerID:   permission.ResourceCustomer,
	changepreview.FieldOpenedByID:   permission.ResourceUser,
	changepreview.FieldResolvedByID: permission.ResourceUser,
}

func disputeLabel(inv *invoice.Invoice) string {
	return "Dispute on " + invoicepreview.InvoiceLabel(inv)
}

func disputePreview(
	plan *invoiceports.InvoiceDisputePreview,
	summary string,
) (*agentpreview.ToolPreview, error) {
	record := toolpreview.Record{
		Resource: permission.ResourceInvoiceDispute,
		Label:    disputeLabel(plan.InvoiceBefore),
	}
	opts := []toolpreview.Option{
		toolpreview.Only(disputeFields...),
		toolpreview.WithRefs(disputeRefs),
		toolpreview.Volatile(changepreview.FieldResolvedAt),
	}

	var (
		change *agentpreview.RecordChange
		err    error
	)
	if plan.Before == nil {
		change, err = toolpreview.Create(record, plan.After, opts...)
	} else {
		record.ID = plan.Before.ID
		record.Version = changepreview.PinnedVersion(plan.Before.Version)
		change, err = toolpreview.Changed(record, plan.Before, plan.After, opts...)
	}
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		plan.InvoiceBefore.CurrencyCode,
		agentpreview.MoneyLine{
			Label: "Disputed",
			After: changepreview.KnownAmount(plan.After.DisputedAmount),
		},
	), toolpreview.SensitiveAs(paramDisputedAmount))

	changes := []*agentpreview.RecordChange{change}
	if plan.InvoiceBefore.DisputeStatus != plan.InvoiceAfter.DisputeStatus {
		flag, flagErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceInvoice,
			ID:       plan.InvoiceBefore.ID,
			Label:    invoicepreview.InvoiceLabel(plan.InvoiceBefore),
			Version:  changepreview.PinnedVersion(plan.InvoiceBefore.Version),
		}, plan.InvoiceBefore, plan.InvoiceAfter, toolpreview.Only(fieldDisputeStatus))
		if flagErr != nil {
			return nil, flagErr
		}
		changes = append(changes, flag)
	}

	return toolpreview.Build(fmt.Sprintf(
		summary,
		plan.After.ReasonCode,
		money.FormatMinor(plan.After.DisputedAmountMinor, plan.InvoiceBefore.CurrencyCode),
		invoicepreview.InvoiceLabel(plan.InvoiceBefore),
	), changes...), nil
}
