package formulatemplate

import (
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// ReviewDecision is one step of a review round.
type ReviewDecision string

const (
	ReviewDecisionSubmitted        = ReviewDecision("Submitted")
	ReviewDecisionApproved         = ReviewDecision("Approved")
	ReviewDecisionRejected         = ReviewDecision("Rejected")
	ReviewDecisionChangesRequested = ReviewDecision("ChangesRequested")
	ReviewDecisionExpired          = ReviewDecision("Expired")
)

func (rd ReviewDecision) String() string {
	return string(rd)
}

func (rd ReviewDecision) IsValid() bool {
	switch rd {
	case ReviewDecisionSubmitted,
		ReviewDecisionApproved,
		ReviewDecisionRejected,
		ReviewDecisionChangesRequested,
		ReviewDecisionExpired:
		return true
	default:
		return false
	}
}

// ClosesRound reports whether the decision ends the round. A request for
// changes leaves it open: the author's next submission continues the same
// conversation rather than starting a new one.
func (rd ReviewDecision) ClosesRound() bool {
	switch rd {
	case ReviewDecisionApproved, ReviewDecisionRejected, ReviewDecisionExpired:
		return true
	default:
		return false
	}
}

// SubmissionExpiry is how long a submission may wait before it is considered
// stale. Rates move; a review of two-week-old content approves something the
// author may no longer stand behind, so the template returns to draft and the
// author resubmits against current tables.
const SubmissionExpiry int64 = 14 * 24 * 60 * 60

const MaxReviewerNoteLength = 4000

// Review is one entry in a template's review history: who did what, in which
// round, against which approved version, and what they said about it.
type Review struct {
	keel.Table `keel:"formula_template_reviews"`

	ID                  pulid.ID       `json:"id" keel:"id,pk,autoid:ftr_"`
	TemplateID          pulid.ID       `json:"templateId" keel:"template_id"`
	OrganizationID      pulid.ID       `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID      pulid.ID       `json:"businessUnitId" keel:"business_unit_id,tenant"`
	Round               int32          `json:"round" keel:"round"`
	Decision            ReviewDecision `json:"decision" keel:"decision"`
	ActorID             *pulid.ID      `json:"actorId" keel:"actor_id"`
	Comment             string         `json:"comment" keel:"comment,nullzero"`
	BaseVersionNumber   int64          `json:"baseVersionNumber" keel:"base_version_number"`
	RequestedReviewerID *pulid.ID      `json:"requestedReviewerId" keel:"requested_reviewer_id"`
	ReviewerNote        string         `json:"reviewerNote" keel:"reviewer_note,nullzero"`
	CreatedAt           int64          `json:"createdAt" keel:"created_at,autocreate"`

	Actor             *tenant.User `json:"actor,omitempty" keel:"belongs:actor_id"`
	RequestedReviewer *tenant.User `json:"requestedReviewer" keel:"belongs:requested_reviewer_id"`
}

func (r *Review) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&r.Round, validation.Min(int32(1)).Error("Round starts at one")),
		validation.Field(&r.Decision,
			validation.Required.Error("Decision is required"),
			domainvalidation.ValidEnum[ReviewDecision]("Decision is invalid"),
		),
	))

	if utf8.RuneCountInString(r.ReviewerNote) > MaxReviewerNoteLength {
		multiErr.Add(
			"reviewerNote",
			errortypes.ErrInvalid,
			"A note for the reviewer cannot be longer than 4,000 characters",
		)
	}

	if r.Decision != ReviewDecisionExpired && (r.ActorID == nil || r.ActorID.IsNil()) {
		multiErr.Add(
			"actorId",
			errortypes.ErrRequired,
			"Every decision but expiry names who made it",
		)
	}
}

// NextRound says which round a new submission belongs to, given the latest
// entry in the history: a fresh round after a closed one, the same round while
// changes are being requested and resubmitted.
func NextRound(latest *Review) int32 {
	if latest == nil {
		return 1
	}
	if latest.Decision.ClosesRound() {
		return latest.Round + 1
	}
	return latest.Round
}

// SubmissionIsStale reports whether a submission has waited longer than the
// expiry allows.
func SubmissionIsStale(submittedAt *int64, now int64) bool {
	return submittedAt != nil && *submittedAt > 0 && now-*submittedAt > SubmissionExpiry
}
