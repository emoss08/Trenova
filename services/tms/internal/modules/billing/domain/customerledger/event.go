package customerledger

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
)

// EventPaymentApplied is the source event of a ledger row that applies cash
// already on account to an invoice.
const EventPaymentApplied = "CustomerPaymentApplied"

var eventLabels = map[string]string{
	tenant.JournalSourceEventInvoicePosted.String():         "Invoice",
	tenant.JournalSourceEventCreditMemoPosted.String():      "Credit memo",
	tenant.JournalSourceEventDebitMemoPosted.String():       "Debit memo",
	tenant.JournalSourceEventCustomerPaymentPosted.String(): "Payment",
	EventPaymentApplied: "Payment applied",
	tenant.JournalSourceEventCustomerShortPayRecognized.String(): "Short payment written off",
	tenant.JournalSourceEventCustomerPaymentReversed.String():    "Payment reversed",
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
