package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
)

type draftDeliveryView struct {
	Memo                   string `json:"memo"`
	RemittanceInstructions string `json:"remittanceInstructions"`
	EmailSubject           string `json:"emailSubject"`
	EmailBody              string `json:"emailBody"`
	EmailTo                string `json:"emailTo"`
	EmailCc                string `json:"emailCc"`
	EmailBcc               string `json:"emailBcc"`
	Attachments            int    `json:"attachments"`
}

var draftDeliveryLabels = map[string]string{
	paramInvoiceMemo:  "Memo",
	paramRemittance:   "Remittance instructions",
	paramEmailSubject: "Email subject",
	paramEmailBody:    "Email body",
	paramEmailTo:      "Emailed to",
	paramEmailCc:      "Copied",
	paramEmailBcc:     "Blind-copied",
	"attachments":     "Attached documents",
}

func draftDeliveryOf(entity *invoice.Invoice, attachments int) *draftDeliveryView {
	return &draftDeliveryView{
		Memo:                   entity.Memo,
		RemittanceInstructions: entity.RemittanceInstructions,
		EmailSubject:           entity.EmailSubjectSnapshot,
		EmailBody:              entity.EmailBodySnapshot,
		EmailTo:                strings.Join(entity.EmailToSnapshot, ", "),
		EmailCc:                strings.Join(entity.EmailCCSnapshot, ", "),
		EmailBcc:               strings.Join(entity.EmailBCCSnapshot, ", "),
		Attachments:            attachments,
	}
}

func invoiceRecord(entity *invoice.Invoice) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInvoice,
		ID:       entity.ID,
		Label:    invoiceLabel(entity),
		Version:  pinnedVersion(entity.Version),
	}
}

func renderDraftUpdate(
	req *serviceports.UpdateInvoiceDraftRequest,
	plan *serviceports.InvoiceDraftUpdatePreview,
) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(
		invoiceRecord(plan.Before),
		draftDeliveryOf(plan.Before, len(plan.AttachmentsBefore)),
		draftDeliveryOf(plan.After, len(plan.AttachmentsAfter)),
		toolpreview.Labels(draftDeliveryLabels),
	)
	if err != nil {
		return nil, err
	}

	named := make([]string, 0, len(change.Fields))
	for idx := range change.Fields {
		if label, ok := draftDeliveryLabels[change.Fields[idx].Path]; ok {
			named = append(named, strings.ToLower(label))
		}
	}
	summary := "Would update draft " + invoiceLabel(plan.Before)
	if len(named) == 0 {
		summary += ", which already reads that way"
	} else {
		summary += ": " + strings.Join(named, ", ")
	}
	if req.AttachmentIDs != nil {
		summary += fmt.Sprintf("; it will carry %s",
			countOf(len(plan.AttachmentsAfter), "attachment"))
	}

	return toolpreview.Build(summary+".", change), nil
}

func renderPDFGeneration(
	_ *serviceports.InvoicePreviewRequest,
	plan *serviceports.InvoicePDFGenerationPlan,
) (*agent.ToolPreview, error) {
	summary := fmt.Sprintf("Would render the PDF of %s from what it says now and file it on "+
		"the invoice", invoiceLabel(plan.Invoice))
	if plan.Invoice.PDFDocumentID.IsNotNil() {
		summary += ", replacing the one there"
	}
	preview := toolpreview.Build(summary + "; nothing is sent.")
	preview.Partial = true

	return preview, nil
}

func renderCreateInvoices(
	_ *createInvoicesRequest,
	plan *serviceports.CreateInvoicesPlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Drafts))
	payers := make([]string, 0, len(plan.Drafts))
	for _, draft := range plan.Drafts {
		change, err := draftInvoiceChange(draft)
		if err != nil {
			return nil, err
		}
		change.Label = "Draft invoice to " + draft.BillToName
		changes = append(changes, change)
		payer := fmt.Sprintf("%s for %s", draft.BillToName,
			money.FormatMinor(draft.TotalAmountMinor, draft.CurrencyCode))
		if draft.IsSplitBill {
			payer += " (their share of a split bill)"
		}
		payers = append(payers, payer)
	}

	summary := fmt.Sprintf("Would create %s: %s. They stay drafts until post_invoice posts "+
		"them.", countOf(len(plan.Drafts), "draft invoice"), strings.Join(payers, "; "))
	preview := toolpreview.Build(summary, changes...)
	preview.Partial = true

	return preview, nil
}

func renderMemo(
	req *serviceports.CreateMemoRequest,
	memo *invoice.Invoice,
) (*agent.ToolPreview, error) {
	change, err := draftInvoiceChange(memo)
	if err != nil {
		return nil, err
	}
	change.Label = fmt.Sprintf("Draft %s to %s", memo.BillType, memo.BillToName)

	summary := fmt.Sprintf("Would raise a draft %s to %s for %s with %s",
		memo.BillType, memo.BillToName,
		money.FormatMinor(memo.TotalAmountMinor, memo.CurrencyCode),
		countOf(len(req.Lines), "line"))
	if memo.ReferenceInvoice != nil {
		summary += ", against " + invoiceLabel(memo.ReferenceInvoice)
	}

	return toolpreview.Build(summary+"; it stays a draft until post_invoice posts it.", change), nil
}

func dispositionSentence(disposition invoice.VoidDisposition) string {
	if disposition == invoice.VoidDispositionRebill {
		return "its freight goes back to the billing queue to be billed again"
	}

	return "its freight is canceled and never billed"
}

func renderVoid(
	_ *serviceports.VoidInvoiceRequest,
	plan *serviceports.InvoiceVoidPreview,
) (*agent.ToolPreview, error) {
	label := invoiceLabel(plan.Before)
	fields := []string{fieldVoidReason, "voidDisposition"}
	if !plan.Posted {
		fields = append(fields, fieldStatus, "voidedAt", "voidedById")
	}
	invoiceChange, err := toolpreview.Changed(invoiceRecord(plan.Before), plan.Before, plan.After,
		toolpreview.Only(fields...),
		toolpreview.Volatile("voidedAt"),
		toolpreview.WithRefs(map[string]permission.Resource{"voidedById": permission.ResourceUser}),
	)
	if err != nil {
		return nil, err
	}
	disposition := dispositionSentence(plan.After.VoidDisposition)

	if !plan.Posted {
		return toolpreview.Build(fmt.Sprintf("Would void draft %s; %s.", label, disposition),
			invoiceChange), nil
	}

	figures := plan.Reversal
	reversal, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceInvoice,
			Label:    adjustmentLabel(figures.Kind, plan.Before.Number),
		},
		&adjustmentView{
			InvoiceID: plan.Before.ID,
			Kind:      string(figures.Kind),
			Reason:    plan.After.VoidReason,
			Status:    string(outcomeStatus(figures)),
		},
		toolpreview.WithRefs(adjustmentRefs),
	)
	if err != nil {
		return nil, err
	}
	adjustmentMoney(reversal, figures, plan.Before.CurrencyCode)

	preview := toolpreview.Build(fmt.Sprintf(
		"Would void posted %s through a full reversal crediting %s, which %s; the invoice "+
			"voids when it executes and %s.%s",
		label, figures.CreditTotalAmount.Abs().StringFixed(2), outcomeSentence(figures),
		disposition, figureWarnings(figures),
	), reversal, invoiceChange)
	preview.Partial = true
	if refusal := figuresRefusal(
		[]*serviceports.InvoiceAdjustmentPreview{figures},
	); refusal != nil {
		return warnWouldFail(preview, refusal), nil
	}

	return preview, nil
}

func renderEDISend(
	req *serviceports.SendInvoiceEDIRequest,
	plan *serviceports.InvoiceEDISendPreview,
) (*agent.ToolPreview, error) {
	label := invoiceLabel(plan.Invoice)
	partner := plan.Plan.PartnerName
	summary := fmt.Sprintf("Would send %s to %s as an EDI 210", label, partner)
	if method := strings.TrimSpace(plan.Plan.CommunicationMethod); method != "" {
		summary += " over " + method
	}
	if req.Force && plan.Invoice.EDISendStatus != invoice.EDISendStatusNotSent {
		summary += "; it was sent before, and this sends it again"
	}

	change := toolpreview.Send(invoiceRecord(plan.Invoice), &agent.MessagePreview{
		Channel: agent.MessageChannelEDI,
		To:      []string{partner},
		Subject: "EDI 210 for " + label,
		Body: fmt.Sprintf("%s billed to %s for %s", label, plan.Invoice.BillToName,
			money.FormatMinor(plan.Invoice.TotalAmountMinor, plan.Invoice.CurrencyCode)),
	})

	return toolpreview.Build(summary+".", change), nil
}
