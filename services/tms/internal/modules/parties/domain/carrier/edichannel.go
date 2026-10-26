package carrier

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CarrierEDIChannel struct {
	keel.Table `keel:"carrier_edi_channels"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:cec_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	CarrierID      pulid.ID `json:"carrierId"      keel:"carrier_id"`
	EDIPartnerID   pulid.ID `json:"ediPartnerId"   keel:"edi_partner_id"`

	CommunicationProfileID *pulid.ID `json:"communicationProfileId"   keel:"communication_profile_id"`
	SCACOverride           string    `json:"scacOverride"             keel:"scac_override,nullzero"`
	IsDefault              bool      `json:"isDefault"                keel:"is_default"`
	Version                int64     `json:"version"                  keel:"version,version"`
	CreatedAt              int64     `json:"createdAt"                keel:"created_at,autocreate"`
	UpdatedAt              int64     `json:"updatedAt"                keel:"updated_at,autoupdate"`

	Carrier    *Carrier        `json:"carrier,omitempty"    keel:"belongs:carrier_id"`
	EDIPartner *edi.EDIPartner `json:"ediPartner,omitempty" keel:"belongs:edi_partner_id"`
}

func (cec *CarrierEDIChannel) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(cec,
		validation.Field(&cec.EDIPartnerID,
			validation.Required.Error("EDI partner is required"),
		),
		validation.Field(&cec.SCACOverride,
			validation.Length(0, 10).Error("SCAC override must be at most 10 characters"),
		),
	))
}

func (cec *CarrierEDIChannel) GetID() pulid.ID {
	return cec.ID
}

func (cec *CarrierEDIChannel) GetCreatedAt() int64 {
	return cec.CreatedAt
}

func (cec *CarrierEDIChannel) GetTableName() string {
	return "carrier_edi_channels"
}

func (cec *CarrierEDIChannel) GetOrganizationID() pulid.ID {
	return cec.OrganizationID
}

func (cec *CarrierEDIChannel) GetBusinessUnitID() pulid.ID {
	return cec.BusinessUnitID
}

func (CarrierEDIChannel) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_carrier_edi_channels_default",
			Field:   "ediChannels",
			Message: "Carrier can have only one default EDI channel",
		},
		{
			Name:    "uq_carrier_edi_channels_partner",
			Field:   "ediChannels",
			Message: "Carrier already has an EDI channel for this partner",
		},
	}
}
