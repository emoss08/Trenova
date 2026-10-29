package xerodocsync

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeropay"
)

const (
	DocTypePayment      = "Payment"
	DocTypeBatchPayment = "BatchPayment"
	RefPaymentPrefix    = "payment:"
	RefReplaced         = "replaced"
	StepReplaceDelete   = 1
	StepReplaceCreate   = 2
)

func FinishPayment(
	result *accountingconnectorports.AccountingDocumentResult,
	payment *xeropay.Payment,
) {
	result.ExternalID = payment.PaymentID
	result.DocNumber = payment.Reference
	result.Refs[accountingsync.ExternalRefDocument] = payment.PaymentID
	result.Refs[accountingsync.ExternalRefDocumentType] = DocTypePayment
	result.Refs[RefReplaced] = ""
	if payment.InvoiceID != "" {
		result.Refs[RefPaymentPrefix+strings.ToLower(payment.InvoiceID)] = payment.PaymentID
	}
}

func PaymentReference(number, requestID string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r < ' ' || r == 0x7f {
			return ' '
		}
		return r
	}, number)
	cleaned = stringutils.CollapseWhitespace(cleaned)
	id := strings.TrimSpace(requestID)
	if id == "" {
		return Reference(cleaned)
	}
	budget := xerodocs.MaxReferenceLength - len([]rune(id)) - 1
	if budget <= 0 {
		return stringutils.TruncateRunes(id, xerodocs.MaxReferenceLength)
	}
	return strings.TrimSpace(
		strings.TrimSpace(stringutils.TruncateRunes(cleaned, budget)) + " " + id,
	)
}
