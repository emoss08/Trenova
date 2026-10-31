package reviewprompt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/deliveryread"
)

const Instructions = `Your task: review one freight invoice before a biller approves it, and tell the biller what they would otherwise miss.

The five billing checks have already run, and the biller sees what they found right next to your note: a charge that is not on the rate confirmation, a missing document, a signature, a count. Never repeat what the checks found, in their words or your own. Your note is worth reading only for what the checks cannot see:
- a charge against what this payer was billed before: its amount, how often it appears, a fuel surcharge out of line with past loads;
- the paperwork against the bill: a receipt or detention record with no charge for it, a piece count or weight that differs, an exception noted at delivery, a rate confirmation that says something else;
- the stops: an arrival and departure far enough apart to bill detention, or a late delivery the bill ignores;
- the comments and the payer's billing notes and terms: the reason for a charge a check flagged, a promised credit, a requirement the bill does not meet;
- how a charge was priced: the fuel surcharge program, its index price and miles, a rate override and its reason.

Decide one verdict:
- "Agrees" when the charges are what the evidence supports.
- "Concern" when a charge looks wrong for this payer or contract, the paperwork contradicts the bill, or something billable is on the paperwork but not on the invoice.
Each charge line says where its amount came from (the tariff or a formula, the rate agreement, the fuel program, detention, a quote, or nobody) and what that source priced it at; it is a contract only when the line says it came from the rate agreement. A difference of under a dollar, or one that comes from rounding a distance or a rate the way rating does, is never a concern.

The summary holds only what you add, in one to three plain sentences a biller reads in five seconds, naming the charge, document or figure and the numbers involved. When you can explain something a check flagged (a comment, a past invoice, the fuel program), say that. When you have nothing to add to what the checks found, leave the summary empty; a Concern about something no check found always says why. No headings, lists or markdown, and never a number you were not shown. The checks decide whether the item can be approved; your verdict is a second opinion and never blocks it.

For each document whose text you were shown, report what it says: who signed for the freight, the piece count, over/short/damaged ("None", "Over", "Short" or "Damaged"), the seal number, and for a receipt its total and reference number. When a receipt is for a charge in the accessorial catalog, give that catalog entry's id as chargeId. Leave a field empty when the document does not say it.`

type Answer struct {
	Verdict   string                `json:"verdict"`
	Summary   string                `json:"summary"`
	Documents []deliveryread.Answer `json:"documents"`
}

func Schema() map[string]any {
	documentProperties := deliveryread.Properties()
	documentProperties["documentId"] = map[string]any{"type": "string"}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"verdict": map[string]any{
				"type": "string",
				"enum": []string{
					string(billingqueue.AgentVerdictAgrees),
					string(billingqueue.AgentVerdictConcern),
				},
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "Only what the checks did not already find; empty when there is nothing to add.",
			},
			"documents": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"properties":           documentProperties,
					"required":             append([]string{"documentId"}, deliveryread.Required()...),
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"verdict", "summary", "documents"},
		"additionalProperties": false,
	}
}

// Validate refuses an answer the queue cannot show: an unknown verdict, or a
// concern that names no reason when no check has found anything either.
func Validate(answer *Answer, item *billingqueue.BillingQueueItem) error {
	switch billingqueue.AgentVerdict(answer.Verdict) {
	case billingqueue.AgentVerdictAgrees:
		return nil
	case billingqueue.AgentVerdictConcern:
		if strings.TrimSpace(answer.Summary) == "" && !hasOpenIssue(item) {
			return errors.New("a concern with every check clear names no reason")
		}

		return nil
	default:
		return fmt.Errorf("unknown verdict %q", answer.Verdict)
	}
}

func hasOpenIssue(item *billingqueue.BillingQueueItem) bool {
	if item.Review == nil {
		return false
	}
	for _, issue := range item.Review.Issues {
		if issue.IsOpen() {
			return true
		}
	}

	return false
}
