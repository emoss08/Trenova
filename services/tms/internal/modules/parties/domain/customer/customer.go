package customer

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/addressutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var digitsPattern = regexp.MustCompile(`^[0-9]*$`)

var (
	_ validationframework.TenantedEntity = (*Customer)(nil)
	_ domaintypes.PostgresSearchable     = (*Customer)(nil)
)

// Customers who tender freight
type Customer struct {
	keel.Table `keel:"customers,report=Parties"`

	ID                      pulid.ID               `json:"id"                      keel:"id,pk,autoid:cus_"`
	BusinessUnitID          pulid.ID               `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID               `json:"organizationId"          keel:"organization_id,pk,tenant"`
	StateID                 pulid.ID               `json:"stateId"                 keel:"state_id"`
	Status                  domaintypes.Status     `json:"status"                  keel:"status"`
	Code                    string                 `json:"code"                    keel:"code"`
	Name                    string                 `json:"name"                    keel:"name,nullzero"`
	AddressLine1            string                 `json:"addressLine1"            keel:"address_line_1,nullzero"`
	AddressLine2            string                 `json:"addressLine2"            keel:"address_line_2,nullzero"`
	City                    string                 `json:"city"                    keel:"city,nullzero"`
	PostalCode              string                 `json:"postalCode"              keel:"postal_code"`
	IsGeocoded              bool                   `json:"isGeocoded"              keel:"is_geocoded"`
	Longitude               *float64               `json:"longitude"               keel:"longitude,dimension"`
	Latitude                *float64               `json:"latitude"                keel:"latitude,dimension"`
	PlaceID                 string                 `json:"placeId"                 keel:"place_id,nullzero"`
	ExternalID              string                 `json:"externalId"              keel:"external_id,nullzero"`
	DOTNumber               string                 `json:"dotNumber"               keel:"dot_number,nullzero"`
	MCNumber                string                 `json:"mcNumber"                keel:"mc_number,nullzero"`
	StatusUpdatePreference  StatusUpdatePreference `json:"statusUpdatePreference"  keel:"status_update_preference"`
	StatusUpdateRecipients  string                 `json:"statusUpdateRecipients"  keel:"status_update_recipients,nullzero"`
	BrokerVettingEnabled    bool                   `json:"brokerVettingEnabled"    keel:"broker_vetting_enabled"`
	CheckCallIntervalHours  *int16                 `json:"checkCallIntervalHours"  keel:"check_call_interval_hours,dimension"`
	EtaRiskThresholdMinutes *int16                 `json:"etaRiskThresholdMinutes" keel:"eta_risk_threshold_minutes,dimension"`
	Geom                    *postgis.Point         `json:"-"                       keel:"geom,readonly"`
	AllowConsolidation      bool                   `json:"allowConsolidation"      keel:"allow_consolidation"`
	ExclusiveConsolidation  bool                   `json:"exclusiveConsolidation"  keel:"exclusive_consolidation"`
	ConsolidationPriority   int                    `json:"consolidationPriority"   keel:"consolidation_priority,dimension"`
	SearchVector            string                 `json:"-"                       keel:"search_vector,tsvector,readonly,noselect"`
	Rank                    string                 `json:"-"                       keel:"rank,computed"`
	Version                 int64                  `json:"version"                 keel:"version,version"`
	CreatedAt               int64                  `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64                  `json:"updatedAt"               keel:"updated_at,autoupdate"`

	BusinessUnit   *tenant.BusinessUnit    `json:"businessUnit,omitempty"`
	Organization   *tenant.Organization    `json:"organization,omitempty"`
	BillingProfile *CustomerBillingProfile `json:"billingProfile,omitempty" keel:"one:customer_id,noreport"`
	EmailProfile   *CustomerEmailProfile   `json:"emailProfile,omitempty"   keel:"one:customer_id,noreport"`
	State          *usstate.UsState        `json:"state,omitempty"          keel:"belongs:state_id,noreport"`
}

func (c *Customer) BeforeUpdate(_ context.Context) error {
	// An older row, or a request that never knew about the field, is a
	// customer who has not opted in — which is the safe reading.
	if !c.StatusUpdatePreference.IsValid() {
		c.StatusUpdatePreference = StatusUpdateNone
	}

	return nil
}

func (c *Customer) BeforeCreate(_ context.Context) error {
	// An older row, or a request that never knew about the field, is a
	// customer who has not opted in — which is the safe reading.
	if !c.StatusUpdatePreference.IsValid() {
		c.StatusUpdatePreference = StatusUpdateNone
	}
	if c.Status == "" {
		c.Status = domaintypes.StatusActive
	}
	if c.StatusUpdatePreference == "" {
		c.StatusUpdatePreference = StatusUpdateNone
	}
	if c.ConsolidationPriority == 0 {
		c.ConsolidationPriority = 1
	}

	return nil
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
		validation.Field(&c.CheckCallIntervalHours,
			validation.When(
				c.CheckCallIntervalHours != nil,
				validation.Min(int16(1)).
					Error("Check call interval must be between 1 and 72 hours"),
				validation.Max(int16(72)).
					Error("Check call interval must be between 1 and 72 hours"),
			),
		),
		validation.Field(&c.EtaRiskThresholdMinutes,
			validation.When(
				c.EtaRiskThresholdMinutes != nil,
				validation.Min(int16(0)).
					Error("ETA risk threshold must be between 0 and 1440 minutes"),
				validation.Max(int16(1440)).
					Error("ETA risk threshold must be between 0 and 1440 minutes"),
			),
		),
	))

	if c.BillingProfile != nil {
		c.BillingProfile.Validate(multiErr.WithPrefix("billingProfile"))
	}

	if c.EmailProfile != nil {
		c.EmailProfile.ValidateForDelivery(
			multiErr.WithPrefix("emailProfile"),
			c.BillingProfile == nil || c.BillingProfile.EmailInvoiceEnabled,
		)
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

func (c *Customer) BillingCurrencyCode() string {
	if c == nil || c.BillingProfile == nil ||
		strings.TrimSpace(c.BillingProfile.BillingCurrency) == "" {
		return money.DefaultCurrencyCode
	}

	return c.BillingProfile.BillingCurrency
}

// DisplayName is how a message to a person names the customer: its name, or
// its code when it has none.
func (c *Customer) DisplayName() string {
	if c == nil {
		return ""
	}
	if name := strings.TrimSpace(c.Name); name != "" {
		return name
	}
	return c.Code
}

// MailingAddress is the customer's address as a document prints it.
func (c *Customer) MailingAddress() addressutils.Mailing {
	if c == nil {
		return addressutils.Mailing{}
	}
	m := addressutils.Mailing{
		Line1:      c.AddressLine1,
		Line2:      c.AddressLine2,
		City:       c.City,
		PostalCode: c.PostalCode,
	}
	if c.State != nil {
		m.State = c.State.Abbreviation
		m.Country = c.State.CountryName
	}
	return m
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
		FuzzyFields:     []domaintypes.FuzzyField{{"name"}},
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

func (c *Customer) BeforeSave(context.Context) error {
	if !c.StatusUpdatePreference.IsValid() {
		c.StatusUpdatePreference = StatusUpdateNone
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

func (Customer) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_customers_code",
			Message: "Customer with this code already exists in your organization",
		},
	}
}
