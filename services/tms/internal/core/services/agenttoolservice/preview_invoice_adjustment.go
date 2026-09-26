package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type adjustmentView struct {
	InvoiceID      pulid.ID `json:"invoiceId"`
	Kind           string   `json:"kind"`
	RebillStrategy string   `json:"rebillStrategy"`
	Reason         string   `json:"reason"`
	Status         string   `json:"status"`
	ApprovalStatus string   `json:"approvalStatus,omitempty"`
	LineCount      int      `json:"lineCount"`
}

var adjustmentRefs = map[string]permission.Resource{paramInvoiceID: permission.ResourceInvoice}

func adjustmentInvoiceLabel(number string) string {
	if strings.TrimSpace(number) == "" {
		return "the invoice"
	}

	return "Invoice " + number
}

func adjustmentLabel(kind invoiceadjustment.Kind, invoiceNumber string) string {
	return string(kind) + " adjustment of " + adjustmentInvoiceLabel(invoiceNumber)
}

func adjustmentMoney(
	change *agent.RecordChange,
	figures *serviceports.InvoiceAdjustmentPreview,
	currency string,
) {
	if figures == nil {
		return
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		currency,
		agent.MoneyLine{Label: "Credited", After: knownAmount(figures.CreditTotalAmount.Abs())},
		agent.MoneyLine{Label: "Rebilled", After: knownAmount(figures.RebillTotalAmount)},
	), toolpreview.SensitiveAs("totalAmount"))
}

func outcomeStatus(figures *serviceports.InvoiceAdjustmentPreview) invoiceadjustment.Status {
	if figures != nil && figures.RequiresApproval {
		return invoiceadjustment.StatusPendingApproval
	}

	return invoiceadjustment.StatusExecuted
}

func outcomeSentence(figures *serviceports.InvoiceAdjustmentPreview) string {
	if outcomeStatus(figures) == invoiceadjustment.StatusPendingApproval {
		return "waits for approval"
	}

	return "executes at once"
}

func figureWarnings(figures *serviceports.InvoiceAdjustmentPreview) string {
	if figures == nil || len(figures.Warnings) == 0 {
		return ""
	}

	return " Policy notes: " + strings.Join(figures.Warnings, "; ") + "."
}

func renderSubmission(req *adjustmentSubmission, plan *submissionPlan) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.figures))
	credited := decimal.Zero
	outcomes := make(map[string]int, 2)
	notes := ""
	for idx, figures := range plan.figures {
		view := &adjustmentView{
			InvoiceID:      figures.InvoiceID,
			Kind:           string(figures.Kind),
			RebillStrategy: string(figures.RebillStrategy),
			Status:         string(outcomeStatus(figures)),
		}
		switch {
		case plan.draft != nil:
			view.Reason = plan.draft.Reason
			view.LineCount = len(plan.draft.Lines)
		case idx < len(req.items):
			view.Reason = req.items[idx].Reason
			view.LineCount = len(req.items[idx].Lines)
		}
		change, err := toolpreview.Create(
			toolpreview.Record{
				Resource: permission.ResourceInvoice,
				Label:    adjustmentLabel(figures.Kind, figures.InvoiceNumber),
			},
			view,
			toolpreview.WithRefs(adjustmentRefs),
		)
		if err != nil {
			return nil, err
		}
		adjustmentMoney(change, figures, figures.CurrencyCode)
		changes = append(changes, change)
		credited = credited.Add(figures.CreditTotalAmount.Abs())
		outcomes[outcomeSentence(figures)]++
		notes += figureWarnings(figures)
	}

	subject := countOf(len(plan.figures), "invoice adjustment")
	if plan.draft != nil {
		subject = "the draft " + adjustmentLabel(plan.draft.Kind, plan.figures[0].InvoiceNumber)
	}
	summary := fmt.Sprintf("Would submit %s crediting %s in all; %s.",
		subject, credited.StringFixed(2), describeOutcomes(outcomes))
	preview := toolpreview.Build(summary+" Credit memos and any replacement invoices are made "+
		"when each executes."+notes, changes...)
	preview.Partial = true

	return preview, nil
}

func describeOutcomes(outcomes map[string]int) string {
	parts := make([]string, 0, len(outcomes))
	for _, sentence := range []string{"executes at once", "waits for approval"} {
		count, ok := outcomes[sentence]
		if !ok {
			continue
		}
		if len(outcomes) == 1 {
			parts = append(parts, map[bool]string{true: "it ", false: "each "}[count == 1]+
				sentence)
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s", count, sentence))
	}

	return strings.Join(parts, ", ")
}

func renderDraft(
	req *serviceports.SaveInvoiceAdjustmentDraftRequest,
	plan *serviceports.InvoiceAdjustmentDraftPreview,
) (*agent.ToolPreview, error) {
	after := &adjustmentView{
		InvoiceID:      plan.After.OriginalInvoiceID,
		Kind:           string(plan.After.Kind),
		RebillStrategy: string(plan.After.RebillStrategy),
		Reason:         plan.After.Reason,
		Status:         string(plan.After.Status),
		LineCount:      len(req.Lines),
	}
	record := toolpreview.Record{
		Resource: permission.ResourceInvoice,
		Label:    "Draft " + adjustmentLabel(plan.After.Kind, plan.Invoice.Number),
	}

	var (
		change *agent.RecordChange
		err    error
	)
	if plan.Before == nil {
		change, err = toolpreview.Create(record, after, toolpreview.WithRefs(adjustmentRefs))
	} else {
		before := &adjustmentView{
			InvoiceID:      plan.Before.OriginalInvoiceID,
			Kind:           string(plan.Before.Kind),
			RebillStrategy: string(plan.Before.RebillStrategy),
			Reason:         plan.Before.Reason,
			Status:         string(plan.Before.Status),
			LineCount:      len(plan.Before.Lines),
		}
		change, err = toolpreview.Changed(record, before, after,
			toolpreview.WithRefs(adjustmentRefs))
	}
	if err != nil {
		return nil, err
	}
	adjustmentMoney(change, plan.Figures, plan.Invoice.CurrencyCode)

	summary := fmt.Sprintf("Would save a draft %s adjustment of %s for a biller to submit; "+
		"nothing is credited until then.", plan.After.Kind, invoiceLabel(plan.Invoice))
	if refusal := figuresRefusal([]*serviceports.InvoiceAdjustmentPreview{plan.Figures}); refusal != nil {
		summary += " As it stands it could not be submitted: " + refusal.Error() + "."
	}

	return toolpreview.Build(summary+figureWarnings(plan.Figures), change), nil
}

func renderDecision(
	req *adjustmentDecision,
	plan *serviceports.InvoiceAdjustmentDecisionPreview,
) (*agent.ToolPreview, error) {
	adjustment := plan.Adjustment
	before := &adjustmentView{
		InvoiceID:      adjustment.OriginalInvoiceID,
		Kind:           string(adjustment.Kind),
		RebillStrategy: string(adjustment.RebillStrategy),
		Reason:         adjustment.Reason,
		Status:         string(adjustment.Status),
		ApprovalStatus: string(adjustment.ApprovalStatus),
		LineCount:      len(adjustment.Lines),
	}
	after := *before
	verb := "approve"
	figures := plan.Figures
	if req.approve {
		after.Status = string(invoiceadjustment.StatusExecuted)
		after.ApprovalStatus = string(invoiceadjustment.ApprovalStatusApproved)
	} else {
		verb = "reject"
		after.Status = string(invoiceadjustment.StatusRejected)
		after.ApprovalStatus = string(invoiceadjustment.ApprovalStatusRejected)
		figures = &serviceports.InvoiceAdjustmentPreview{
			CreditTotalAmount: adjustment.CreditTotalAmount,
			RebillTotalAmount: adjustment.RebillTotalAmount,
		}
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceInvoice,
		Label:    adjustmentLabel(adjustment.Kind, plan.Invoice.Number),
	}, before, &after, toolpreview.WithRefs(adjustmentRefs))
	if err != nil {
		return nil, err
	}
	adjustmentMoney(change, figures, plan.Invoice.CurrencyCode)

	summary := fmt.Sprintf("Would %s the %s adjustment of %s, crediting %s.",
		verb, adjustment.Kind, invoiceLabel(plan.Invoice),
		figures.CreditTotalAmount.Abs().StringFixed(2))
	if req.approve {
		summary = fmt.Sprintf("Would approve the %s adjustment of %s, which executes it: %s "+
			"credited and %s rebilled.", adjustment.Kind, invoiceLabel(plan.Invoice),
			figures.CreditTotalAmount.Abs().StringFixed(2),
			figures.RebillTotalAmount.StringFixed(2))
	}
	preview := toolpreview.Build(summary, change)
	preview.Partial = req.approve

	return preview, nil
}
