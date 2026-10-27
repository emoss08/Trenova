package carrier

import (
	"context"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	numericPattern = regexp.MustCompile(`^[0-9]*$`)
	scacPattern    = regexp.MustCompile(`^[A-Z]{2,4}$`)
)

var (
	_ validationframework.TenantedEntity = (*Carrier)(nil)
	_ domaintypes.PostgresSearchable     = (*Carrier)(nil)
)

// External carriers that cover brokered freight
type Carrier struct {
	keel.Table `keel:"carriers,report=Parties"`

	ID             pulid.ID  `json:"id"             keel:"id,pk,autoid:car_"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	StateID        *pulid.ID `json:"stateId"        keel:"state_id"`
	RemitStateID   *pulid.ID `json:"remitStateId"   keel:"remit_state_id"`

	Status              Status           `json:"status"              keel:"status"`
	Code                string           `json:"code"                keel:"code"`
	Name                string           `json:"name"                keel:"name"`
	DBAName             string           `json:"dbaName"             keel:"dba_name,nullzero"`
	CarrierType         Type             `json:"carrierType"         keel:"carrier_type"`
	DOTNumber           string           `json:"dotNumber"           keel:"dot_number,nullzero"`
	MCNumber            string           `json:"mcNumber"            keel:"mc_number,nullzero"`
	SCAC                string           `json:"scac"                keel:"scac,nullzero"`
	ComplianceStatus    ComplianceStatus `json:"complianceStatus"    keel:"compliance_status"`
	SafetyRating        SafetyRating     `json:"safetyRating"        keel:"safety_rating"`
	QualifiedAt         *int64           `json:"qualifiedAt"         keel:"qualified_at"`
	DisqualifiedReason  string           `json:"disqualifiedReason"  keel:"disqualified_reason,nullzero"`
	TaxID               string           `json:"taxId"               keel:"tax_id,nullzero"`
	TaxIDType           *TaxIDType       `json:"taxIdType"           keel:"tax_id_type"`
	W9OnFile            bool             `json:"w9OnFile"            keel:"w9_on_file,label=W-9 On File"`
	Is1099Eligible      bool             `json:"is1099Eligible"      keel:"is_1099_eligible,label=1099 Eligible"`
	PaymentMethod       PaymentMethod    `json:"paymentMethod"       keel:"payment_method"`
	PaymentTermDays     int              `json:"paymentTermDays"     keel:"payment_term_days,dimension"`
	RemitToName         string           `json:"remitToName"         keel:"remit_to_name,nullzero"`
	RemitAddressLine1   string           `json:"remitAddressLine1"   keel:"remit_address_line_1,nullzero"`
	RemitAddressLine2   string           `json:"remitAddressLine2"   keel:"remit_address_line_2,nullzero"`
	RemitCity           string           `json:"remitCity"           keel:"remit_city,nullzero"`
	RemitPostalCode     string           `json:"remitPostalCode"     keel:"remit_postal_code,nullzero"`
	AddressLine1        string           `json:"addressLine1"        keel:"address_line_1,nullzero"`
	AddressLine2        string           `json:"addressLine2"        keel:"address_line_2,nullzero"`
	City                string           `json:"city"                keel:"city,nullzero"`
	PostalCode          string           `json:"postalCode"          keel:"postal_code,nullzero"`
	Phone               string           `json:"phone"               keel:"phone,nullzero"`
	Email               string           `json:"email"               keel:"email,nullzero"`
	ExternalID          string           `json:"externalId"          keel:"external_id,nullzero"`
	Notes               string           `json:"notes"               keel:"notes,nullzero"`
	IntelRiskLevel      string           `json:"intelRiskLevel"      keel:"intel_risk_level,nullzero"`
	IntelReviewRequired bool             `json:"intelReviewRequired" keel:"intel_review_required"`
	IntelBlockingCount  int              `json:"intelBlockingCount"  keel:"intel_blocking_count"`
	SearchVector        string           `json:"-"                   keel:"search_vector,tsvector,readonly,noselect"`
	Rank                string           `json:"-"                   keel:"rank,computed"`
	Version             int64            `json:"version"             keel:"version,version"`
	CreatedAt           int64            `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64            `json:"updatedAt"           keel:"updated_at,autoupdate"`

	BusinessUnit      *tenant.BusinessUnit      `json:"businessUnit,omitempty"`
	Organization      *tenant.Organization      `json:"organization,omitempty"`
	State             *usstate.UsState          `json:"state,omitempty"             keel:"belongs:state_id,noreport"`
	RemitState        *usstate.UsState          `json:"remitState,omitempty"        keel:"belongs:remit_state_id,noreport"`
	Contacts          []*CarrierContact         `json:"contacts,omitempty"          keel:"many:carrier_id,noreport"`
	InsurancePolicies []*CarrierInsurancePolicy `json:"insurancePolicies,omitempty" keel:"many:carrier_id,noreport"`
	EDIChannels       []*CarrierEDIChannel      `json:"ediChannels,omitempty"       keel:"many:carrier_id,noreport"`
}

func (c *Carrier) BeforeCreate(_ context.Context) error {
	if c.Status == "" {
		c.Status = StatusActive
	}
	if c.CarrierType == "" {
		c.CarrierType = TypeCommon
	}
	if c.ComplianceStatus == "" {
		c.ComplianceStatus = ComplianceStatusPending
	}
	if c.SafetyRating == "" {
		c.SafetyRating = SafetyRatingNotRated
	}
	if c.PaymentMethod == "" {
		c.PaymentMethod = PaymentMethodCheck
	}
	if c.PaymentTermDays == 0 {
		c.PaymentTermDays = 30
	}

	return nil
}

func (c *Carrier) applyDefaults() {
	if c.Status == "" {
		c.Status = StatusActive
	}
	if c.CarrierType == "" {
		c.CarrierType = TypeCommon
	}
	if c.ComplianceStatus == "" {
		c.ComplianceStatus = ComplianceStatusPending
	}
	if c.SafetyRating == "" {
		c.SafetyRating = SafetyRatingNotRated
	}
	if c.PaymentMethod == "" {
		c.PaymentMethod = PaymentMethodCheck
	}
	if c.PaymentTermDays == 0 {
		c.PaymentTermDays = 30
	}
}

func (c *Carrier) Validate(multiErr *errortypes.MultiError) {
	c.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&c.CarrierType,
			validation.Required.Error("Carrier type is required"),
			domainvalidation.ValidEnum[Type]("Carrier type is invalid"),
		),
		validation.Field(&c.ComplianceStatus,
			validation.Required.Error("Compliance status is required"),
			domainvalidation.ValidEnum[ComplianceStatus]("Compliance status is invalid"),
		),
		validation.Field(&c.SafetyRating,
			validation.Required.Error("Safety rating is required"),
			domainvalidation.ValidEnum[SafetyRating]("Safety rating is invalid"),
		),
		validation.Field(&c.DOTNumber,
			validation.Length(0, 12).Error("DOT number must be at most 12 characters"),
			validation.Match(numericPattern).Error("DOT number must contain only digits"),
		),
		validation.Field(&c.MCNumber,
			validation.Length(0, 12).Error("MC number must be at most 12 characters"),
			validation.Match(numericPattern).Error("MC number must contain only digits"),
		),
		validation.Field(&c.SCAC,
			validation.Length(0, 4).Error("SCAC must be at most 4 characters"),
			validation.Match(scacPattern).Error("SCAC must be 2-4 uppercase letters"),
		),
		validation.Field(&c.TaxIDType,
			validation.When(c.TaxID != "",
				validation.Required.Error("Tax ID type is required when a tax ID is provided"),
			),
		),
		validation.Field(&c.PaymentMethod,
			validation.Required.Error("Payment method is required"),
			domainvalidation.ValidEnum[PaymentMethod]("Payment method is invalid"),
		),
		validation.Field(&c.PaymentTermDays,
			validation.Min(0).Error("Payment term days cannot be negative"),
			validation.Max(365).Error("Payment term days cannot exceed 365"),
		),
		validation.Field(&c.Email,
			validation.When(c.Email != "",
				is.EmailFormat.Error("Email must be a valid email address"),
			),
		),
		validation.Field(&c.DisqualifiedReason,
			validation.When(
				c.ComplianceStatus == ComplianceStatusDisqualified,
				validation.Required.Error(
					"A disqualified carrier must record the disqualification reason",
				),
			),
		),
	))

	if c.TaxIDType != nil && !c.TaxIDType.IsValid() {
		multiErr.Add("taxIdType", errortypes.ErrInvalid, "Tax ID type is invalid")
	}

	for idx, contact := range c.Contacts {
		contact.Validate(multiErr.WithIndex("contacts", idx))
	}

	for idx, policy := range c.InsurancePolicies {
		policy.Validate(multiErr.WithIndex("insurancePolicies", idx))
	}

	defaultChannels := 0
	for idx, channel := range c.EDIChannels {
		if channel == nil {
			continue
		}
		channel.Validate(multiErr.WithIndex("ediChannels", idx))
		if channel.IsDefault {
			defaultChannels++
		}
	}
	if defaultChannels > 1 {
		multiErr.Add(
			"ediChannels",
			errortypes.ErrInvalid,
			"Only one EDI channel can be marked as default",
		)
	}
}

func (c *Carrier) GetID() pulid.ID {
	return c.ID
}

func (c *Carrier) GetCreatedAt() int64 {
	return c.CreatedAt
}

func (c *Carrier) GetTableName() string {
	return "carriers"
}

func (c *Carrier) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *Carrier) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *Carrier) IsActive() bool {
	return c.Status == StatusActive
}

func (c *Carrier) IsQualified() bool {
	return c.ComplianceStatus == ComplianceStatusQualified
}

// RequiredPolicy returns the newest policy of the given type, preferring the one
// with the latest expiration so an overlapping renewal supersedes the old policy.
func (c *Carrier) RequiredPolicy(policyType InsurancePolicyType) *CarrierInsurancePolicy {
	var latest *CarrierInsurancePolicy
	for _, policy := range c.InsurancePolicies {
		if policy.PolicyType != policyType {
			continue
		}
		if latest == nil || policy.ExpirationDate > latest.ExpirationDate {
			latest = policy
		}
	}
	return latest
}

func (c *Carrier) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "carr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "dba_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "dot_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "scac", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (c *Carrier) SamePartyDetails(other *Carrier) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.Name == other.Name &&
		c.Email == other.Email &&
		c.Is1099Eligible == other.Is1099Eligible &&
		c.AddressLine1 == other.AddressLine1 &&
		c.City == other.City &&
		pulid.ConvertFromPtr(c.StateID) == pulid.ConvertFromPtr(other.StateID) &&
		c.PostalCode == other.PostalCode &&
		c.RemitAddressLine1 == other.RemitAddressLine1 &&
		c.RemitCity == other.RemitCity &&
		pulid.ConvertFromPtr(c.RemitStateID) == pulid.ConvertFromPtr(other.RemitStateID) &&
		c.RemitPostalCode == other.RemitPostalCode
}

func (Carrier) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_carriers_code",
			Message: "Carrier with this code already exists in your organization",
		},
		{
			Name:    "idx_carriers_dot_number",
			Message: "Carrier with this DOT number already exists in your organization",
		},
		{
			Name:    "idx_carriers_scac",
			Message: "Carrier with this SCAC already exists in your organization",
		},
	}
}
