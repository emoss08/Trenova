package invoicetools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

const fieldDisputeStatus = "disputeStatus"

var disputeFields = []string{
	common.FieldStatus,
	paramDisputeReasonCode,
	paramDisputedAmount,
	fieldDisputeNotes,
	paramDisputeResolution,
	paramResolutionNotes,
	common.FieldResolvedByID,
	common.FieldResolvedAt,
	common.FieldOpenedByID,
	common.ParamInvoiceID,
	common.FieldCustomerID,
}

var disputeRefs = map[string]permission.Resource{
	common.ParamInvoiceID:    permission.ResourceInvoice,
	common.FieldCustomerID:   permission.ResourceCustomer,
	common.FieldOpenedByID:   permission.ResourceUser,
	common.FieldResolvedByID: permission.ResourceUser,
}

func disputeLabel(inv *invoice.Invoice) string {
	return "Dispute on " + common.InvoiceLabel(inv)
}

func disputePreview(
	plan *invoiceports.InvoiceDisputePreview,
	summary string,
) (*agent.ToolPreview, error) {
	record := toolpreview.Record{
		Resource: permission.ResourceInvoiceDispute,
		Label:    disputeLabel(plan.InvoiceBefore),
	}
	opts := []toolpreview.Option{
		toolpreview.Only(disputeFields...),
		toolpreview.WithRefs(disputeRefs),
		toolpreview.Volatile(common.FieldResolvedAt),
	}

	var (
		change *agent.RecordChange
		err    error
	)
	if plan.Before == nil {
		change, err = toolpreview.Create(record, plan.After, opts...)
	} else {
		record.ID = plan.Before.ID
		record.Version = common.PinnedVersion(plan.Before.Version)
		change, err = toolpreview.Changed(record, plan.Before, plan.After, opts...)
	}
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		plan.InvoiceBefore.CurrencyCode,
		agent.MoneyLine{Label: "Disputed", After: common.KnownAmount(plan.After.DisputedAmount)},
	), toolpreview.SensitiveAs(paramDisputedAmount))

	changes := []*agent.RecordChange{change}
	if plan.InvoiceBefore.DisputeStatus != plan.InvoiceAfter.DisputeStatus {
		flag, flagErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceInvoice,
			ID:       plan.InvoiceBefore.ID,
			Label:    common.InvoiceLabel(plan.InvoiceBefore),
			Version:  common.PinnedVersion(plan.InvoiceBefore.Version),
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
		common.InvoiceLabel(plan.InvoiceBefore),
	), changes...), nil
}
