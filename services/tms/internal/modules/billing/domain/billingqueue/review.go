package billingqueue

import (
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

// BillingTerms is how and when the bill-to pays, and who receives the invoice.
type BillingTerms struct {
	PaymentTerm string `json:"paymentTerm"`
	NetDays     int    `json:"netDays"`
	DueDate     *int64 `json:"dueDate"`
	// Recipients are the bill-to's invoice addresses, first one first.
	Recipients []string `json:"recipients"`
	CreditHold bool     `json:"creditHold"`
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

// OpenIssues are the flagged checks on the item nobody has settled, read with
// its review. While any is open the item cannot be approved.
func (b *BillingQueueItem) OpenIssues() []*Issue {
	if b == nil || b.Review == nil {
		return nil
	}
	open := make([]*Issue, 0, len(b.Review.Issues))
	for _, issue := range b.Review.Issues {
		if issue != nil && issue.IsOpen() {
			open = append(open, issue)
		}
	}

	return open
}
