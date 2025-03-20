package document

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Document)(nil)
	_ domain.Validatable        = (*Document)(nil)
	_ infra.PostgresSearchable  = (*Document)(nil)
)

type Document struct {
	bun.BaseModel `bun:"table:documents,alias:doc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Core Properties
	FileName     string         `json:"fileName" bun:"file_name,notnull,type:VARCHAR(255)"`
	OriginalName string         `json:"originalName" bun:"original_name,notnull,type:VARCHAR(255)"`
	FileSize     int64          `json:"fileSize" bun:"file_size,notnull,type:BIGINT"`
	FileType     string         `json:"fileType" bun:"file_type,notnull,type:VARCHAR(100)"`
	StoragePath  string         `json:"storagePath" bun:"storage_path,notnull,type:TEXT"`
	DocumentType DocumentType   `json:"documentType" bun:"document_type,notnull,type:document_type_enum"`
	Status       DocumentStatus `json:"status" bun:"status,notnull,type:document_status_enum"`
	Description  string         `json:"description" bun:"description,type:TEXT"`

	// Entity Association (polymorphic relationship)
	ResourceID   pulid.ID            `json:"resourceId" bun:"resource_id,notnull,type:VARCHAR(100)"`
	ResourceType permission.Resource `json:"resourceType" bun:"resource_type,notnull,type:VARCHAR(100)"`

	// Additional Metadata
	ExpirationDate *int64   `json:"expirationDate" bun:"expiration_date,type:BIGINT,nullzero"`
	Tags           []string `json:"tags" bun:"tags,array,type:VARCHAR(100)"`

	// Audit Fields
	UploadedByID pulid.ID  `json:"uploadedById" bun:"uploaded_by_id,notnull,type:VARCHAR(100)"`
	ApprovedByID *pulid.ID `json:"approvedByID" bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt   *int64    `json:"approvedAt" bun:"approved_at,type:BIGINT,nullzero"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`
	PresignedURL string `json:"presignedURL,omitempty" bun:"presigned_url,type:TEXT,nullzero,scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	UploadedBy   *user.User                 `bun:"rel:belongs-to,join:uploaded_by_id=id" json:"uploadedBy,omitempty"`
	ApprovedBy   *user.User                 `bun:"rel:belongs-to,join:approved_by_id=id" json:"approvedBy,omitempty"`
}

// Validate performs validation on the Document struct
func (d *Document) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, d,
		// * Ensure file name is required and valid length
		validation.Field(&d.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, 255).Error("file name must be between 1 and 255 characters"),
		),

		// * Ensure original name is requried and valid length
		validation.Field(&d.OriginalName,
			validation.Required.Error("Original file name is required"),
			validation.Length(1, 255).Error("Original file name must be between 1 and 255 characters"),
		),

		// * Ensure file size is required and greater than 1
		validation.Field(&d.FileSize,
			validation.Required.Error("File size is required"),
			validation.Min(1).Error("File size must be greater than 0"),
		),

		// * Ensure field type is required and valid length
		validation.Field(&d.FileType,
			validation.Required.Error("File type is required"),
			validation.Length(1, 100).Error("File type must be between 1 and 100 characters"),
		),

		// * Ensure storage path is required and valid length
		validation.Field(&d.StoragePath,
			validation.Required.Error("Storage path is required"),
			validation.Length(1, 500).Error("Storage path must be between 1 and 500 characters"),
		),

		// * Document classification validations
		validation.Field(&d.DocumentType,
			validation.Required.Error("Document type is required"),
			validation.In(
				DocumentTypeLicense,
				DocumentTypeRegistration,
				DocumentTypeInsurance,
				DocumentTypeInvoice,
				DocumentTypeProofOfDelivery,
				DocumentTypeBillOfLading,
				DocumentTypeDriverLog,
				DocumentTypeMedicalCert,
				DocumentTypeContract,
				DocumentTypeMaintenance,
				DocumentTypeAccidentReport,
				DocumentTypeTrainingRecord,
				DocumentTypeOther,
			).Error("Document type must be valid"),
		),
		validation.Field(&d.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				DocumentStatusDraft,
				DocumentStatusActive,
				DocumentStatusArchived,
				DocumentStatusExpired,
				DocumentStatusRejected,
				DocumentStatusPendingApproval,
			).Error("Status must be valid"),
		),

		// * Entity association validations
		validation.Field(&d.ResourceID,
			validation.Required.Error("Entity ID is required"),
		),

		// * Audit field validations
		validation.Field(&d.UploadedByID,
			validation.Required.Error("Uploaded by ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the document ID as a string
func (d *Document) GetID() string {
	return d.ID.String()
}

// GetTableName returns the database table name
func (d *Document) GetTableName() string {
	return "documents"
}

// BeforeAppendModel is called before the model is appended to a query
func (d *Document) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("doc_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

// CheckExpiration determines if the document is expired
func (d *Document) CheckExpiration() bool {
	if d.ExpirationDate == nil {
		return false
	}

	now := timeutils.NowUnix()
	return *d.ExpirationDate <= now
}

// IsApproved checks if the document has been approved
func (d *Document) IsApproved() bool {
	return d.ApprovedByID != nil && d.ApprovedAt != nil
}

func (d *Document) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "doc",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "file_name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "original_name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:       "description",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeText,
				Dictionary: "english",
			},
			{
				Name:       "document_type",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
			{
				Name:       "tags",
				Weight:     "C",
				Type:       infra.PostgresSearchTypeArray,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
