package carrier

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierContact)(nil)

type CarrierContact struct {
	bun.BaseModel `bun:"table:carrier_contacts,alias:carc" json:"-"`
	keel.Table    `keel:"carrier_contacts"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:carc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull" keel:"carrier_id"`

	Name                      string `json:"name"                      bun:"name,type:VARCHAR(255),notnull" keel:"name"`
	Title                     string `json:"title"                     bun:"title,type:VARCHAR(100),nullzero" keel:"title,nullzero"`
	Email                     string `json:"email"                     bun:"email,type:VARCHAR(255),nullzero" keel:"email,nullzero"`
	Phone                     string `json:"phone"                     bun:"phone,type:VARCHAR(20),nullzero" keel:"phone,nullzero"`
	IsPrimary                 bool   `json:"isPrimary"                 bun:"is_primary,type:BOOLEAN,notnull,default:false" keel:"is_primary"`
	ReceivesRateConfirmations bool   `json:"receivesRateConfirmations" bun:"receives_rate_confirmations,type:BOOLEAN,notnull,default:false" keel:"receives_rate_confirmations"`
	Version                   int64  `json:"version"                   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt                 int64  `json:"createdAt"                 bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                 int64  `json:"updatedAt"                 bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Carrier *Carrier `json:"carrier,omitempty" bun:"rel:belongs-to,join:carrier_id=id" keel:"belongs:carrier_id"`
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

func (cc *CarrierContact) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cc.ID.IsNil() {
			cc.ID = pulid.MustNew("carc_")
		}

		cc.CreatedAt = now
	case *bun.UpdateQuery:
		cc.UpdatedAt = now
	}

	return nil
}
