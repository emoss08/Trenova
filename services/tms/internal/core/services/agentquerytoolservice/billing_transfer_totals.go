package agentquerytoolservice

import (
	"cmp"
	"slices"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

const (
	candidateTotalsEveryMatch = "every matching shipment"
	candidateTotalsThisPage   = "only the shipments on this page; later pages are not counted"
)

type candidateTotal struct {
	Currency      string `json:"currency"`
	Count         int    `json:"count"`
	Amount        string `json:"amount"`
	WithoutCharge int    `json:"withoutCharge,omitempty"`
}

type candidateTotals struct {
	Covers               string           `json:"covers"`
	Transfer             []candidateTotal `json:"transfer"`
	MarkReadyAndTransfer []candidateTotal `json:"markReadyAndTransfer"`
	Refused              []candidateTotal `json:"refused"`
}

type currencyTally struct {
	count         int
	amount        decimal.Decimal
	withoutCharge int
}

type outcomeTally map[string]*currencyTally

func (t outcomeTally) add(decision *serviceports.BillingTransferDecision) {
	currency := cmp.Or(decision.CurrencyCode, money.DefaultCurrencyCode)
	tally, ok := t[currency]
	if !ok {
		tally = &currencyTally{amount: decimal.Zero}
		t[currency] = tally
	}
	tally.count++
	if decision.TotalCharge.Valid {
		tally.amount = tally.amount.Add(decision.TotalCharge.Decimal)
	} else {
		tally.withoutCharge++
	}
}

func (t outcomeTally) totals() []candidateTotal {
	totals := make([]candidateTotal, 0, len(t))
	for currency, tally := range t {
		totals = append(totals, candidateTotal{
			Currency:      currency,
			Count:         tally.count,
			Amount:        tally.amount.StringFixed(2),
			WithoutCharge: tally.withoutCharge,
		})
	}
	slices.SortFunc(totals, func(a, b candidateTotal) int {
		return cmp.Compare(a.Currency, b.Currency)
	})

	return totals
}

func candidateTotalsOf(
	decisions []serviceports.BillingTransferDecision,
	everyMatch bool,
) candidateTotals {
	transfer := outcomeTally{}
	markReady := outcomeTally{}
	refused := outcomeTally{}
	for idx := range decisions {
		decision := &decisions[idx]
		switch decision.Outcome {
		case serviceports.BillingTransferOutcomeTransfer:
			transfer.add(decision)
		case serviceports.BillingTransferOutcomeMarkReadyAndTransfer:
			markReady.add(decision)
		case serviceports.BillingTransferOutcomeRefused:
			refused.add(decision)
		case serviceports.BillingTransferOutcomeReturnToOperations:
		}
	}

	covers := candidateTotalsThisPage
	if everyMatch {
		covers = candidateTotalsEveryMatch
	}

	return candidateTotals{
		Covers:               covers,
		Transfer:             transfer.totals(),
		MarkReadyAndTransfer: markReady.totals(),
		Refused:              refused.totals(),
	}
}
