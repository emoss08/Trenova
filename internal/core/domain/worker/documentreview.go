package worker

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DocumentReview)(nil)

type DocumentReview struct {
	bun.BaseModel `bun:"table:document_reviews,alias:dr" json:"-"`

	// Primary identifiers
	ID               pulid.ID `json:"id" bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID   pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkerDocumentID pulid.ID `json:"workerDocumentId" bun:"worker_document_id,type:VARCHAR(100),pk,notnull"`
	ReviewerID       pulid.ID `json:"reviewerId" bun:"reviewer_id,type:VARCHAR(100),notnull"`
	// Core Fields
	Status     DocumentStatus `json:"status" bun:"status,type:document_status_enum,notnull"`
	Comments   string         `json:"comments" bun:"comments,type:TEXT"`
	ReviewedAt int64          `json:"reviewedAt" bun:"reviewed_at,type:BIGINT,notnull"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	WorkerDocument *WorkerDocument            `json:"workerDocument,omitempty" bun:"rel:belongs-to,join:worker_document_id=id"`
	Reviewer       *user.User                 `json:"reviewer,omitempty" bun:"rel:belongs-to,join:reviewer_id=id"`
	BusinessUnit   *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dr *DocumentReview) Validate(ctx context.Context, multiErr *errors.MultiError, index int) {
	err := validation.ValidateStructWithContext(ctx, dr,
		// Status is required and must be a valid document status
		validation.Field(&dr.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				DocumentStatusPending,
				DocumentStatusActive,
				DocumentStatusExpired,
				DocumentStatusRejected,
			).Error("Invalid document status"),
		),

		// Comments are required when rejecting a document and must be between 10 and 1000 characters
		validation.Field(&dr.Comments,
			validation.When(dr.Status == DocumentStatusRejected,
				validation.Required.Error("Comments are required when rejecting a document"),
				validation.Length(10, 1000).Error("Comments must be between 10 and 1000 characters"),
			),
		),

		validation.Field(&dr.ReviewedAt,
			validation.Required.Error("Reviewed at is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, fmt.Sprintf("documentReview[%d]", index))
		}
	}
}

func (dr *DocumentReview) GetTableName() string {
	return "document_reviews"
}

// BeforeAppendModel is a bun hook that sets the createdAt and updatedAt fields
func (dr *DocumentReview) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dr.ID == "" {
			dr.ID = pulid.MustNew("dr_")
		}

		dr.CreatedAt = now
	case *bun.UpdateQuery:
		dr.UpdatedAt = now
	}

	return nil
}
