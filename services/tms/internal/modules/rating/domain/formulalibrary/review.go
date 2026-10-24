package formulalibrary

import (
	"context"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*FormulaLibraryReview)(nil)

type FormulaLibraryReview struct {
	bun.BaseModel `bun:"table:formula_library_reviews,alias:flibr" json:"-"`
	keel.Table    `keel:"formula_library_reviews"`

	ID                  pulid.ID                       `json:"id"                  bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:flr_"`
	OrganizationID      pulid.ID                       `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID                       `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	LibraryID           pulid.ID                       `json:"libraryId"           bun:"library_id,type:VARCHAR(100),notnull" keel:"library_id"`
	Round               int32                          `json:"round"               bun:"round,type:SMALLINT,notnull" keel:"round"`
	Decision            formulatemplate.ReviewDecision `json:"decision"            bun:"decision,type:formula_template_review_decision_enum,notnull" keel:"decision"`
	ActorID             *pulid.ID                      `json:"actorId"             bun:"actor_id,type:VARCHAR(100),nullzero" keel:"actor_id"`
	Comment             string                         `json:"comment"             bun:"comment,type:TEXT,nullzero" keel:"comment,nullzero"`
	BaseVersionNumber   int64                          `json:"baseVersionNumber"   bun:"base_version_number,type:BIGINT,notnull,default:0" keel:"base_version_number"`
	RequestedReviewerID *pulid.ID                      `json:"requestedReviewerId" bun:"requested_reviewer_id,type:VARCHAR(100),nullzero" keel:"requested_reviewer_id"`
	ReviewerNote        string                         `json:"reviewerNote"        bun:"reviewer_note,type:TEXT,nullzero" keel:"reviewer_note,nullzero"`
	CreatedAt           int64                          `json:"createdAt"           bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	Actor             *tenant.User `json:"actor,omitempty"             bun:"rel:belongs-to,join:actor_id=id" keel:"belongs:actor_id"`
	RequestedReviewer *tenant.User `json:"requestedReviewer,omitempty" bun:"rel:belongs-to,join:requested_reviewer_id=id" keel:"belongs:requested_reviewer_id"`
}

func (r *FormulaLibraryReview) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.LibraryID, validation.Required.Error("Library is required")),
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

func (r *FormulaLibraryReview) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("flr_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

func (r *FormulaLibraryReview) GetID() pulid.ID {
	return r.ID
}

func (r *FormulaLibraryReview) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *FormulaLibraryReview) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *FormulaLibraryReview) GetTableName() string {
	return "formula_library_reviews"
}
