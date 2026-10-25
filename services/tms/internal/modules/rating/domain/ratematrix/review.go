package ratematrix

import (
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type RateMatrixReview struct {
	keel.Table `keel:"rate_matrix_reviews"`

	ID                  pulid.ID                       `json:"id"                  keel:"id,pk,autoid:rmr_"`
	OrganizationID      pulid.ID                       `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID                       `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	RateMatrixID        pulid.ID                       `json:"rateMatrixId"        keel:"rate_matrix_id"`
	Round               int32                          `json:"round"               keel:"round"`
	Decision            formulatemplate.ReviewDecision `json:"decision"            keel:"decision"`
	ActorID             *pulid.ID                      `json:"actorId"             keel:"actor_id"`
	Comment             string                         `json:"comment"             keel:"comment,nullzero"`
	BaseVersionNumber   int64                          `json:"baseVersionNumber"   keel:"base_version_number"`
	RequestedReviewerID *pulid.ID                      `json:"requestedReviewerId" keel:"requested_reviewer_id"`
	ReviewerNote        string                         `json:"reviewerNote"        keel:"reviewer_note,nullzero"`
	CreatedAt           int64                          `json:"createdAt"           keel:"created_at,autocreate"`

	Actor             *tenant.User `json:"actor,omitempty"             keel:"belongs:actor_id"`
	RequestedReviewer *tenant.User `json:"requestedReviewer,omitempty" keel:"belongs:requested_reviewer_id"`
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
