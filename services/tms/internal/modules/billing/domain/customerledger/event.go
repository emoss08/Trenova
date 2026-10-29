package customerledger

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
)

// EventPaymentApplied is the source event of a ledger row that applies cash
// already on account to an invoice.
const EventPaymentApplied = "CustomerPaymentApplied"

// SourceFactoringItem is the source object of a ledger row a factoring
// movement wrote when it settled or reopened an invoice.
const SourceFactoringItem = "FactoringSubmissionItem"

// CollectionWindowDays is the trailing window the receivables dashboard
// measures collection performance over.
const CollectionWindowDays = 91

var eventLabels = map[string]string{
	accountingcontrol.JournalSourceEventInvoicePosted.String():         "Invoice",
	accountingcontrol.JournalSourceEventCreditMemoPosted.String():      "Credit memo",
	accountingcontrol.JournalSourceEventDebitMemoPosted.String():       "Debit memo",
	accountingcontrol.JournalSourceEventCustomerPaymentPosted.String(): "Payment",
	EventPaymentApplied: "Payment applied",
	accountingcontrol.JournalSourceEventCustomerShortPayRecognized.String(): "Short payment written off",
	accountingcontrol.JournalSourceEventCustomerPaymentReversed.String():    "Payment reversed",
}

// EventLabel is how a statement names the event that moved the balance. An
// event without a label of its own is spelled out from its name, so a new
// event never reaches a customer as a code.
func EventLabel(eventType string) string {
	if label, ok := eventLabels[eventType]; ok {
		return label
	}
	return spellOut(eventType)
}

func spellOut(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name) + 4)
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(runes[i-1]) {
			b.WriteByte(' ')
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
