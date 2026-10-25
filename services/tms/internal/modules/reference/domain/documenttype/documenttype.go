package documenttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// CodePOD is the code a proof of delivery is filed under.
const CodePOD = "POD"

var (
	_ validationframework.TenantedEntity = (*DocumentType)(nil)
	_ domaintypes.PostgresSearchable     = (*DocumentType)(nil)
	_ pagination.CursorEntity            = (*DocumentType)(nil)
)

type DocumentType struct {
	keel.Table             `keel:"document_types"`
	ID                     pulid.ID               `keel:"id,pk,autoid:dt_"           json:"id"`
	BusinessUnitID         pulid.ID               `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID         pulid.ID               `keel:"organization_id,pk,tenant"  json:"organizationId"`
	Code                   string                 `keel:"code"                       json:"code"`
	Name                   string                 `keel:"name"                       json:"name"`
	Description            string                 `keel:"description,nullzero"       json:"description"`
	Color                  string                 `keel:"color,nullzero"             json:"color"`
	DocumentClassification DocumentClassification `keel:"document_classification"    json:"documentClassification"`
	DocumentCategory       DocumentCategory       `keel:"document_category"          json:"documentCategory"`
	IsSystem               bool                   `keel:"is_system"                  json:"isSystem"`
	// RequiresSignature makes a document of this type count only once it is
	// signed, the way a proof of delivery is proof only with the receiver's name.
	RequiresSignature bool   `json:"requiresSignature" keel:"requires_signature"`
	Version           int64  `json:"version"           keel:"version,version"`
	CreatedAt         int64  `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64  `json:"updatedAt"         keel:"updated_at,autoupdate"`
	SearchVector      string `json:"-"                 keel:"search_vector,tsvector,readonly,noselect"`
	Rank              string `json:"-"                 keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (dt *DocumentType) BeforeCreate(_ context.Context) error {
	if dt.DocumentClassification == "" {
		dt.DocumentClassification = ClassificationPublic
	}
	if dt.DocumentCategory == "" {
		dt.DocumentCategory = CategoryOther
	}

	return nil
}

func (dt *DocumentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dt,
		validation.Field(&dt.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&dt.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&dt.Color,
			is.HexColor.Error("Color must be a valid hex color"),
		),
		validation.Field(&dt.DocumentClassification,
			validation.Required.Error("Document classification is required"),
			validation.In(
				ClassificationPublic,
				ClassificationPrivate,
				ClassificationSensitive,
				ClassificationRegulatory,
			).Error("Document classification must be valid"),
		),
		validation.Field(&dt.DocumentCategory,
			validation.Required.Error("Document category is required"),
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
	))
}

func (dt *DocumentType) GetID() pulid.ID {
	return dt.ID
}

func (dt *DocumentType) GetCreatedAt() int64 {
	return dt.CreatedAt
}

func (dt *DocumentType) GetOrganizationID() pulid.ID {
	return dt.OrganizationID
}

func (dt *DocumentType) GetBusinessUnitID() pulid.ID {
	return dt.BusinessUnitID
}

func (dt *DocumentType) GetTableName() string {
	return "document_types"
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

func (DocumentType) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_document_types_code",
			Message: "Document type with this code already exists in your organization",
		},
	}
}
