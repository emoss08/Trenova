package documenttemplate

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*GeneratedDocument)(nil)
	_ domaintypes.PostgresSearchable = (*GeneratedDocument)(nil)
	_ domain.Validatable             = (*GeneratedDocument)(nil)
	_ framework.TenantedEntity       = (*GeneratedDocument)(nil)
)

type GeneratedDocument struct {
	bun.BaseModel `bun:"table:generated_documents,alias:gdoc" json:"-"`

	ID             pulid.ID         `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	DocumentTypeID pulid.ID         `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),notnull"`
	TemplateID     pulid.ID         `json:"templateId"     bun:"template_id,type:VARCHAR(100),notnull"`
	ReferenceType  string           `json:"referenceType"  bun:"reference_type,type:VARCHAR(50),notnull"`
	ReferenceID    pulid.ID         `json:"referenceId"    bun:"reference_id,type:VARCHAR(100),notnull"`
	FileName       string           `json:"fileName"       bun:"file_name,type:VARCHAR(255),notnull"`
	FilePath       string           `json:"filePath"       bun:"file_path,type:VARCHAR(500),notnull"`
	FileSize       int64            `json:"fileSize"       bun:"file_size,type:BIGINT,notnull"`
	MimeType       string           `json:"mimeType"       bun:"mime_type,type:VARCHAR(100),notnull,default:'application/pdf'"`
	Checksum       string           `json:"checksum"       bun:"checksum,type:VARCHAR(64),nullzero"`
	Status         GenerationStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	ErrorMessage   string           `json:"errorMessage"   bun:"error_message,type:TEXT,nullzero"`
	GeneratedAt    *int64           `json:"generatedAt"    bun:"generated_at,type:BIGINT,nullzero"`
	GeneratedByID  *pulid.ID        `json:"generatedById"  bun:"generated_by_id,type:VARCHAR(100),nullzero"`
	DeliveryMethod DeliveryMethod   `json:"deliveryMethod" bun:"delivery_method,type:VARCHAR(50),notnull,default:'None'"`
	DeliveredAt    *int64           `json:"deliveredAt"    bun:"delivered_at,type:BIGINT,nullzero"`
	DeliveredTo    string           `json:"deliveredTo"    bun:"delivered_to,type:VARCHAR(255),nullzero"`
	SearchVector   string           `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string           `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64            `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64            `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit       `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization       `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	DocumentType *documenttype.DocumentType `json:"documentType,omitempty" bun:"rel:belongs-to,join:document_type_id=id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	Template     *DocumentTemplate          `json:"template,omitempty"     bun:"rel:belongs-to,join:template_id=id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	GeneratedBy  *tenant.User               `json:"generatedBy,omitempty"  bun:"rel:belongs-to,join:generated_by_id=id"`
}

func (gd *GeneratedDocument) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(gd,
		validation.Field(
			&gd.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(
			&gd.TemplateID,
			validation.Required.Error("Template is required"),
		),
		validation.Field(
			&gd.ReferenceType,
			validation.Required.Error("Reference type is required"),
			validation.Length(1, 50).Error("Reference type must be between 1 and 50 characters"),
		),
		validation.Field(
			&gd.ReferenceID,
			validation.Required.Error("Reference ID is required"),
		),
		validation.Field(
			&gd.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, 255).Error("File name must be between 1 and 255 characters"),
		),
		validation.Field(
			&gd.FilePath,
			validation.Required.Error("File path is required"),
		),
		validation.Field(
			&gd.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				GenerationStatusPending,
				GenerationStatusProcessing,
				GenerationStatusCompleted,
				GenerationStatusFailed,
			).Error("Status must be Pending, Processing, Completed, or Failed"),
		),
		validation.Field(
			&gd.DeliveryMethod,
			validation.In(
				DeliveryMethodNone,
				DeliveryMethodEmail,
				DeliveryMethodDownload,
				DeliveryMethodPortal,
			).Error("Delivery method must be None, Email, Download, or Portal"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (gd *GeneratedDocument) GetID() string {
	return gd.ID.String()
}

func (gd *GeneratedDocument) GetTableName() string {
	return "generated_documents"
}

func (gd *GeneratedDocument) GetOrganizationID() pulid.ID {
	return gd.OrganizationID
}

func (gd *GeneratedDocument) GetBusinessUnitID() pulid.ID {
	return gd.BusinessUnitID
}

func (gd *GeneratedDocument) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "gdoc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "reference_type",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (gd *GeneratedDocument) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if gd.ID.IsNil() {
			gd.ID = pulid.MustNew("gdoc_")
		}
		gd.CreatedAt = now
	}

	return nil
}

func (gd *GeneratedDocument) MarkCompleted() {
	now := utils.NowUnix()
	gd.Status = GenerationStatusCompleted
	gd.GeneratedAt = &now
}

func (gd *GeneratedDocument) MarkFailed(errMsg string) {
	gd.Status = GenerationStatusFailed
	gd.ErrorMessage = errMsg
}

func (gd *GeneratedDocument) MarkDelivered(method DeliveryMethod, deliveredTo string) {
	now := utils.NowUnix()
	gd.DeliveryMethod = method
	gd.DeliveredAt = &now
	gd.DeliveredTo = deliveredTo
}
