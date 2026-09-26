package agenttoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
)

const fieldDisputeStatus = "disputeStatus"

var (
	disputeFields = []string{
		fieldStatus,
		paramDisputeReasonCode,
		paramDisputedAmount,
		paramDisputeNotes,
		paramDisputeResolution,
		paramResolutionNotes,
		fieldResolvedByID,
		fieldResolvedAt,
		fieldOpenedByID,
		paramInvoiceID,
		fieldCustomerID,
	}
	disputeRefs = map[string]permission.Resource{
		paramInvoiceID:    permission.ResourceInvoice,
		fieldCustomerID:   permission.ResourceCustomer,
		fieldOpenedByID:   permission.ResourceUser,
		fieldResolvedByID: permission.ResourceUser,
	}
)

func disputeLabel(inv *invoice.Invoice) string {
	return "Dispute on " + invoiceLabel(inv)
}

func disputePreview(
	plan *serviceports.InvoiceDisputePreview,
	summary string,
) (*agent.ToolPreview, error) {
	record := toolpreview.Record{
		Resource: permission.ResourceInvoiceDispute,
		Label:    disputeLabel(plan.InvoiceBefore),
	}
	opts := []toolpreview.Option{
		toolpreview.Only(disputeFields...),
		toolpreview.WithRefs(disputeRefs),
		toolpreview.Volatile(fieldResolvedAt),
	}

	var (
		change *agent.RecordChange
		err    error
	)
	if plan.Before == nil {
		change, err = toolpreview.Create(record, plan.After, opts...)
	} else {
		record.ID = plan.Before.ID
		record.Version = pinnedVersion(plan.Before.Version)
		change, err = toolpreview.Changed(record, plan.Before, plan.After, opts...)
	}
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		plan.InvoiceBefore.CurrencyCode,
		agent.MoneyLine{Label: "Disputed", After: knownAmount(plan.After.DisputedAmount)},
	), toolpreview.SensitiveAs(paramDisputedAmount))

	changes := []*agent.RecordChange{change}
	if plan.InvoiceBefore.DisputeStatus != plan.InvoiceAfter.DisputeStatus {
		flag, flagErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceInvoice,
			ID:       plan.InvoiceBefore.ID,
			Label:    invoiceLabel(plan.InvoiceBefore),
			Version:  pinnedVersion(plan.InvoiceBefore.Version),
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
		invoiceLabel(plan.InvoiceBefore),
	), changes...), nil
}
