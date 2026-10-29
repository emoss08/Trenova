package invoiceadjustmentengine

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/shopspring/decimal"
)

type previewAccumulator struct {
	lines            []*invoiceadjustment.InvoiceAdjustmentLine
	creditLines      []*invoice.InvoiceLine
	replacementLines []*invoice.InvoiceLine
}

func collectPreviewLines(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	usageByLine map[string]decimal.Decimal,
) *previewAccumulator {
	acc := &previewAccumulator{
		lines:            make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(entity.Lines)),
		creditLines:      make([]*invoice.InvoiceLine, 0, len(entity.Lines)),
		replacementLines: make([]*invoice.InvoiceLine, 0, len(entity.Lines)),
	}
	requestedByLine := requestedLinesByID(req)
	fullScope := len(req.Lines) == 0 || req.Kind == invoiceadjustment.KindFullReversal
	for _, sourceLine := range entity.Lines {
		if sourceLine == nil {
			continue
		}
		input, hasInput := requestedByLine[sourceLine.ID.String()]
		if !fullScope && !hasInput {
			continue
		}
		acc.add(preview, &previewLineRequest{
			req:        req,
			entity:     entity,
			sourceLine: sourceLine,
			input:      input,
			used:       usageByLine[sourceLine.ID.String()],
			fullScope:  fullScope,
		})
	}

	return acc
}

type previewLineRequest struct {
	req        *invoiceadjustmentports.InvoiceAdjustmentRequest
	entity     *invoice.Invoice
	sourceLine *invoice.InvoiceLine
	input      *invoiceadjustmentports.InvoiceAdjustmentLineInput
	used       decimal.Decimal
	fullScope  bool
}

func (a *previewAccumulator) add(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	p *previewLineRequest,
) {
	sourceLine := p.sourceLine
	eligible := sourceLine.Amount.Abs().Sub(p.used)
	if eligible.IsNegative() {
		eligible = decimal.Zero
	}

	values := resolvePreviewLineValues(previewLineValuesRequest{
		sourceLine: sourceLine,
		input:      p.input,
		kind:       p.req.Kind,
		eligible:   eligible,
	})
	if p.req.Kind == invoiceadjustment.KindCreditRebill && p.fullScope &&
		p.req.RebillStrategy != invoiceadjustment.RebillStrategyRerate {
		values.rebillAmount = sourceLine.Amount.Abs()
		values.rebillQuantity = sourceLine.Quantity
	}

	linePreview := previewLineFor(preview, sourceLine, eligible, p.used, &values)
	preview.Lines = append(preview.Lines, linePreview)
	preview.CreditTotalAmount = preview.CreditTotalAmount.Add(values.creditAmount)
	preview.RebillTotalAmount = preview.RebillTotalAmount.Add(values.rebillAmount)

	a.lines = append(a.lines, &invoiceadjustment.InvoiceAdjustmentLine{
		OriginalInvoiceID:       p.entity.ID,
		OriginalLineID:          sourceLine.ID,
		LineNumber:              sourceLine.LineNumber,
		Description:             values.description,
		CreditQuantity:          values.creditQuantity,
		CreditAmount:            values.creditAmount,
		RemainingEligibleAmount: linePreview.RemainingEligibleAmount,
		RebillQuantity:          values.rebillQuantity,
		RebillAmount:            values.rebillAmount,
		ReplacementPayload:      values.payload,
	})

	if values.creditAmount.GreaterThan(decimal.Zero) {
		a.creditLines = append(a.creditLines, creditLineFor(sourceLine, &values))
	}
	if values.rebillAmount.GreaterThan(decimal.Zero) {
		a.replacementLines = append(
			a.replacementLines,
			replacementLineFor(sourceLine, &values, len(a.replacementLines)+1),
		)
	}
}

func previewLineFor(
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	sourceLine *invoice.InvoiceLine,
	eligible, used decimal.Decimal,
	values *previewLineValues,
) *invoiceadjustmentports.InvoiceAdjustmentPreviewLine {
	hasEligibilityError := values.creditAmount.GreaterThan(eligible)
	eligibilityOverageAmount := decimal.Zero
	eligibilityMessage := ""
	if hasEligibilityError {
		eligibilityOverageAmount = values.creditAmount.Sub(eligible)
		eligibilityMessage = fmt.Sprintf(
			"Line %d exceeds the remaining eligible amount by %s",
			sourceLine.LineNumber,
			eligibilityOverageAmount.StringFixed(4),
		)
		appendPreviewError(preview, "lines", eligibilityMessage)
	}

	return &invoiceadjustmentports.InvoiceAdjustmentPreviewLine{
		LineNumber:               sourceLine.LineNumber,
		OriginalLineID:           sourceLine.ID,
		Description:              sourceLine.Description,
		EligibleAmount:           eligible,
		AlreadyCreditedAmount:    used,
		RequestedCreditAmount:    values.creditAmount,
		RequestedRebillAmount:    values.rebillAmount,
		RemainingEligibleAmount:  eligible.Sub(values.creditAmount),
		HasEligibilityError:      hasEligibilityError,
		EligibilityOverageAmount: eligibilityOverageAmount,
		EligibilityMessage:       eligibilityMessage,
	}
}

func creditLineFor(
	sourceLine *invoice.InvoiceLine,
	values *previewLineValues,
) *invoice.InvoiceLine {
	unitPrice := values.creditAmount
	if values.creditQuantity.GreaterThan(decimal.Zero) {
		unitPrice = values.creditAmount.Div(values.creditQuantity)
	}
	line := &invoice.InvoiceLine{
		LineNumber:  sourceLine.LineNumber,
		Type:        sourceLine.Type,
		Description: values.description,
		Quantity:    values.creditQuantity,
		UnitPrice:   unitPrice.Neg(),
		Amount:      values.creditAmount.Neg(),
	}
	line.CopyChargeDetail(sourceLine)

	return line
}

func replacementLineFor(
	sourceLine *invoice.InvoiceLine,
	values *previewLineValues,
	lineNumber int,
) *invoice.InvoiceLine {
	unitPrice := values.rebillAmount
	if values.rebillQuantity.GreaterThan(decimal.Zero) {
		unitPrice = values.rebillAmount.Div(values.rebillQuantity)
	}
	line := &invoice.InvoiceLine{
		LineNumber:  lineNumber,
		Type:        sourceLine.Type,
		Description: values.description,
		Quantity:    decimalutils.Max(values.rebillQuantity, decimal.NewFromInt(1)),
		UnitPrice:   unitPrice,
		Amount:      values.rebillAmount,
	}
	line.CopyChargeDetail(sourceLine)

	return line
}
