// Package rateagreement models the commercial contracts a shipment is priced
// under — what a customer is charged, and what a carrier is paid.
//
// An agreement is a header of negotiated terms plus a set of rules, where each
// rule is one priced lane. The header is versioned; the rules carry their own
// effective windows. That split is deliberate. Tariffs are amended a few lanes
// at a time — "these forty lanes take a three percent increase on the first" —
// and versioning the whole agreement would copy every untouched rule to record
// a change to a handful. Dating the rules instead makes an amendment a matter
// of closing out the rows that changed and inserting their replacements, which
// is also exactly what a rate sheet import and a simulated rate change do.
package rateagreement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*RateAgreement)(nil)
	_ domaintypes.PostgresSearchable     = (*RateAgreement)(nil)
)

const (
	maxCodeLength        = 50
	maxNameLength        = 150
	maxDescriptionLength = 2000
	maxContractRefLength = 100
)

// Negotiated customer and carrier contracts
type RateAgreement struct {
	keel.Table `keel:"rate_agreements,report=Billing"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rag_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	PartyType  PartyType `json:"partyType"  keel:"party_type"`
	CustomerID *pulid.ID `json:"customerId" keel:"customer_id"`
	CarrierID  *pulid.ID `json:"carrierId"  keel:"carrier_id"`

	Code          string        `json:"code"          keel:"code,label=Agreement Code"`
	Name          string        `json:"name"          keel:"name,label=Agreement Name"`
	Description   string        `json:"description"   keel:"description,nullzero"`
	AgreementType AgreementType `json:"agreementType" keel:"agreement_type"`
	Status        Status        `json:"status"        keel:"status"`
	ContractRef   string        `json:"contractRef"   keel:"contract_ref,nullzero,label=Contract Reference"`
	DocumentID    *pulid.ID     `json:"documentId"    keel:"document_id"`

	Priority          int16  `json:"priority"          keel:"priority,dimension"`
	EffectiveFrom     int64  `json:"effectiveFrom"     keel:"effective_from,type=epoch"`
	EffectiveTo       *int64 `json:"effectiveTo"       keel:"effective_to,type=epoch"`
	AutoRenew         bool   `json:"autoRenew"         keel:"auto_renew"`
	RenewalNoticeDays int16  `json:"renewalNoticeDays" keel:"renewal_notice_days,label=Renewal Notice (days),format=count,dimension"`

	Currency string             `json:"currency" keel:"currency"`
	Inputs   tarifftypes.Inputs `json:"inputs"   keel:"inputs,jsonb"`

	BillToCustomerID *pulid.ID `json:"billToCustomerId" keel:"bill_to_customer_id"`

	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent"  keel:"margin_floor_percent,label=Margin Floor (%)"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" keel:"max_pay_percent_of_sell,label=Max Pay % of Sell"`

	SubmittedByID *pulid.ID `json:"submittedById" keel:"submitted_by_id"`
	SubmittedAt   *int64    `json:"submittedAt"   keel:"submitted_at"`
	ApprovedByID  *pulid.ID `json:"approvedById"  keel:"approved_by_id"`
	ApprovedAt    *int64    `json:"approvedAt"    keel:"approved_at"`
	ReviewComment string    `json:"reviewComment" keel:"review_comment,nullzero"`

	CurrentVersionNumber int64  `json:"currentVersionNumber" keel:"current_version_number,label=Current Version,dimension"`
	Version              int64  `json:"version"              keel:"version,version"`
	CreatedAt            int64  `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64  `json:"updatedAt"            keel:"updated_at,autoupdate"`
	SearchVector         string `json:"-"                    keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string `json:"-"                    keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	Customer     *customer.Customer   `json:"customer,omitempty" keel:"belongs:customer_id"`
	Carrier      *carrier.Carrier     `json:"carrier,omitempty"  keel:"belongs:carrier_id"`

	Rules          []*RateAgreementRule        `json:"rules,omitempty"          keel:"many:rate_agreement_id,label=Lane Rules"`
	Accessorials   []*RateAgreementAccessorial `json:"accessorials,omitempty"   keel:"many:rate_agreement_id,label=Accessorial Schedule"`
	Versions       []*RateAgreementVersion     `json:"versions,omitempty"       keel:"many:rate_agreement_id"`
	FuelBinding    *RateAgreementFuelBinding   `json:"fuelBinding,omitempty"    keel:"one:rate_agreement_id,noreport"`
	CurrentVersion *RateAgreementVersion       `json:"currentVersion,omitempty" keel:"pick:versions,last=versions.version_number"`
}

func (ra *RateAgreement) BeforeUpdate(_ context.Context) error {
	ra.applyDefaults()

	return nil
}

// PartyID is whichever of the customer or carrier this agreement is with.
func (ra *RateAgreement) PartyID() pulid.ID {
	switch ra.PartyType {
	case PartyTypeCustomer:
		if ra.CustomerID != nil {
			return *ra.CustomerID
		}
	case PartyTypeCarrier:
		if ra.CarrierID != nil {
			return *ra.CarrierID
		}
	}

	return pulid.Nil
}

// IsEffectiveAt reports whether the agreement's own window covers a moment.
//
// The window is half open, matching every other date range in the rating
// domain: an agreement that runs to the thirty-first stops pricing at the
// instant the first begins, so a replacement can start without an overlap.
func (ra *RateAgreement) IsEffectiveAt(timestamp int64) bool {
	if timestamp < ra.EffectiveFrom {
		return false
	}

	if ra.EffectiveTo != nil && timestamp >= *ra.EffectiveTo {
		return false
	}

	return true
}

// RatesAt reports whether the agreement may price a shipment at this moment,
// which needs both an active status and a live window.
func (ra *RateAgreement) RatesAt(timestamp int64) bool {
	return ra.Status.RatesShipments() && ra.IsEffectiveAt(timestamp)
}

func (ra *RateAgreement) applyDefaults() {
	if ra.Status == "" {
		ra.Status = StatusDraft
	}
	if ra.AgreementType == "" {
		ra.AgreementType = AgreementTypeContract
	}
	if ra.Currency == "" {
		ra.Currency = money.DefaultCurrencyCode
	}
	ra.Inputs = ra.Inputs.OrEmpty()
	if ra.CurrentVersionNumber == 0 {
		ra.CurrentVersionNumber = 1
	}
}

func (ra *RateAgreement) Validate(multiErr *errortypes.MultiError) {
	ra.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(ra,
		validation.Field(&ra.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxCodeLength).
				Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&ra.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name must be between 1 and 150 characters"),
		),
		validation.Field(&ra.Description,
			validation.Length(0, maxDescriptionLength).
				Error("Description cannot be longer than 2000 characters"),
		),
		validation.Field(&ra.ContractRef,
			validation.Length(0, maxContractRefLength).
				Error("Contract reference cannot be longer than 100 characters"),
		),
		validation.Field(&ra.PartyType,
			validation.Required.Error("Party type is required"),
			domainvalidation.ValidEnum[PartyType]("Party type is invalid"),
		),
		validation.Field(&ra.AgreementType,
			validation.Required.Error("Agreement type is required"),
			domainvalidation.ValidEnum[AgreementType]("Agreement type is invalid"),
		),
		validation.Field(&ra.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&ra.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&ra.EffectiveFrom,
			validation.Required.Error("Effective from is required"),
		),
	))

	ra.validateParty(multiErr)
	ra.validateWindow(multiErr)
	ra.validateGuardrails(multiErr)
	ra.validateChildren(multiErr)
}

// validateParty keeps the discriminator and the two foreign keys honest. A
// customer agreement holding a carrier id would resolve against neither party
// and price nothing, which is a much harder failure to diagnose later than a
// rejected save now.
func (ra *RateAgreement) validateParty(multiErr *errortypes.MultiError) {
	hasCustomer := ra.CustomerID != nil && !ra.CustomerID.IsNil()
	hasCarrier := ra.CarrierID != nil && !ra.CarrierID.IsNil()

	switch ra.PartyType {
	case PartyTypeCustomer:
		if !hasCustomer {
			multiErr.Add("customerId", errortypes.ErrRequired, "Customer is required")
		}
		if hasCarrier {
			multiErr.Add(
				"carrierId",
				errortypes.ErrInvalid,
				"A customer agreement cannot name a carrier",
			)
		}
		if ra.MarginFloorPercent.Valid || ra.MaxPayPercentOfSell.Valid {
			multiErr.Add(
				"marginFloorPercent",
				errortypes.ErrInvalid,
				"Margin guardrails belong on a carrier agreement",
			)
		}
	case PartyTypeCarrier:
		if !hasCarrier {
			multiErr.Add("carrierId", errortypes.ErrRequired, "Carrier is required")
		}
		if hasCustomer {
			multiErr.Add(
				"customerId",
				errortypes.ErrInvalid,
				"A carrier agreement cannot name a customer",
			)
		}
		if ra.BillToCustomerID != nil && !ra.BillToCustomerID.IsNil() {
			multiErr.Add(
				"billToCustomerId",
				errortypes.ErrInvalid,
				"A bill-to customer belongs on a customer agreement",
			)
		}
	}
}

func (ra *RateAgreement) validateWindow(multiErr *errortypes.MultiError) {
	if ra.EffectiveTo != nil && *ra.EffectiveTo <= ra.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to must be after effective from",
		)
	}

	if ra.RenewalNoticeDays < 0 {
		multiErr.Add(
			"renewalNoticeDays",
			errortypes.ErrInvalid,
			"Renewal notice days cannot be negative",
		)
	}
}

func (ra *RateAgreement) validateGuardrails(multiErr *errortypes.MultiError) {
	validatePercent(multiErr, "marginFloorPercent", ra.MarginFloorPercent, "Margin floor")
	validatePercent(multiErr, "maxPayPercentOfSell", ra.MaxPayPercentOfSell, "Maximum pay")
}

func validatePercent(
	multiErr *errortypes.MultiError,
	field string,
	value decimal.NullDecimal,
	label string,
) {
	if !value.Valid {
		return
	}

	if value.Decimal.IsNegative() {
		multiErr.Add(field, errortypes.ErrInvalid, "{0} cannot be negative", label)
	}
}

// StampRules re-seats on every rule the values it inherits from its agreement:
// the tenancy, the parent, and the party the resolution index narrows by. None
// of them are the caller's to supply, and a rule carrying a stale party would
// be looked up under a customer the contract is not with.
func (ra *RateAgreement) StampRules() {
	partyID := ra.PartyID()

	for _, rule := range ra.Rules {
		if rule == nil {
			continue
		}

		rule.RateAgreementID = ra.ID
		rule.OrganizationID = ra.OrganizationID
		rule.BusinessUnitID = ra.BusinessUnitID
		rule.PartyType = ra.PartyType
		rule.PartyID = partyID
	}
}

func (ra *RateAgreement) validateChildren(multiErr *errortypes.MultiError) {
	ra.StampRules()

	for i, rule := range ra.Rules {
		if rule == nil {
			continue
		}
		rule.ValidateWithin(ra, multiErr.WithIndex("rules", i))
	}

	seenAccessorials := make(map[pulid.ID]struct{}, len(ra.Accessorials))
	for i, accessorial := range ra.Accessorials {
		if accessorial == nil {
			continue
		}

		accessorialErr := multiErr.WithIndex("accessorials", i)
		accessorial.Validate(accessorialErr)

		// Two schedule rows for the same accessorial would each try to price
		// the same charge, and which one won would depend on row order.
		if _, dup := seenAccessorials[accessorial.AccessorialChargeID]; dup {
			accessorialErr.Add(
				"accessorialChargeId",
				errortypes.ErrDuplicate,
				"This accessorial is already priced by the agreement",
			)
		}
		seenAccessorials[accessorial.AccessorialChargeID] = struct{}{}
	}

	if ra.FuelBinding != nil {
		ra.FuelBinding.Validate(multiErr.WithPrefix("fuelBinding"))
	}
}

func (ra *RateAgreement) BeforeCreate(ctx context.Context) error {
	if ra.AgreementType == "" {
		ra.AgreementType = AgreementTypeContract
	}
	if ra.Status == "" {
		ra.Status = StatusDraft
	}
	if ra.RenewalNoticeDays == 0 {
		ra.RenewalNoticeDays = 30
	}
	if ra.Currency == "" {
		ra.Currency = "USD"
	}
	if ra.CurrentVersionNumber == 0 {
		ra.CurrentVersionNumber = 1
	}
	ra.applyDefaults()

	return nil
}

func (ra *RateAgreement) BeforeSave(_ context.Context) error {
	ra.applyDefaults()
	return nil
}

func (ra *RateAgreement) GetID() pulid.ID {
	return ra.ID
}

func (ra *RateAgreement) GetCreatedAt() int64 {
	return ra.CreatedAt
}

func (ra *RateAgreement) GetOrganizationID() pulid.ID {
	return ra.OrganizationID
}

func (ra *RateAgreement) GetBusinessUnitID() pulid.ID {
	return ra.BusinessUnitID
}

func (ra *RateAgreement) GetVersion() int64 {
	return ra.Version
}

func (ra *RateAgreement) GetTableName() string {
	return "rate_agreements"
}

func (ra *RateAgreement) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rag",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "contract_ref",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
			{Name: "party_type", Type: domaintypes.FieldTypeEnum},
			{Name: "agreement_type", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (RateAgreement) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_rate_agreements_code",
			Message: "A rate agreement with this code already exists in your organization",
		},
	}
}
