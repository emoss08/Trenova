package documenttype

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*DocumentType)(nil)
	_ domaintypes.PostgresSearchable = (*DocumentType)(nil)
	_ domain.Validatable             = (*DocumentType)(nil)
	_ framework.TenantedEntity       = (*DocumentType)(nil)
)

type DocumentType struct {
	bun.BaseModel `bun:"table:document_types,alias:dt" json:"-"`

	ID                     pulid.ID               `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID               `json:"businessUnitId"         bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID         pulid.ID               `json:"organizationId"         bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Code                   string                 `json:"code"                   bun:"code,type:VARCHAR(10),notnull"`
	Name                   string                 `json:"name"                   bun:"name,type:VARCHAR(100),notnull"`
	Description            string                 `json:"description"            bun:"description,type:TEXT,nullzero"`
	Color                  string                 `json:"color"                  bun:"color,type:VARCHAR(10)"`
	SearchVector           string                 `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                   string                 `json:"-"                      bun:"rank,type:VARCHAR(100),scanonly"`
	DocumentClassification DocumentClassification `json:"documentClassification" bun:"document_classification,type:document_classification_enum,notnull,default:'Public'"`
	DocumentCategory       DocumentCategory       `json:"documentCategory"       bun:"document_category,type:document_category_enum,notnull,default:'Other'"`
	Version                int64                  `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64                  `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                  `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	BusinessUnit           *tenant.BusinessUnit   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization           *tenant.Organization   `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dt *DocumentType) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dt,
		validation.Field(
			&dt.Code,
			validation.Required.Error("Code is required. Please try again."),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(
			&dt.Name,
			validation.Required.Error("Name is required. Please try again."),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(
			&dt.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
		validation.Field(
			&dt.DocumentClassification,
			validation.Required.Error("Document classification is required. Please try again."),
			validation.In(
				ClassificationPublic,
				ClassificationPrivate,
				ClassificationSensitive,
				ClassificationRegulatory,
			).Error("Document classification must be valid"),
		),
		validation.Field(
			&dt.DocumentCategory,
			validation.Required.Error("Document category is required. Please try again."),
			validation.In(
				CategoryShipment,
				CategoryWorker,
				CategoryRegulatory,
				CategoryProfile,
				CategoryBranding,
				CategoryInvoice,
				CategoryContract,
				CategoryOther,
			).Error("Document category must be valid"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dt *DocumentType) GetID() string {
	return dt.ID.String()
}

func (dt *DocumentType) GetTableName() string {
	return "document_types"
}

func (dt *DocumentType) GetOrganizationID() pulid.ID {
	return dt.OrganizationID
}

func (dt *DocumentType) GetBusinessUnitID() pulid.ID {
	return dt.BusinessUnitID
}

func (dt *DocumentType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dt",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "document_classification",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "document_category",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (dt *DocumentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dt.ID.IsNil() {
			dt.ID = pulid.MustNew("dt_")
		}

		dt.CreatedAt = now
	case *bun.UpdateQuery:
		dt.UpdatedAt = now
	}

	return nil
}
