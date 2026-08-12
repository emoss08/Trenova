package carrier

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	numericPattern = regexp.MustCompile(`^[0-9]*$`)
	scacPattern    = regexp.MustCompile(`^[A-Z]{2,4}$`)
)

var (
	_ bun.BeforeAppendModelHook          = (*Carrier)(nil)
	_ validationframework.TenantedEntity = (*Carrier)(nil)
	_ domaintypes.PostgresSearchable     = (*Carrier)(nil)
)

type Carrier struct {
	bun.BaseModel `bun:"table:carriers,alias:carr" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID  `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	StateID        *pulid.ID `json:"stateId"        bun:"state_id,type:VARCHAR(100),nullzero"`
	RemitStateID   *pulid.ID `json:"remitStateId"   bun:"remit_state_id,type:VARCHAR(100),nullzero"`

	Status             Status           `json:"status"              bun:"status,type:VARCHAR(50),notnull,default:'Active'"`
	Code               string           `json:"code"                bun:"code,type:VARCHAR(10),notnull"`
	Name               string           `json:"name"                bun:"name,type:VARCHAR(255),notnull"`
	DBAName            string           `json:"dbaName"             bun:"dba_name,type:VARCHAR(255),nullzero"`
	CarrierType        Type             `json:"carrierType"         bun:"carrier_type,type:VARCHAR(50),notnull,default:'Common'"`
	DOTNumber          string           `json:"dotNumber"           bun:"dot_number,type:VARCHAR(12),nullzero"`
	MCNumber           string           `json:"mcNumber"            bun:"mc_number,type:VARCHAR(12),nullzero"`
	SCAC               string           `json:"scac"                bun:"scac,type:VARCHAR(4),nullzero"`
	ComplianceStatus   ComplianceStatus `json:"complianceStatus"    bun:"compliance_status,type:VARCHAR(50),notnull,default:'Pending'"`
	SafetyRating       SafetyRating     `json:"safetyRating"        bun:"safety_rating,type:VARCHAR(50),notnull,default:'NotRated'"`
	QualifiedAt        *int64           `json:"qualifiedAt"         bun:"qualified_at,type:BIGINT,nullzero"`
	DisqualifiedReason string           `json:"disqualifiedReason"  bun:"disqualified_reason,type:TEXT,nullzero"`
	TaxID              string           `json:"taxId"               bun:"tax_id,type:VARCHAR(20),nullzero"`
	TaxIDType          *TaxIDType       `json:"taxIdType"           bun:"tax_id_type,type:VARCHAR(10),nullzero"`
	W9OnFile           bool             `json:"w9OnFile"            bun:"w9_on_file,type:BOOLEAN,notnull,default:false"`
	Is1099Eligible     bool             `json:"is1099Eligible"      bun:"is_1099_eligible,type:BOOLEAN,notnull,default:false"`
	PaymentMethod      PaymentMethod    `json:"paymentMethod"       bun:"payment_method,type:VARCHAR(50),notnull,default:'Check'"`
	PaymentTermDays    int              `json:"paymentTermDays"     bun:"payment_term_days,type:INTEGER,notnull,default:30"`
	RemitToName        string           `json:"remitToName"         bun:"remit_to_name,type:VARCHAR(255),nullzero"`
	RemitAddressLine1  string           `json:"remitAddressLine1"   bun:"remit_address_line_1,type:VARCHAR(150),nullzero"`
	RemitAddressLine2  string           `json:"remitAddressLine2"   bun:"remit_address_line_2,type:VARCHAR(150),nullzero"`
	RemitCity          string           `json:"remitCity"           bun:"remit_city,type:VARCHAR(100),nullzero"`
	RemitPostalCode    string           `json:"remitPostalCode"     bun:"remit_postal_code,type:VARCHAR(10),nullzero"`
	AddressLine1       string           `json:"addressLine1"        bun:"address_line_1,type:VARCHAR(150),nullzero"`
	AddressLine2       string           `json:"addressLine2"        bun:"address_line_2,type:VARCHAR(150),nullzero"`
	City               string           `json:"city"                bun:"city,type:VARCHAR(100),nullzero"`
	PostalCode         string           `json:"postalCode"          bun:"postal_code,type:VARCHAR(10),nullzero"`
	Phone              string           `json:"phone"               bun:"phone,type:VARCHAR(20),nullzero"`
	Email              string           `json:"email"               bun:"email,type:VARCHAR(255),nullzero"`
	ExternalID         string           `json:"externalId"          bun:"external_id,type:TEXT,nullzero"`
	Notes              string           `json:"notes"               bun:"notes,type:TEXT,nullzero"`
	SearchVector       string           `json:"-"                   bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank               string           `json:"-"                   bun:"rank,type:VARCHAR(100),scanonly"`
	Version            int64            `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt          int64            `json:"createdAt"           bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64            `json:"updatedAt"           bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit      `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization      `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	State             *usstate.UsState          `json:"state,omitempty"             bun:"rel:belongs-to,join:state_id=id"`
	RemitState        *usstate.UsState          `json:"remitState,omitempty"        bun:"rel:belongs-to,join:remit_state_id=id"`
	Contacts          []*CarrierContact         `json:"contacts,omitempty"          bun:"rel:has-many,join:id=carrier_id"`
	InsurancePolicies []*CarrierInsurancePolicy `json:"insurancePolicies,omitempty" bun:"rel:has-many,join:id=carrier_id"`
	EDIChannels       []*CarrierEDIChannel      `json:"ediChannels,omitempty"       bun:"rel:has-many,join:id=carrier_id"`
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

func (c *Carrier) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("car_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
