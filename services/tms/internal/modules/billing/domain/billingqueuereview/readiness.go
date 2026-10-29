package billingqueuereview

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
)

// DraftInvoiceNumber marks an invoice number as a draft's: INV-24101 reads
// INV-D-24101 until the invoice posts.
func DraftInvoiceNumber(number string) string {
	number = strings.TrimSpace(number)
	if number == "" {
		return ""
	}
	if prefix, rest, ok := strings.Cut(number, "-"); ok && prefix != "" && rest != "" {
		return prefix + "-D-" + rest
	}

	return "D-" + number
}

// NetDaysFor is the number of days a payment term gives.
func NetDaysFor(term string) int {
	switch term {
	case "Net10":
		return 10
	case "Net15":
		return 15
	case "Net30":
		return 30
	case "Net45":
		return 45
	case "Net60":
		return 60
	case "Net90":
		return 90
	default:
		return 0
	}
}

// ParseRecipients splits an email profile's recipient list.
func ParseRecipients(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}

	return out
}

// IsReviewable reports whether an item is still before approval, where its
// checks mean something.
func IsReviewable(status billingqueue.Status) bool {
	return status == billingqueue.StatusReadyForReview || status == billingqueue.StatusInReview
}

// PODState is what the shipment's proof of delivery shows.
type PODState struct {
	Required     bool     `json:"required"`
	Present      bool     `json:"present"`
	Signed       bool     `json:"signed"`
	Unsigned     bool     `json:"unsigned"`
	DocumentID   pulid.ID `json:"documentId,omitempty"`
	DocumentType string   `json:"documentType"`
	SignedAt     *int64   `json:"signedAt"`
	DeliveredAt  *int64   `json:"deliveredAt"`
	DriverName   string   `json:"driverName"`
	DriverID     pulid.ID `json:"driverId,omitempty"`
}

// CheckFacts is what the checks read beyond the item and its issues.
type CheckFacts struct {
	BillerName   string
	Charges      *billingqueue.ChargeReview
	POD          *PODState
	Terms        *billingqueue.BillingTerms
	ShipmentRef  string
	CustomerName string
}

// BuildReadiness decides the five checks, how many need the person, and
// whether Approve is open. It reads only what it is given, so the same item
// always reads the same way, and the issues are the only thing a person
// settles from the item: a check with an open issue waits on it, and a check
// whose issue a person settled says how.
func BuildReadiness(
	item *billingqueue.BillingQueueItem,
	issues []*billingqueue.Issue,
	facts CheckFacts,
) *billingqueue.Review {
	review := &billingqueue.Review{
		Checks:     make([]*billingqueue.Check, 0, 5),
		Issues:     issues,
		Charges:    facts.Charges,
		Terms:      facts.Terms,
		BillerName: facts.BillerName,
	}

	review.Checks = append(review.Checks, billerCheck(item, facts))
	for _, key := range []billingqueue.CheckKey{billingqueue.CheckCharges, billingqueue.CheckPOD, billingqueue.CheckTerms, billingqueue.CheckDuplicate} {
		if check := issueCheck(key, issues); check != nil {
			review.Checks = append(review.Checks, check)
			continue
		}
		review.Checks = append(review.Checks, defaultCheck(key, facts))
	}

	for _, check := range review.Checks {
		if check.State != billingqueue.CheckStateOK {
			review.NeedsCount++
		}
	}

	switch {
	case item.Status == billingqueue.StatusOnHold:
		review.Blocker = billingqueue.BlockerHold
	case !IsReviewable(item.Status):
		review.Blocker = billingqueue.BlockerStatus
	case item.AssignedBillerID == nil || item.AssignedBillerID.IsNil():
		review.Blocker = billingqueue.BlockerBiller
	case review.NeedsCount > 0:
		review.Blocker = billingqueue.BlockerIssue
	}
	review.Ready = review.Blocker == billingqueue.BlockerNone

	return review
}

func billerCheck(item *billingqueue.BillingQueueItem, facts CheckFacts) *billingqueue.Check {
	if item.AssignedBillerID == nil || item.AssignedBillerID.IsNil() {
		return &billingqueue.Check{
			Key:    billingqueue.CheckBiller,
			State:  billingqueue.CheckStateFail,
			Code:   billingqueue.CheckCodeUnassigned,
			Detail: "Nobody is assigned",
		}
	}

	name := facts.BillerName
	if name == "" {
		name = "Assigned"
	}

	return &billingqueue.Check{
		Key:    billingqueue.CheckBiller,
		State:  billingqueue.CheckStateOK,
		Code:   billingqueue.CheckCodeAssigned,
		Detail: name,
		Facts:  map[string]any{"biller": facts.BillerName},
	}
}

// issueCheck is the check as its issues leave it: waiting on the oldest open
// one, settled by the latest a person resolved, or nil when it has none that
// a person has seen.
func issueCheck(key billingqueue.CheckKey, issues []*billingqueue.Issue) *billingqueue.Check {
	var settled *billingqueue.Issue
	for _, issue := range issues {
		if issue.CheckKey != key {
			continue
		}
		if issue.IsOpen() {
			state := billingqueue.CheckStateWarn
			if len(issue.Options) == 0 {
				state = billingqueue.CheckStateFail
			}
			return &billingqueue.Check{
				Key:     key,
				State:   state,
				Code:    billingqueue.CheckCodeIssue,
				Detail:  issue.Summary,
				IssueID: issue.ID,
			}
		}
		if issue.ResolutionKey != nil && *issue.ResolutionKey != billingqueue.ResolutionCleared &&
			(settled == nil || derefInt(issue.ResolvedAt) >= derefInt(settled.ResolvedAt)) {
			settled = issue
		}
	}
	if settled == nil {
		return nil
	}

	return &billingqueue.Check{
		Key:     key,
		State:   billingqueue.CheckStateOK,
		Code:    billingqueue.CheckCodeResolved,
		Detail:  settled.ResolutionText,
		IssueID: settled.ID,
	}
}

func defaultCheck(key billingqueue.CheckKey, facts CheckFacts) *billingqueue.Check {
	switch key { //nolint:exhaustive // the biller check is decided above
	case billingqueue.CheckCharges:
		if facts.Charges != nil && facts.Charges.HasRateCon {
			return &billingqueue.Check{
				Key:    key,
				State:  billingqueue.CheckStateOK,
				Code:   billingqueue.CheckCodeMatches,
				Detail: "Every line is on the rate con",
			}
		}
		return &billingqueue.Check{
			Key:    key,
			State:  billingqueue.CheckStateOK,
			Code:   billingqueue.CheckCodeNoRateCon,
			Detail: "Billed as rated",
		}
	case billingqueue.CheckPOD:
		pod := facts.POD
		switch {
		case pod != nil && pod.Signed:
			at := pod.SignedAt
			if at == nil {
				at = pod.DeliveredAt
			}
			return &billingqueue.Check{
				Key:    key,
				State:  billingqueue.CheckStateOK,
				Code:   billingqueue.CheckCodeSigned,
				Detail: "Signed",
				Facts:  map[string]any{"at": at},
			}
		case pod != nil && pod.Present:
			return &billingqueue.Check{
				Key:    key,
				State:  billingqueue.CheckStateOK,
				Code:   billingqueue.CheckCodeOnFile,
				Detail: "On file",
				Facts:  map[string]any{"at": pod.DeliveredAt},
			}
		default:
			return &billingqueue.Check{
				Key:    key,
				State:  billingqueue.CheckStateOK,
				Code:   billingqueue.CheckCodeNotRequired,
				Detail: "Not required for this customer",
			}
		}
	case billingqueue.CheckTerms:
		terms := facts.Terms
		if terms == nil || terms.PaymentTerm == "" {
			return &billingqueue.Check{
				Key:    key,
				State:  billingqueue.CheckStateOK,
				Code:   billingqueue.CheckCodeNoProfile,
				Detail: facts.CustomerName,
			}
		}
		contact := ""
		if len(terms.Recipients) > 0 {
			contact = terms.Recipients[0]
		}
		return &billingqueue.Check{
			Key:   key,
			State: billingqueue.CheckStateOK,
			Code:  billingqueue.CheckCodeTerms,
			Detail: strings.TrimPrefix(
				strings.Join([]string{contact, terms.PaymentTerm}, " · "),
				" · ",
			),
			Facts: map[string]any{
				"contact":     contact,
				"paymentTerm": terms.PaymentTerm,
				"netDays":     terms.NetDays,
			},
		}
	default:
		return &billingqueue.Check{
			Key:    key,
			State:  billingqueue.CheckStateOK,
			Code:   billingqueue.CheckCodeUnique,
			Detail: "No other invoice for " + facts.ShipmentRef,
			Facts:  map[string]any{"shipment": facts.ShipmentRef},
		}
	}
}

func derefInt(v *int64) int64 {
	if v == nil {
		return 0
	}

	return *v
}
