package customer

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var digitsPattern = regexp.MustCompile(`^[0-9]*$`)

var (
	_ bun.BeforeAppendModelHook          = (*Customer)(nil)
	_ validationframework.TenantedEntity = (*Customer)(nil)
	_ domaintypes.PostgresSearchable     = (*Customer)(nil)
)

type Customer struct {
	bun.BaseModel `bun:"table:customers,alias:cus" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                     pulid.ID               `json:"id"                     bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID         pulid.ID               `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID               `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	StateID                pulid.ID               `json:"stateId"                bun:"state_id,type:VARCHAR(100),notnull"`
	Status                 domaintypes.Status     `json:"status"                 bun:"status,type:status_enum,notnull,default:'Active'"`
	Code                   string                 `json:"code"                   bun:"code,type:VARCHAR(10),notnull"`
	Name                   string                 `json:"name"                   bun:"name,type:VARCHAR(255),nullzero"`
	AddressLine1           string                 `json:"addressLine1"           bun:"address_line_1,type:VARCHAR(150),nullzero"`
	AddressLine2           string                 `json:"addressLine2"           bun:"address_line_2,type:VARCHAR(150),nullzero"`
	City                   string                 `json:"city"                   bun:"city,type:VARCHAR(100),nullzero"`
	PostalCode             string                 `json:"postalCode"             bun:"postal_code,type:us_postal_code,notnull"`
	IsGeocoded             bool                   `json:"isGeocoded"             bun:"is_geocoded,type:BOOLEAN"`
	Longitude              *float64               `json:"longitude"              bun:"longitude,type:FLOAT,nullzero"`
	Latitude               *float64               `json:"latitude"               bun:"latitude,type:FLOAT,nullzero"`
	PlaceID                string                 `json:"placeId"                bun:"place_id,type:TEXT,nullzero"`
	ExternalID             string                 `json:"externalId"             bun:"external_id,type:TEXT,nullzero"`
	DOTNumber              string                 `json:"dotNumber"              bun:"dot_number,type:VARCHAR(12),nullzero"`
	MCNumber               string                 `json:"mcNumber"               bun:"mc_number,type:VARCHAR(12),nullzero"`
	StatusUpdatePreference StatusUpdatePreference `json:"statusUpdatePreference" bun:"status_update_preference,type:VARCHAR(30),notnull,default:'None'"`
	StatusUpdateRecipients string                 `json:"statusUpdateRecipients" bun:"status_update_recipients,type:TEXT,nullzero"`
	BrokerVettingEnabled   bool                   `json:"brokerVettingEnabled"   bun:"broker_vetting_enabled,type:BOOLEAN,notnull"`
	Geom                   *postgis.Point         `json:"-"                      bun:"geom,type:geography,scanonly"`
	AllowConsolidation     bool                   `json:"allowConsolidation"     bun:"allow_consolidation,type:BOOLEAN"`
	ExclusiveConsolidation bool                   `json:"exclusiveConsolidation" bun:"exclusive_consolidation,type:BOOLEAN"`
	ConsolidationPriority  int                    `json:"consolidationPriority"  bun:"consolidation_priority,type:INTEGER,default:1"`
	SearchVector           string                 `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                   string                 `json:"-"                      bun:"rank,type:VARCHAR(100),scanonly"`
	Version                int64                  `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64                  `json:"createdAt"              bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                  `json:"updatedAt"              bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit   *tenant.BusinessUnit    `json:"businessUnit,omitempty"   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization    `json:"organization,omitempty"   bun:"rel:belongs-to,join:organization_id=id"`
	BillingProfile *CustomerBillingProfile `json:"billingProfile,omitempty" bun:"rel:has-one,join:id=customer_id"`
	EmailProfile   *CustomerEmailProfile   `json:"emailProfile,omitempty"   bun:"rel:has-one,join:id=customer_id"`
	State          *usstate.UsState        `json:"state,omitempty"          bun:"rel:belongs-to,join:state_id=id"`
}

// NormalizeStatusUpdates reads an unset preference as the customer not
// having opted in, which is what an older row and a request that predates
// the field both mean. Requiring it instead would fail every caller that
// has no opinion, for a field whose whole point is that silence means no.
func (c *Customer) NormalizeStatusUpdates() {
	if !c.StatusUpdatePreference.IsValid() {
		c.StatusUpdatePreference = StatusUpdateNone
	}
	if c.StatusUpdatePreference == StatusUpdateNone {
		c.StatusUpdateRecipients = ""
	}
}

func (c *Customer) Validate(multiErr *errortypes.MultiError) {
	c.NormalizeStatusUpdates()

	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(&c.StatusUpdateRecipients,
			validation.Length(0, 2000).Error(
				"Status update recipients cannot exceed 2000 characters",
			),
		),
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&c.AddressLine1,
			validation.Required.Error("Address line 1 is required"),
			validation.Length(1, 150).Error("Address line 1 must be between 1 and 150 characters"),
		),
		validation.Field(&c.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),
		validation.Field(&c.StateID,
			validation.Required.Error("State is required"),
		),
		validation.Field(
			&c.AllowConsolidation,
			validation.When(
				c.ExclusiveConsolidation,
				validation.Required.Error(
					"Allow consolidation is required when exclusive consolidation is true",
				),
			),
		),
		validation.Field(&c.PostalCode,
			validation.Required.Error("Postal code is required"),
			validation.By(domaintypes.ValidatePostalCode),
		),
		validation.Field(&c.DOTNumber,
			validation.Length(0, 12).Error("DOT number must be at most 12 characters"),
			validation.Match(digitsPattern).Error("DOT number must contain only digits"),
			validation.When(
				c.BrokerVettingEnabled,
				validation.Required.Error(
					"A DOT number is required to vet this customer as a broker",
				),
			),
		),
		validation.Field(&c.MCNumber,
			validation.Length(0, 12).Error("MC number must be at most 12 characters"),
			validation.Match(digitsPattern).Error("MC number must contain only digits"),
		),
	))

	if c.BillingProfile != nil {
		c.BillingProfile.Validate(multiErr.WithPrefix("billingProfile"))
	}

	if c.EmailProfile != nil {
		c.EmailProfile.Validate(multiErr.WithPrefix("emailProfile"))
	}
}

func (c *Customer) GetID() pulid.ID {
	return c.ID
}

func (c *Customer) GetCreatedAt() int64 {
	return c.CreatedAt
}

func (c *Customer) GetTableName() string {
	return "customers"
}

func (c *Customer) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *Customer) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *Customer) HasBillingProfile() bool {
	return c.BillingProfile != nil
}

func (c *Customer) HasEmailProfile() bool {
	return c.EmailProfile != nil
}

func (c *Customer) ResetGeocoding() {
	c.IsGeocoded = false
	c.Longitude = nil
	c.Latitude = nil
	c.PlaceID = ""
	c.Geom = nil
}

func (c *Customer) SetGeocoding(
	isGeocoded bool,
	longitude *float64,
	latitude *float64,
	placeID string,
) {
	c.IsGeocoded = isGeocoded
	c.Longitude = longitude
	c.Latitude = latitude
	c.PlaceID = placeID
}

func (c *Customer) MeetGeocodingRequirements() bool {
	return c.Longitude != nil && c.Latitude != nil && c.PlaceID != ""
}

func (c *Customer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "cus",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (c *Customer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	// An older row, or a request that never knew about the field, is a
	// customer who has not opted in — which is the safe reading.
	if !c.StatusUpdatePreference.IsValid() {
		c.StatusUpdatePreference = StatusUpdateNone
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cus_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *Customer) SamePartyDetails(other *Customer) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.Name == other.Name &&
		c.AddressLine1 == other.AddressLine1 &&
		c.City == other.City &&
		c.StateID == other.StateID &&
		c.PostalCode == other.PostalCode
}
