package invoicepreview

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
)

const moneyLineCredit = "Credit remaining"

func InvoiceBalanceChanges(before, after []*invoice.Invoice) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, len(after))
	for idx, invAfter := range after {
		if invAfter == nil || idx >= len(before) || before[idx] == nil {
			continue
		}
		change, err := InvoicePaymentChange(before[idx], invAfter)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
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
			Version:  changepreview.PinnedVersion(before.Version),
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
		Before: changepreview.KnownAmount(money.DecimalFromMinor(before.CreditRemainingMinor())),
		After:  changepreview.KnownAmount(money.DecimalFromMinor(after.CreditRemainingMinor())),
	}), toolpreview.SensitiveAs("totalAmount", "appliedAmount"))

	return change, nil
}

func creditMemoLabel(memo *invoice.Invoice) string {
	return "Credit memo " + memo.Number
}

var creditApplicationRefs = map[string]permission.Resource{
	"creditMemoInvoiceId":     permission.ResourceInvoice,
	toolparams.ParamInvoiceID: permission.ResourceInvoice,
	"unappliedById":           permission.ResourceUser,
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
			toolpreview.Only(
				changepreview.FieldStatus,
				"unappliedReason",
				"unappliedById",
				"unappliedAt",
			),
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
		toolpreview.Only(
			changepreview.FieldStatus,
			"creditMemoInvoiceId",
			toolparams.ParamInvoiceID,
			toolparams.ParamAccountingDate,
		),
		toolpreview.WithRefs(creditApplicationRefs),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			toolparams.ParamAccountingDate: assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		plan.CreditMemoBefore.CurrencyCode,
		agent.MoneyLine{
			Label: "Applied",
			After: changepreview.KnownAmount(
				money.DecimalFromMinor(application.AppliedAmountMinor),
			),
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
	invoices, err := InvoiceBalanceChanges(plan.InvoicesBefore, plan.InvoicesAfter)
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

func RenderApplyCredit(
	req *customerpaymentports.ApplyCreditMemoRequest,
	plan *customerpaymentports.CreditMemoApplicationPreview,
) (*agent.ToolPreview, error) {
	currency := plan.CreditMemoBefore.CurrencyCode
	return creditPreview(plan, fmt.Sprintf(
		"Would apply %s of credit memo %s to %s, leaving %s of credit.",
		money.FormatMinor(creditAppliedMinor(plan), currency),
		plan.CreditMemoBefore.Number,
		changepreview.CountOf(len(req.Applications), "invoice"),
		money.FormatMinor(plan.CreditMemoAfter.CreditRemainingMinor(), currency),
	))
}

func RenderUnapplyCredit(
	_ *customerpaymentports.UnapplyCreditMemoApplicationRequest,
	plan *customerpaymentports.CreditMemoApplicationPreview,
) (*agent.ToolPreview, error) {
	target := "the invoice"
	if len(plan.InvoicesBefore) > 0 && plan.InvoicesBefore[0] != nil {
		target = InvoiceLabel(plan.InvoicesBefore[0])
	}

	return creditPreview(plan, fmt.Sprintf(
		"Would take back %s of credit memo %s from %s, reopening its balance by that much.",
		money.FormatMinor(creditAppliedMinor(plan), plan.CreditMemoBefore.CurrencyCode),
		plan.CreditMemoBefore.Number,
		target,
	))
}
