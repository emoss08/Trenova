package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*EDITemplate)(nil)
	_ domaintypes.PostgresSearchable = (*EDITemplate)(nil)
)

type EDITemplate struct {
	bun.BaseModel             `json:"-" bun:"table:edi_templates,alias:et"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	DocumentTypeID pulid.ID          `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),notnull"`
	Name           string            `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string            `json:"description"    bun:"description,type:TEXT,nullzero"`
	Direction      DocumentDirection `json:"direction"      bun:"direction,type:edi_document_direction_enum,notnull"`
	Standard       EDIStandard       `json:"standard"       bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet TransactionSet    `json:"transactionSet" bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	Status         TemplateStatus    `json:"status"         bun:"status,type:edi_template_status_enum,notnull"`
	Version        int64             `json:"version"        bun:"version,type:BIGINT,notnull"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string            `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`

	DocumentType  *EDIDocumentType      `json:"documentType,omitempty"  bun:"rel:belongs-to,join:document_type_id=id"`
	ActiveVersion *EDITemplateVersion   `json:"activeVersion,omitempty" bun:"rel:has-one,join:id=template_id"`
	Versions      []*EDITemplateVersion `json:"versions,omitempty"      bun:"rel:has-many,join:id=template_id"`
}

func (t *EDITemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("editpl_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}
	return nil
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
