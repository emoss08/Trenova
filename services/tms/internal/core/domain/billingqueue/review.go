package billingqueue

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// CheckKey names one of the five things a biller confirms before approving.
type CheckKey string

const (
	CheckBiller    = CheckKey("biller")
	CheckCharges   = CheckKey("charges")
	CheckPOD       = CheckKey("pod")
	CheckTerms     = CheckKey("terms")
	CheckDuplicate = CheckKey("duplicate")
)

// AllCheckKeys is the five checks in the order a biller works through them.
func AllCheckKeys() []CheckKey {
	return []CheckKey{CheckBiller, CheckCharges, CheckPOD, CheckTerms, CheckDuplicate}
}

// CheckState is what a check asks of the person: nothing, a decision they can
// make from the item, or a fix that has to happen first.
type CheckState string

const (
	CheckStateOK   = CheckState("ok")
	CheckStateWarn = CheckState("warn")
	CheckStateFail = CheckState("fail")
)

// Check codes say what a check found, so the reader can word it in the
// person's language. Detail carries the same finding in English for anything
// that reads the item without a translation table.
const (
	CheckCodeAssigned    = "assigned"
	CheckCodeUnassigned  = "unassigned"
	CheckCodeIssue       = "issue"
	CheckCodeResolved    = "resolved"
	CheckCodeMatches     = "matches"
	CheckCodeNoRateCon   = "no_rate_con"
	CheckCodeSigned      = "signed"
	CheckCodeOnFile      = "on_file"
	CheckCodeNotRequired = "not_required"
	CheckCodeTerms       = "terms"
	CheckCodeNoProfile   = "no_profile"
	CheckCodeUnique      = "unique"
)

// Check is one of the five checks as it stands for the item right now.
type Check struct {
	Key   CheckKey   `json:"key"`
	State CheckState `json:"state"`
	Code  string     `json:"code"`
	// Detail is the finding in a line: who is assigned, what was flagged.
	Detail string `json:"detail"`
	// IssueID is the issue the check is waiting on, or the one that settled it.
	IssueID pulid.ID `json:"issueId,omitempty"`
	// Facts are what the reader needs to word the finding itself.
	Facts map[string]any `json:"facts,omitempty"`
}

// Blocker is why Approve is not available, as a key the reader words.
const (
	BlockerNone   = ""
	BlockerHold   = "hold"
	BlockerBiller = "biller"
	BlockerIssue  = "issue"
	BlockerStatus = "status"
)

// ChargeSource is where a line's expected amount came from.
type ChargeSource string

const (
	ChargeSourceRating    = ChargeSource("rating")
	ChargeSourceAgreement = ChargeSource("agreement")
	ChargeSourceFuel      = ChargeSource("fuel")
	ChargeSourceDetention = ChargeSource("detention")
	ChargeSourceQuote     = ChargeSource("quote")
	ChargeSourceNone      = ChargeSource("none")
)

// ChargeLine is one charge in the ledger: what it is and how it was priced,
// what the rate con says it should be, and what the item bills.
type ChargeLine struct {
	Key                string              `json:"key"`
	AdditionalChargeID pulid.ID            `json:"additionalChargeId,omitempty"`
	Label              string              `json:"label"`
	Basis              string              `json:"basis"`
	Expected           decimal.NullDecimal `json:"expected"`
	Billed             decimal.Decimal     `json:"billed"`
	Source             ChargeSource        `json:"source"`
	// Flagged is a line an open issue points at.
	Flagged bool `json:"flagged"`
	// Removed is a line a settled issue took off the bill; it is kept in the
	// ledger, struck, so the person can see and undo what was done.
	Removed bool `json:"removed"`
	// Adjusted is a line a settled issue repriced.
	Adjusted bool `json:"adjusted"`
	// IssueID is the issue whose settlement removed or adjusted the line.
	IssueID pulid.ID `json:"issueId,omitempty"`
}

// ChargeReview is the item's charges against the rate con.
type ChargeReview struct {
	Lines         []*ChargeLine   `json:"lines"`
	ExpectedTotal decimal.Decimal `json:"expectedTotal"`
	BilledTotal   decimal.Decimal `json:"billedTotal"`
	// Difference is billed less expected: above zero is over the rate con.
	Difference decimal.Decimal `json:"difference"`
	// HasRateCon is false when nothing priced the load to compare against.
	HasRateCon bool `json:"hasRateCon"`
}

// Totals recomputes the review's totals from its lines. A line without an
// expected amount expects nothing, which is how a charge off the rate con
// shows up as an overage.
func (r *ChargeReview) Totals() {
	r.ExpectedTotal = decimal.Zero
	r.BilledTotal = decimal.Zero
	r.HasRateCon = false
	for _, line := range r.Lines {
		if line.Expected.Valid {
			r.ExpectedTotal = r.ExpectedTotal.Add(line.Expected.Decimal)
			r.HasRateCon = true
		}
		if !line.Removed {
			r.BilledTotal = r.BilledTotal.Add(line.Billed)
		}
	}
	r.Difference = r.BilledTotal.Sub(r.ExpectedTotal)
}

// InvoiceRef is the invoice the item became, as the item's header shows it.
type InvoiceRef struct {
	ID     pulid.ID `json:"id"`
	Number string   `json:"number"`
	// DraftNumber is the number with the draft mark, shown until it posts.
	DraftNumber string `json:"draftNumber"`
	Status      string `json:"status"`
	InvoiceDate int64  `json:"invoiceDate"`
	DueDate     *int64 `json:"dueDate"`
	PostedAt    *int64 `json:"postedAt"`
	Posted      bool   `json:"posted"`
}

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

// BillingTerms is how and when the bill-to pays, and who receives the invoice.
type BillingTerms struct {
	PaymentTerm string `json:"paymentTerm"`
	NetDays     int    `json:"netDays"`
	DueDate     *int64 `json:"dueDate"`
	// Recipients are the bill-to's invoice addresses, first one first.
	Recipients []string `json:"recipients"`
	CreditHold bool     `json:"creditHold"`
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

// DocumentState is how a document the item needs stands.
type DocumentState string

const (
	DocumentStateOK       = DocumentState("ok")
	DocumentStateMissing  = DocumentState("missing")
	DocumentStateUnsigned = DocumentState("unsigned")
)

// DocumentTile is one paperwork tile under the shipment.
type DocumentTile struct {
	Code       string        `json:"code"`
	Name       string        `json:"name"`
	DocumentID pulid.ID      `json:"documentId,omitempty"`
	State      DocumentState `json:"state"`
	Required   bool          `json:"required"`
	Signed     bool          `json:"signed"`
	FileName   string        `json:"fileName"`
}

// DuplicateRef is another bill for the same freight and payer.
type DuplicateRef struct {
	Kind   string   `json:"kind"`
	ID     pulid.ID `json:"id"`
	Number string   `json:"number"`
	Status string   `json:"status"`
}

// Review is the item as a biller reviews it.
type Review struct {
	Checks     []*Check        `json:"checks"`
	NeedsCount int             `json:"needsCount"`
	Ready      bool            `json:"ready"`
	Blocker    string          `json:"blocker"`
	Issues     []*Issue        `json:"issues"`
	Charges    *ChargeReview   `json:"charges"`
	Invoice    *InvoiceRef     `json:"invoice,omitempty"`
	Terms      *BillingTerms   `json:"terms,omitempty"`
	Documents  []*DocumentTile `json:"documents"`
	Duplicates []*DuplicateRef `json:"duplicates"`
	BillerName string          `json:"billerName"`
}

// IsReviewable reports whether an item is still before approval, where its
// checks mean something.
func IsReviewable(status Status) bool {
	return status == StatusReadyForReview || status == StatusInReview
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
	Charges      *ChargeReview
	POD          *PODState
	Terms        *BillingTerms
	ShipmentRef  string
	CustomerName string
}

// BuildReadiness decides the five checks, how many need the person, and
// whether Approve is open. It reads only what it is given, so the same item
// always reads the same way, and the issues are the only thing a person
// settles from the item: a check with an open issue waits on it, and a check
// whose issue a person settled says how.
func BuildReadiness(item *BillingQueueItem, issues []*Issue, facts CheckFacts) *Review {
	review := &Review{
		Checks:     make([]*Check, 0, 5),
		Issues:     issues,
		Charges:    facts.Charges,
		Terms:      facts.Terms,
		BillerName: facts.BillerName,
	}

	review.Checks = append(review.Checks, billerCheck(item, facts))
	for _, key := range []CheckKey{CheckCharges, CheckPOD, CheckTerms, CheckDuplicate} {
		if check := issueCheck(key, issues); check != nil {
			review.Checks = append(review.Checks, check)
			continue
		}
		review.Checks = append(review.Checks, defaultCheck(key, facts))
	}

	for _, check := range review.Checks {
		if check.State != CheckStateOK {
			review.NeedsCount++
		}
	}

	switch {
	case item.Status == StatusOnHold:
		review.Blocker = BlockerHold
	case !IsReviewable(item.Status):
		review.Blocker = BlockerStatus
	case item.AssignedBillerID == nil || item.AssignedBillerID.IsNil():
		review.Blocker = BlockerBiller
	case review.NeedsCount > 0:
		review.Blocker = BlockerIssue
	}
	review.Ready = review.Blocker == BlockerNone

	return review
}

func billerCheck(item *BillingQueueItem, facts CheckFacts) *Check {
	if item.AssignedBillerID == nil || item.AssignedBillerID.IsNil() {
		return &Check{
			Key:    CheckBiller,
			State:  CheckStateFail,
			Code:   CheckCodeUnassigned,
			Detail: "Nobody is assigned",
		}
	}

	name := facts.BillerName
	if name == "" {
		name = "Assigned"
	}

	return &Check{
		Key:    CheckBiller,
		State:  CheckStateOK,
		Code:   CheckCodeAssigned,
		Detail: name,
		Facts:  map[string]any{"biller": facts.BillerName},
	}
}

// issueCheck is the check as its issues leave it: waiting on the oldest open
// one, settled by the latest a person resolved, or nil when it has none that
// a person has seen.
func issueCheck(key CheckKey, issues []*Issue) *Check {
	var settled *Issue
	for _, issue := range issues {
		if issue.CheckKey != key {
			continue
		}
		if issue.IsOpen() {
			state := CheckStateWarn
			if len(issue.Options) == 0 {
				state = CheckStateFail
			}
			return &Check{
				Key:     key,
				State:   state,
				Code:    CheckCodeIssue,
				Detail:  issue.Summary,
				IssueID: issue.ID,
			}
		}
		if issue.ResolutionKey != nil && *issue.ResolutionKey != ResolutionCleared &&
			(settled == nil || derefInt(issue.ResolvedAt) >= derefInt(settled.ResolvedAt)) {
			settled = issue
		}
	}
	if settled == nil {
		return nil
	}

	return &Check{
		Key:     key,
		State:   CheckStateOK,
		Code:    CheckCodeResolved,
		Detail:  settled.ResolutionText,
		IssueID: settled.ID,
	}
}

func defaultCheck(key CheckKey, facts CheckFacts) *Check {
	switch key { //nolint:exhaustive // the biller check is decided above
	case CheckCharges:
		if facts.Charges != nil && facts.Charges.HasRateCon {
			return &Check{
				Key:    key,
				State:  CheckStateOK,
				Code:   CheckCodeMatches,
				Detail: "Every line is on the rate con",
			}
		}
		return &Check{
			Key:    key,
			State:  CheckStateOK,
			Code:   CheckCodeNoRateCon,
			Detail: "Billed as rated",
		}
	case CheckPOD:
		pod := facts.POD
		switch {
		case pod != nil && pod.Signed:
			at := pod.SignedAt
			if at == nil {
				at = pod.DeliveredAt
			}
			return &Check{
				Key:    key,
				State:  CheckStateOK,
				Code:   CheckCodeSigned,
				Detail: "Signed",
				Facts:  map[string]any{"at": at},
			}
		case pod != nil && pod.Present:
			return &Check{
				Key:    key,
				State:  CheckStateOK,
				Code:   CheckCodeOnFile,
				Detail: "On file",
				Facts:  map[string]any{"at": pod.DeliveredAt},
			}
		default:
			return &Check{
				Key:    key,
				State:  CheckStateOK,
				Code:   CheckCodeNotRequired,
				Detail: "Not required for this customer",
			}
		}
	case CheckTerms:
		terms := facts.Terms
		if terms == nil || terms.PaymentTerm == "" {
			return &Check{
				Key:    key,
				State:  CheckStateOK,
				Code:   CheckCodeNoProfile,
				Detail: facts.CustomerName,
			}
		}
		contact := ""
		if len(terms.Recipients) > 0 {
			contact = terms.Recipients[0]
		}
		return &Check{
			Key:    key,
			State:  CheckStateOK,
			Code:   CheckCodeTerms,
			Detail: strings.TrimPrefix(strings.Join([]string{contact, terms.PaymentTerm}, " · "), " · "),
			Facts: map[string]any{
				"contact":     contact,
				"paymentTerm": terms.PaymentTerm,
				"netDays":     terms.NetDays,
			},
		}
	default:
		return &Check{
			Key:    key,
			State:  CheckStateOK,
			Code:   CheckCodeUnique,
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
