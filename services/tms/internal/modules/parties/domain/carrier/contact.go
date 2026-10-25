package carrier

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type CarrierContact struct {
	keel.Table `keel:"carrier_contacts"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:carc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	CarrierID      pulid.ID `json:"carrierId"      keel:"carrier_id"`

	Name                      string `json:"name"                      keel:"name"`
	Title                     string `json:"title"                     keel:"title,nullzero"`
	Email                     string `json:"email"                     keel:"email,nullzero"`
	Phone                     string `json:"phone"                     keel:"phone,nullzero"`
	IsPrimary                 bool   `json:"isPrimary"                 keel:"is_primary"`
	ReceivesRateConfirmations bool   `json:"receivesRateConfirmations" keel:"receives_rate_confirmations"`
	Version                   int64  `json:"version"                   keel:"version,version"`
	CreatedAt                 int64  `json:"createdAt"                 keel:"created_at,autocreate"`
	UpdatedAt                 int64  `json:"updatedAt"                 keel:"updated_at,autoupdate"`

	Carrier *Carrier `json:"carrier,omitempty" keel:"belongs:carrier_id"`
}

func (cc *CarrierContact) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		cc,
		validation.Field(&cc.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&cc.Email,
			validation.When(cc.Email != "",
				is.EmailFormat.Error("Email must be a valid email address"),
			),
			validation.When(cc.ReceivesRateConfirmations,
				validation.Required.Error(
					"A rate confirmation recipient must have an email address",
				),
			),
		),
		validation.Field(&cc.Title,
			validation.Length(0, 100).Error("Title must be at most 100 characters"),
		),
	))
}

func (cc *CarrierContact) GetID() pulid.ID {
	return cc.ID
}

func (cc *CarrierContact) GetCreatedAt() int64 {
	return cc.CreatedAt
}

func (cc *CarrierContact) GetTableName() string {
	return "carrier_contacts"
}

func (cc *CarrierContact) GetOrganizationID() pulid.ID {
	return cc.OrganizationID
}

func (cc *CarrierContact) GetBusinessUnitID() pulid.ID {
	return cc.BusinessUnitID
}
