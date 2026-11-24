package bankreceipt

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	receiptIDPrefix      = "brcpt_"
	MaxCandidates        = 10
	MaxLookupNumbers     = 24
	NearAmountMinor      = 100
	PaymentWindowSeconds = 7 * 86400
	minTokenLength       = 3
	maxTokenLength       = 40
	nearDateSeconds      = 3 * 86400
	autoMatchScore       = 90
	autoMatchGap         = 15
	invoiceInReference   = 60
	invoiceInMemo        = 45
	invoiceAmountExact   = 25
	invoicePayer         = 5
)

type PaymentEvidence struct {
	ID          pulid.ID
	CustomerID  pulid.ID
	Reference   string
	AmountMinor int64
	PaymentDate int64
}

type InvoiceEvidence struct {
	ID         pulid.ID
	CustomerID pulid.ID
	Number     string
	OpenMinor  int64
}

type paymentFacts struct {
	refExact    bool
	refContains bool
	amountExact bool
	amountNear  bool
	dateNear    bool
}

func Score(
	receipt *BankReceipt,
	payments []PaymentEvidence,
	invoices []InvoiceEvidence,
) []*Candidate {
	if receipt == nil {
		return nil
	}
	out := make([]*Candidate, 0, len(payments)+len(invoices))
	payers := make(map[pulid.ID]bool, len(payments))
	seen := make(map[pulid.ID]bool, len(payments)+len(invoices))
	for idx := range payments {
		payment := &payments[idx]
		if seen[payment.ID] {
			continue
		}
		candidate := scorePayment(receipt, payment)
		if candidate == nil {
			continue
		}
		seen[payment.ID] = true
		payers[payment.CustomerID] = true
		out = append(out, candidate)
	}
	inReference := tokenKeys(receipt.ReferenceNumber)
	inMemo := tokenKeys(receipt.Memo)
	for idx := range invoices {
		invoice := &invoices[idx]
		if seen[invoice.ID] {
			continue
		}
		candidate := scoreInvoice(receipt, invoice, invoiceFacts{
			inReference: inReference,
			inMemo:      inMemo,
			payers:      payers,
		})
		if candidate == nil {
			continue
		}
		seen[invoice.ID] = true
		out = append(out, candidate)
	}

	return rank(receipt, out)
}

func scorePayment(receipt *BankReceipt, payment *PaymentEvidence) *Candidate {
	receiptRef := stringutils.NormalizeIdentifier(receipt.ReferenceNumber)
	paymentRef := stringutils.NormalizeIdentifier(payment.Reference)
	facts := paymentFacts{
		refExact: receiptRef != "" && receiptRef == paymentRef,
		refContains: receiptRef != "" && paymentRef != "" &&
			(strings.Contains(receiptRef, paymentRef) || strings.Contains(paymentRef, receiptRef)),
		amountExact: receipt.AmountMinor == payment.AmountMinor,
		amountNear: receipt.AmountMinor != payment.AmountMinor &&
			intutils.AbsDiff(receipt.AmountMinor, payment.AmountMinor) <= NearAmountMinor,
		dateNear: receipt.ReceiptDate > 0 && payment.PaymentDate > 0 &&
			intutils.AbsDiff(receipt.ReceiptDate, payment.PaymentDate) <= nearDateSeconds,
	}
	score, _ := paymentScore(facts)
	if score == 0 {
		return nil
	}
	evidence := make([]string, 0, 3)
	switch {
	case facts.refExact:
		evidence = append(evidence, string(EvidenceReferenceExact))
	case facts.refContains:
		evidence = append(evidence, string(EvidenceReferencePartial))
	}
	switch {
	case facts.amountExact:
		evidence = append(evidence, string(EvidenceAmountExact))
	case facts.amountNear:
		evidence = append(evidence, string(EvidenceAmountNear))
	}
	if facts.dateNear {
		evidence = append(evidence, string(EvidenceDateNear))
	}
	difference := receipt.AmountMinor - payment.AmountMinor

	return &Candidate{
		Kind:              CandidateCustomerPayment,
		CustomerPaymentID: payment.ID,
		CustomerID:        payment.CustomerID,
		Score:             score,
		Evidence:          evidence,
		Standing:          standingOf(difference, facts.refExact),
		OpenMinor:         payment.AmountMinor,
		DifferenceMinor:   difference,
	}
}

type invoiceFacts struct {
	inReference map[string]bool
	inMemo      map[string]bool
	payers      map[pulid.ID]bool
}

func scoreInvoice(receipt *BankReceipt, invoice *InvoiceEvidence, facts invoiceFacts) *Candidate {
	key := stringutils.NormalizeIdentifier(invoice.Number)
	if key == "" || invoice.OpenMinor <= 0 {
		return nil
	}
	evidence := make([]string, 0, 3)
	var score int
	named := facts.inReference[key]
	switch {
	case named:
		score = invoiceInReference
		evidence = append(evidence, string(EvidenceNumberInReference))
	case facts.inMemo[key]:
		score = invoiceInMemo
		evidence = append(evidence, string(EvidenceNumberInMemo))
	default:
		return nil
	}
	difference := receipt.AmountMinor - invoice.OpenMinor
	if difference == 0 {
		score += invoiceAmountExact
		evidence = append(evidence, string(EvidenceAmountExact))
	}
	if facts.payers[invoice.CustomerID] {
		score += invoicePayer
		evidence = append(evidence, string(EvidencePayer))
	}

	return &Candidate{
		Kind:            CandidateInvoice,
		InvoiceID:       invoice.ID,
		CustomerID:      invoice.CustomerID,
		Score:           score,
		Evidence:        evidence,
		Standing:        standingOf(difference, named),
		OpenMinor:       invoice.OpenMinor,
		DifferenceMinor: difference,
	}
}

func standingOf(difference int64, namedByReference bool) Standing {
	switch {
	case difference < 0:
		return StandingShort
	case difference > 0:
		return StandingOver
	case namedByReference:
		return StandingExact
	default:
		return StandingSameAmount
	}
}

func rank(receipt *BankReceipt, candidates []*Candidate) []*Candidate {
	slices.SortStableFunc(candidates, func(a, b *Candidate) int {
		if byScore := cmp.Compare(b.Score, a.Score); byScore != 0 {
			return byScore
		}
		if a.Kind != b.Kind {
			if a.Kind == CandidateCustomerPayment {
				return -1
			}

			return 1
		}

		return cmp.Compare(a.ItemID().String(), b.ItemID().String())
	})
	if len(candidates) > MaxCandidates {
		candidates = candidates[:MaxCandidates]
	}
	for idx, candidate := range candidates {
		candidate.OrganizationID = receipt.OrganizationID
		candidate.BusinessUnitID = receipt.BusinessUnitID
		candidate.BankReceiptID = receipt.ID
		candidate.Rank = idx + 1
	}

	return candidates
}

func paymentScore(f paymentFacts) (score int, reason string) {
	switch {
	case f.refExact && f.amountExact && f.dateNear:
		return 100, "Exact normalized reference, exact amount, and close date match"
	case f.refExact && f.amountExact:
		return 95, "Exact normalized reference and amount match"
	case f.refExact && f.amountNear:
		return 80, "Exact normalized reference and near amount match"
	case f.refContains && f.amountExact:
		return 75, "Partial normalized reference and exact amount match"
	case f.refExact:
		return 60, "Exact reference match"
	case f.refContains && f.amountNear:
		return 55, "Partial normalized reference and near amount match"
	case f.amountExact:
		return 40, "Exact amount match"
	case f.amountNear:
		return 20, "Near amount match"
	default:
		return 0, ""
	}
}

func AutoMatchPayment(candidates []*Candidate) (pulid.ID, bool) {
	payments := make([]*Candidate, 0, 2)
	for _, candidate := range candidates {
		if candidate != nil && candidate.Kind == CandidateCustomerPayment {
			payments = append(payments, candidate)
		}
		if len(payments) == 2 {
			break
		}
	}
	if len(payments) == 0 || payments[0].Score < autoMatchScore {
		return pulid.Nil, false
	}
	if len(payments) == 2 && payments[0].Score-payments[1].Score < autoMatchGap {
		return pulid.Nil, false
	}

	return payments[0].CustomerPaymentID, true
}

func IsReceiptID(id pulid.ID) bool {
	return strings.HasPrefix(id.String(), receiptIDPrefix)
}

func (r *BankReceipt) LookupNumbers() []string {
	out := make([]string, 0, MaxLookupNumbers)
	seen := make(map[string]bool, MaxLookupNumbers)
	for _, text := range []string{r.ReferenceNumber, r.Memo} {
		for _, token := range tokens(text) {
			upper := strings.ToUpper(token)
			for _, value := range []string{token, upper} {
				if seen[value] || len(out) == MaxLookupNumbers {
					continue
				}
				seen[value] = true
				out = append(out, value)
			}
		}
	}

	return out
}

func tokenKeys(text string) map[string]bool {
	keys := make(map[string]bool)
	for _, token := range tokens(text) {
		if key := stringutils.NormalizeIdentifier(token); key != "" {
			keys[key] = true
		}
	}

	return keys
}

func tokens(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		token := strings.Trim(field, "-_.")
		if len(token) < minTokenLength || len(token) > maxTokenLength {
			continue
		}
		if !strings.ContainsFunc(token, unicode.IsDigit) {
			continue
		}
		out = append(out, token)
	}

	return out
}
