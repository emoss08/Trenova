package carrier

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierEDIChannel)(nil)

type CarrierEDIChannel struct {
	bun.BaseModel `bun:"table:carrier_edi_channels,alias:cec" json:"-"`
	keel.Table    `keel:"carrier_edi_channels"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:cec_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull" keel:"carrier_id"`
	EDIPartnerID   pulid.ID `json:"ediPartnerId"   bun:"edi_partner_id,type:VARCHAR(100),notnull" keel:"edi_partner_id"`

	PartnerDocumentProfileID *pulid.ID `json:"partnerDocumentProfileId" bun:"partner_document_profile_id,type:VARCHAR(100),nullzero" keel:"partner_document_profile_id"`
	CommunicationProfileID   *pulid.ID `json:"communicationProfileId"   bun:"communication_profile_id,type:VARCHAR(100),nullzero" keel:"communication_profile_id"`
	SCACOverride             string    `json:"scacOverride"             bun:"scac_override,type:VARCHAR(10),nullzero" keel:"scac_override,nullzero"`
	IsDefault                bool      `json:"isDefault"                bun:"is_default,type:BOOLEAN,notnull,default:false" keel:"is_default"`
	Version                  int64     `json:"version"                  bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt                int64     `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                int64     `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Carrier    *Carrier        `json:"carrier,omitempty"    bun:"rel:belongs-to,join:carrier_id=id" keel:"belongs:carrier_id"`
	EDIPartner *edi.EDIPartner `json:"ediPartner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id" keel:"belongs:edi_partner_id"`
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

func (cec *CarrierEDIChannel) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cec.ID.IsNil() {
			cec.ID = pulid.MustNew("cec_")
		}
		cec.CreatedAt = now
	case *bun.UpdateQuery:
		cec.UpdatedAt = now
	}

	return nil
}

func (cec *CarrierEDIChannel) BeforeCreate(ctx context.Context) error {
	return cec.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
