package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Organization)(nil)
	_ domain.Validatable        = (*Organization)(nil)
)

type Metadata struct {
	ObjectID string `json:"objectId"`
}

type Organization struct {
	bun.BaseModel `bun:"table:organizations,alias:org" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	StateID        pulid.ID  `json:"stateId"        bun:"state_id,type:VARCHAR(100),notnull"`
	Name           string    `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	ScacCode       string    `json:"scacCode"       bun:"scac_code,type:VARCHAR(4),notnull"`
	DOTNumber      string    `json:"dotNumber"      bun:"dot_number,type:VARCHAR(8),notnull"`
	LogoURL        string    `json:"logoUrl"        bun:"logo_url,type:VARCHAR(255)"`
	OrgType        Type      `json:"orgType"        bun:"org_type,type:org_type_enum,notnull,default:'Carrier'"`
	BucketName     string    `json:"bucketName"     bun:"bucket_name,type:VARCHAR(63),notnull"`
	AddressLine1   string    `json:"addressLine1"   bun:"address_line1,type:VARCHAR(150),notnull"`
	AddressLine2   string    `json:"addressLine2"   bun:"address_line2,type:VARCHAR(150)"`
	City           string    `json:"city"           bun:"city,type:VARCHAR(100),notnull"`
	PostalCode     string    `json:"postalCode"     bun:"postal_code,type:us_postal_code,notnull"`
	Timezone       string    `json:"timezone"       bun:"timezone,type:VARCHAR(100),notnull,default:'America/New_York'"`
	TaxID          string    `json:"taxId"          bun:"tax_id,type:VARCHAR(50)"`
	Metadata       *Metadata `json:"-"              bun:"metadata,type:JSONB"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit    *BusinessUnit    `json:"businessUnit,omitempty"    bun:"rel:belongs-to,join:business_unit_id=id"`
	BillingControl  *BillingControl  `json:"billingControl,omitempty"  bun:"rel:has-one,join:id=organization_id"`
	ShipmentControl *ShipmentControl `json:"shipmentControl,omitempty" bun:"rel:has-one,join:id=organization_id"`
	DataRetention   *DataRetention   `json:"dataRetention,omitempty"   bun:"rel:has-one,join:id=organization_id"`
	State           *usstate.UsState `json:"state,omitempty"           bun:"rel:belongs-to,join:state_id=id"`
}

func (o *Organization) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(o,
		validation.Field(&o.Name,
			validation.Required.Error("Name is required. Please try again"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters")),
		validation.Field(&o.ScacCode,
			validation.Required.Error("SCAC code is required. Please try again"),
			validation.Length(4, 4).Error("SCAC code must be 4 characters")),
		validation.Field(&o.DOTNumber,
			validation.Required.Error("DOT number is required. Please try again"),
			validation.Length(1, 8).Error("DOT number must be between 1 and 8 characters"),
			is.Digit.Error("DOT number must be numeric")),
		validation.Field(
			&o.AddressLine1,
			validation.Length(0, 150).
				Error("Address line 1 must be less than 150 characters. Please try again"),
		),
		validation.Field(&o.City,
			validation.Required.Error("City is required. Please try again")),
		validation.Field(&o.Timezone,
			validation.Required.Error("Timezone is required. Please try again"),
			validation.By(domain.ValidateTimezone)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (o *Organization) GetTableName() string {
	return "organizations"
}

func (o *Organization) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("org_")
		}

		o.CreatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}
