package edi

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable = (*EDITemplate)(nil)
)

type EDITemplate struct {
	keel.Table `keel:"edi_templates"`

	ID             pulid.ID          `json:"id"             keel:"id,pk,autoid:editpl_"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	DocumentTypeID pulid.ID          `json:"documentTypeId" keel:"document_type_id"`
	Name           string            `json:"name"           keel:"name"`
	Description    string            `json:"description"    keel:"description,nullzero"`
	Direction      DocumentDirection `json:"direction"      keel:"direction"`
	Standard       EDIStandard       `json:"standard"       keel:"standard"`
	TransactionSet TransactionSet    `json:"transactionSet" keel:"transaction_set"`
	Status         TemplateStatus    `json:"status"         keel:"status"`
	Version        int64             `json:"version"        keel:"version,version"`
	CreatedAt      int64             `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64             `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string            `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`

	DocumentType  *EDIDocumentType      `json:"documentType,omitempty"  keel:"belongs:document_type_id"`
	ActiveVersion *EDITemplateVersion   `json:"activeVersion,omitempty" keel:"one:template_id"`
	Versions      []*EDITemplateVersion `json:"versions,omitempty"      keel:"many:template_id"`
}

func (t *EDITemplate) GetID() pulid.ID {
	return t.ID
}

func (t *EDITemplate) GetCreatedAt() int64 {
	return t.CreatedAt
}

func (t *EDITemplate) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *EDITemplate) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *EDITemplate) GetTableName() string {
	return "edi_templates"
}

func (t *EDITemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "et",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}

func (t *EDITemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&t.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&t.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&t.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TemplateStatus]("Status is invalid"),
		),
	))
}
