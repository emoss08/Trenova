package customer

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type CustomerBillingProfileDocumentType struct {
	bun.BaseModel `bun:"table:customer_billing_profile_document_types,alias:cbpdt" json:"-"`
	keel.Table    `keel:"customer_billing_profile_document_types"`

	OrganizationID   pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	BillingProfileID pulid.ID `bun:"billing_profile_id,pk,notnull,type:VARCHAR(100)" keel:"billing_profile_id,pk"`
	DocumentTypeID   pulid.ID `bun:"document_type_id,pk,notnull,type:VARCHAR(100)" keel:"document_type_id,pk"`

	BillingProfile *CustomerBillingProfile    `bun:"rel:belongs-to,join:billing_profile_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:billing_profile_id"`
	DocumentType   *documenttype.DocumentType `bun:"rel:belongs-to,join:document_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:document_type_id"`
}

func (d *CustomerBillingProfileDocumentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&d.BillingProfileID,
			validation.Required.Error("Billing profile is required"),
		),
		validation.Field(&d.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
	))
}
