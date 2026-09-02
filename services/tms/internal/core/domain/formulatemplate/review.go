package formulatemplate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Review)(nil)

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

// Review is one entry in a template's review history: who did what, in which
// round, against which approved version, and what they said about it.
type Review struct {
	bun.BaseModel `bun:"table:formula_template_reviews,alias:ftr" json:"-"`

	ID                pulid.ID       `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	TemplateID        pulid.ID       `json:"templateId"        bun:"template_id,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Round             int32          `json:"round"             bun:"round,type:SMALLINT,notnull"`
	Decision          ReviewDecision `json:"decision"          bun:"decision,type:formula_template_review_decision_enum,notnull"`
	ActorID           *pulid.ID      `json:"actorId"           bun:"actor_id,type:VARCHAR(100),nullzero"`
	Comment           string         `json:"comment"           bun:"comment,type:TEXT,nullzero"`
	BaseVersionNumber int64          `json:"baseVersionNumber" bun:"base_version_number,type:BIGINT,notnull,default:0"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Actor *tenant.User `json:"actor,omitempty" bun:"rel:belongs-to,join:actor_id=id"`
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

	if r.Decision != ReviewDecisionExpired && (r.ActorID == nil || r.ActorID.IsNil()) {
		multiErr.Add("actorId", errortypes.ErrRequired, "Every decision but expiry names who made it")
	}
}

func (r *Review) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ftr_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
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
