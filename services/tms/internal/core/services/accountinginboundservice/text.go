package accountinginboundservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
)

func paymentName(p *postingPlan) string {
	noun := "Payment"
	if p.change.Kind == accountingsync.InboundBillPayment {
		noun = "Bill payment"
	}
	if p.change.ExternalNumber != "" {
		return noun + " " + p.change.ExternalNumber
	}
	return noun
}

func dayOf(ts int64) string {
	return timeutils.FormatCalendarDate(ts, nil)
}

func sentFromTrenovaText(p *postingPlan) string {
	return paymentName(p) + " was sent from Trenova, so there is nothing to bring back."
}

func notTrenovaText(p *postingPlan) string {
	return paymentName(p) + " pays nothing Trenova sent to " + p.provider + "."
}

func unknownDocumentText(p *postingPlan, count int) string {
	documents := "1 document"
	if count != 1 {
		documents = strconv.Itoa(count) + " documents"
	}
	return paymentName(p) + " also pays " + documents + " Trenova did not send. " +
		"Record them in Trenova or remove them from the payment in " + p.provider +
		", or ignore this payment and record it by hand."
}

func partyMismatchText(p *postingPlan) string {
	return paymentName(p) + " pays documents that belong to more than one Trenova party. " +
		"Split it in " + p.provider + " or record it by hand."
}

func currencyMismatchText(p *postingPlan, number, currency string) string {
	return paymentName(p) + " is in " + p.change.CurrencyCode + ", but " + number +
		" is in " + strings.ToUpper(currency) + "."
}

func notOpenText(number string, status invoice.Status) string {
	if status == invoice.StatusVoided {
		return number + " was voided in Trenova, so nothing is owed on it."
	}
	return number + " is not posted in Trenova, so nothing can be paid on it yet."
}

func alreadyPaidText(number string) string {
	return number + " is already paid in Trenova. If both payments are real, record the " +
		"second as unapplied cash by hand; otherwise remove one of them."
}

func settlementVoidedText(number string) string {
	return number + " was voided in Trenova, so nothing is owed on it."
}

func overpaymentText(number string, amount, open int64, currency string) string {
	return "Pays " + money.FormatMinor(amount, currency) + " on " + number +
		", which has " + money.FormatMinor(open, currency) + " open in Trenova."
}

func creditOverText(number string, amount, remaining int64, currency string) string {
	return "Uses " + money.FormatMinor(amount, currency) + " of credit memo " + number +
		", which has " + money.FormatMinor(remaining, currency) + " left in Trenova."
}

func totalOverText(applied, amount int64, currency string) string {
	return "Applies " + money.FormatMinor(applied, currency) + " in cash but totals only " +
		money.FormatMinor(amount, currency) + "."
}

func partialBillText(number string, paid, net int64, currency string) string {
	return "Pays " + money.FormatMinor(paid, currency) + " of settlement " + number +
		", whose net is " + money.FormatMinor(net, currency) +
		". Trenova settlements are paid in full, so record a partial payment by hand."
}

func noPeriodText(p *postingPlan) string {
	return "No fiscal period in Trenova covers " + dayOf(p.paidAt) +
		". Add the period, then apply the payment."
}

func periodNotOpenText(p *postingPlan, status fiscalperiod.Status) string {
	return "Dated " + dayOf(p.paidAt) + ", in a fiscal period that is " +
		strings.ToLower(string(status)) + ". Reopen the period, then apply the payment."
}

func proposeText(p *postingPlan) string {
	return paymentName(p) + " was recorded in " + p.provider + " on " + dayOf(p.paidAt) +
		". Apply it to post it in Trenova, or ignore it if it was entered here too."
}

func appliedText(p *postingPlan) string {
	return paymentName(p) + " was recorded in " + p.provider + " and applied in Trenova."
}

func linkedText(p *postingPlan) string {
	return "Recorded in " + p.provider + " and applied in Trenova, so nothing is sent."
}
