package customer

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CustomerBillingProfileDocumentType struct {
	keel.Table `keel:"customer_billing_profile_document_types"`

	OrganizationID   pulid.ID `keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `keel:"business_unit_id,pk,tenant"`
	BillingProfileID pulid.ID `keel:"billing_profile_id,pk"`
	DocumentTypeID   pulid.ID `keel:"document_type_id,pk"`

	BillingProfile *CustomerBillingProfile    `keel:"belongs:billing_profile_id"`
	DocumentType   *documenttype.DocumentType `keel:"belongs:document_type_id"`
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
