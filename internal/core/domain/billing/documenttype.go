package billing

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DocumentType)(nil)
	_ domain.Validatable        = (*DocumentType)(nil)
	_ infra.PostgresSearchable  = (*DocumentType)(nil)
)

type DocumentType struct {
	bun.BaseModel `bun:"table:document_types,alias:dt" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// Core fields
	Code        string `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Name        string `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`
	Color       string `json:"color" bun:"color,type:VARCHAR(10)"`

	// Bucket Metadata
	DocumentClassification DocumentClassification `json:"documentClassification" bun:"document_classification,type:document_classification_enum,notnull,default:'Public'"`
	DocumentCategory       DocumentCategory       `json:"documentCategory" bun:"document_category,type:document_category_enum,notnull,default:'Other'"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (dt *DocumentType) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, dt,
		// * Ensure code is populated and between 1 and 10 characters
		validation.Field(
			&dt.Code,
			validation.Required.Error("Code is required. Please try again."),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),

		// * Ensure name is populated and between 1 and 100 characters
		validation.Field(
			&dt.Name,
			validation.Required.Error("Name is required. Please try again."),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// * Ensure color is a valid hex color
		validation.Field(
			&dt.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),

		// * Ensure document classification is valid
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

		// * Ensure document category is valid
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
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dt *DocumentType) GetID() string {
	return dt.ID.String()
}

func (dt *DocumentType) GetTableName() string {
	return "document_types"
}

func (dt *DocumentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

func (dt *DocumentType) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "dt",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "name",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
