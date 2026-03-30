package tenant

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

type Metadata struct {
	ObjectID string `json:"objectId"`
}

var _ bun.BeforeAppendModelHook = (*Organization)(nil)

type Organization struct {
	bun.BaseModel `bun:"table:organizations,alias:org" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	StateID        pulid.ID  `json:"stateId"        bun:"state_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Name           string    `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	LoginSlug      string    `json:"loginSlug"      bun:"login_slug,type:VARCHAR(100)"`
	ScacCode       string    `json:"scacCode"       bun:"scac_code,type:VARCHAR(4),notnull"`
	DOTNumber      string    `json:"dotNumber"      bun:"dot_number,type:VARCHAR(8),notnull"`
	LogoURL        string    `json:"logoUrl"        bun:"logo_url,type:VARCHAR(255)"`
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
	State        *usstate.UsState `json:"state"        bun:"rel:belongs-to,join:state_id=id"`
	BusinessUnit *BusinessUnit    `json:"businessUnit" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (o *Organization) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(o,
		validation.Field(&o.Name,
			validation.Required.Error("Name is required. Please try again"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters")),
		validation.Field(&o.LoginSlug,
			validation.Length(0, 100).Error("Login slug must be between 1 and 100 characters"),
			validation.Match(regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)).
				Error("Login slug may only contain lowercase letters, numbers, and hyphens")),
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
			validation.By(domainvalidation.ValidateTimezone)),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (o *Organization) GetID() pulid.ID {
	return o.ID
}

func (o *Organization) GetTableName() string {
	return "organizations"
}

func (o *Organization) GetResourceID() string {
	return o.ID.String()
}

func (o *Organization) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

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
