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
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*WorkerDocument)(nil)

type WorkerDocument struct {
	bun.BaseModel `bun:"table:worker_documents,alias:wdoc" json:"-"`

	// Primary identifiers
	ID                    pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	WorkerID              pulid.ID `json:"workerId" bun:"worker_id,type:VARCHAR(100),notnull"`
	DocumentRequirementID pulid.ID `json:"documentRequirementId" bun:"document_requirement_id,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	// Core fields
	Status     DocumentStatus `json:"status" bun:"status,type:document_status_enum,notnull"`
	FileURL    string         `json:"fileUrl" bun:"file_url,type:VARCHAR(255),notnull"`
	IssueDate  int64          `json:"issueDate" bun:"issue_date,type:BIGINT,notnull"`
	ExpiryDate *int64         `bun:"expiry_date,type:BIGINT" json:"expiryDate"`

	// Document Metadata/Validation
	ValidationData map[string]any `json:"validationData" bun:"validation_data,type:JSONB"`
	ReviewerID     *pulid.ID      `json:"reviewerId" bun:"reviewer_id,type:VARCHAR(100)"`
	ReviewedAt     *int64         `json:"reviewedAt" bun:"reviewed_at,type:BIGINT"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Worker              *Worker                    `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id"`
	DocumentRequirement *DocumentRequirement       `json:"documentRequirement,omitempty" bun:"rel:belongs-to,join:document_requirement_id=id"`
	BusinessUnit        *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization        *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Reviewer            *user.User                 `json:"reviewer,omitempty" bun:"rel:belongs-to,join:reviewer_id=id"`
}

func (wd *WorkerDocument) Validate(ctx context.Context, multiErr *errors.MultiError, index int) {
	err := validation.ValidateStructWithContext(ctx, wd,
		// Status is required and must be a valid document status
		validation.Field(&wd.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				DocumentStatusPending,
				DocumentStatusActive,
				DocumentStatusExpired,
				DocumentStatusRejected,
			).Error("Invalid document status"),
		),

		// File URL is required and must be a valid URL (if the document is not pending)
		validation.Field(&wd.FileURL,
			validation.When(wd.Status != DocumentStatusPending,
				validation.Required.Error("File URL is required"),
				is.URL.Error("Invalid file URL"),
			),
		),

		// Issue date is required
		validation.Field(&wd.IssueDate,
			validation.Required.Error("Issue date is required"),
		),

		// Expiry date (based on requirement type)
		validation.Field(&wd.ExpiryDate,
			validation.When(wd.DocumentRequirement != nil && wd.DocumentRequirement.RequirementType == RequirementTypeOngoing,
				validation.Required.Error("Expiry date is required for ongoing documents"),
				validation.Min(wd.IssueDate).Error("Expiry date must be after issue date"),
			),
		),

		// Reviewer validation (if status is not pending)
		validation.Field(&wd.ReviewerID,
			validation.When(wd.Status != DocumentStatusPending,
				validation.Required.Error("Reviewer is required for non-pending documents"),
			),
		),

		// Review timestamp
		validation.Field(&wd.ReviewedAt,
			validation.When(wd.ReviewerID.IsNotNil(),
				validation.Required.Error("Review timestamp is required when a reviewer is set"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, fmt.Sprintf("document[%d]", index))
		}
	}
}

func (wd *WorkerDocument) DBValidate(ctx context.Context, _ bun.IDB, multiErr *errors.MultiError, index int) {
	// Validate common fields
	wd.Validate(ctx, multiErr, index)

	// Validate document type specific rules
	wd.validateRequirementSpecificRules(multiErr)
}

func (wd *WorkerDocument) validateRequirementSpecificRules(multiErr *errors.MultiError) {
	// Validate that the validation data contains all the required fields from the document requirement
	for rule := range wd.DocumentRequirement.ValidationRules {
		if _, exists := wd.ValidationData[rule]; !exists {
			multiErr.Add(
				"validationData",
				errors.ErrInvalid,
				fmt.Sprintf("Missing required validation data for rule: %s", rule),
			)
		}
	}

	// Check expiration date based on document type
	if wd.DocumentRequirement.RequirementType == RequirementTypeOngoing {
		if wd.ExpiryDate == nil {
			multiErr.Add("expiryDate", errors.ErrInvalid, "Expiry date is required for ongoing documents")
			return
		}

		// Check if the document is expired
		if *wd.ExpiryDate < timeutils.NowUnix() {
			wd.Status = DocumentStatusExpired
		}
	}
}

func (wd *WorkerDocument) GetTableName() string {
	return "worker_documents"
}

// BeforeAppendModel is a bun hook that sets the createdAt and updatedAt fields
func (wd *WorkerDocument) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if wd.ID == "" {
			wd.ID = pulid.MustNew("wd_")
		}

		wd.CreatedAt = now
	case *bun.UpdateQuery:
		wd.UpdatedAt = now
	}

	return nil
}
