package ratematrix

import (
	"context"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateMatrixReview)(nil)

type RateMatrixReview struct {
	bun.BaseModel `bun:"table:rate_matrix_reviews,alias:rmr" json:"-"`

	ID                  pulid.ID                       `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID      pulid.ID                       `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID                       `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	RateMatrixID        pulid.ID                       `json:"rateMatrixId"        bun:"rate_matrix_id,type:VARCHAR(100),notnull"`
	Round               int32                          `json:"round"               bun:"round,type:SMALLINT,notnull"`
	Decision            formulatemplate.ReviewDecision `json:"decision"            bun:"decision,type:formula_template_review_decision_enum,notnull"`
	ActorID             *pulid.ID                      `json:"actorId"             bun:"actor_id,type:VARCHAR(100),nullzero"`
	Comment             string                         `json:"comment"             bun:"comment,type:TEXT,nullzero"`
	BaseVersionNumber   int64                          `json:"baseVersionNumber"   bun:"base_version_number,type:BIGINT,notnull,default:0"`
	RequestedReviewerID *pulid.ID                      `json:"requestedReviewerId" bun:"requested_reviewer_id,type:VARCHAR(100),nullzero"`
	ReviewerNote        string                         `json:"reviewerNote"        bun:"reviewer_note,type:TEXT,nullzero"`
	CreatedAt           int64                          `json:"createdAt"           bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Actor             *tenant.User `json:"actor,omitempty"             bun:"rel:belongs-to,join:actor_id=id"`
	RequestedReviewer *tenant.User `json:"requestedReviewer,omitempty" bun:"rel:belongs-to,join:requested_reviewer_id=id"`
}

func (r *RateMatrixReview) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.RateMatrixID, validation.Required.Error("Rate matrix is required")),
		validation.Field(&r.Round, validation.Min(int32(1)).Error("Round starts at one")),
		validation.Field(&r.Decision,
			validation.Required.Error("Decision is required"),
			domainvalidation.ValidEnum[formulatemplate.ReviewDecision]("Decision is invalid"),
		),
	))

	if utf8.RuneCountInString(r.ReviewerNote) > formulatemplate.MaxReviewerNoteLength {
		multiErr.Add(
			"reviewerNote",
			errortypes.ErrInvalid,
			"A note for the reviewer cannot be longer than 4,000 characters",
		)
	}

	if r.Decision != formulatemplate.ReviewDecisionExpired &&
		(r.ActorID == nil || r.ActorID.IsNil()) {
		multiErr.Add(
			"actorId",
			errortypes.ErrRequired,
			"Every decision but expiry names who made it",
		)
	}
}

func (r *RateMatrixReview) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rmr_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

func (r *RateMatrixReview) GetID() pulid.ID {
	return r.ID
}

func (r *RateMatrixReview) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *RateMatrixReview) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *RateMatrixReview) GetTableName() string {
	return "rate_matrix_reviews"
}
