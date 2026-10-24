package tenant

import (
	"context"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/addressutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Metadata struct {
	ObjectID string `json:"objectId"`
}

const OrganizationIDPrefix = "org_"

type Organization struct {
	keel.Table `keel:"organizations"`

	ID                     pulid.ID  `json:"id" keel:"id,pk,autoid:org_"`
	StateID                pulid.ID  `json:"stateId" keel:"state_id"`
	BusinessUnitID         pulid.ID  `json:"businessUnitId" keel:"business_unit_id,tenant"`
	Name                   string    `json:"name" keel:"name"`
	LoginSlug              string    `json:"loginSlug" keel:"login_slug,nullzero"`
	ScacCode               string    `json:"scacCode" keel:"scac_code"`
	DOTNumber              string    `json:"dotNumber" keel:"dot_number"`
	LogoURL                string    `json:"logoUrl" keel:"logo_url,nullzero"`
	BucketName             string    `json:"bucketName" keel:"bucket_name"`
	AddressLine1           string    `json:"addressLine1" keel:"address_line1"`
	AddressLine2           string    `json:"addressLine2" keel:"address_line2,nullzero"`
	City                   string    `json:"city" keel:"city"`
	PostalCode             string    `json:"postalCode" keel:"postal_code"`
	Timezone               string    `json:"timezone" keel:"timezone"`
	Locale                 string    `json:"locale" keel:"locale"`
	TaxID                  string    `json:"taxId" keel:"tax_id,nullzero"`
	BrokerageEnabled       bool      `json:"brokerageEnabled" keel:"brokerage_enabled"`
	AssetOperationsEnabled bool      `json:"assetOperationsEnabled" keel:"asset_operations_enabled"`
	Metadata               *Metadata `json:"-" keel:"metadata,jsonb"`
	Version                int64     `json:"version" keel:"version,version"`
	CreatedAt              int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt              int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	// Relationships
	State        *usstate.UsState `json:"state" keel:"belongs:state_id"`
	BusinessUnit *BusinessUnit    `json:"businessUnit" keel:"belongs:business_unit_id"`
}

func (o *Organization) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		o,
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
		validation.Field(&o.Locale,
			validation.By(domainvalidation.ValidateLocale)),
	))
}

// MailingAddress is the organization's address as a document prints it.
func (o *Organization) MailingAddress() addressutils.Mailing {
	if o == nil {
		return addressutils.Mailing{}
	}
	m := addressutils.Mailing{
		Line1:      o.AddressLine1,
		Line2:      o.AddressLine2,
		City:       o.City,
		PostalCode: o.PostalCode,
	}
	if o.State != nil {
		m.State = o.State.Abbreviation
		m.Country = o.State.CountryName
	}
	return m
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

func (o *Organization) BeforeCreate(ctx context.Context) error {
	if o.Timezone == "" {
		o.Timezone = "America/New_York"
	}
	if o.Locale == "" {
		o.Locale = "en"
	}

	return nil
}
