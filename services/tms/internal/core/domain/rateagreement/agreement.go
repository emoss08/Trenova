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

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateAgreement)(nil)
	_ validationframework.TenantedEntity = (*RateAgreement)(nil)
	_ domaintypes.PostgresSearchable     = (*RateAgreement)(nil)
)

const (
	maxCodeLength        = 50
	maxNameLength        = 150
	maxDescriptionLength = 2000
	maxContractRefLength = 100
	maxRoundingPrecision = 6
	hundredPercent       = 100
)

type RateAgreement struct {
	bun.BaseModel             `bun:"table:rate_agreements,alias:rag" json:"-"`
	pagination.CursorValueSet `bun:",embed"                          json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	PartyType  PartyType `json:"partyType"  bun:"party_type,type:rate_agreement_party_type_enum,notnull"`
	CustomerID *pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero"`
	CarrierID  *pulid.ID `json:"carrierId"  bun:"carrier_id,type:VARCHAR(100),nullzero"`

	Code          string        `json:"code"          bun:"code,type:VARCHAR(50),notnull"`
	Name          string        `json:"name"          bun:"name,type:VARCHAR(150),notnull"`
	Description   string        `json:"description"   bun:"description,type:TEXT,nullzero"`
	AgreementType AgreementType `json:"agreementType" bun:"agreement_type,type:rate_agreement_type_enum,notnull,default:'Contract'"`
	Status        Status        `json:"status"        bun:"status,type:rate_agreement_status_enum,notnull,default:'Draft'"`
	ContractRef   string        `json:"contractRef"   bun:"contract_ref,type:VARCHAR(100),nullzero"`
	DocumentID    *pulid.ID     `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`

	Priority          int16  `json:"priority"      bun:"priority,type:SMALLINT,notnull,default:0"`
	EffectiveFrom     int64  `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo       *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero"`
	AutoRenew         bool   `json:"autoRenew"     bun:"auto_renew,type:BOOLEAN,notnull,default:false"`
	RenewalNoticeDays int16  `json:"renewalNoticeDays" bun:"renewal_notice_days,type:SMALLINT,notnull,default:30"`

	Currency          string                 `json:"currency"          bun:"currency,type:VARCHAR(3),notnull,default:'USD'"`
	DefaultMinCharge  decimal.NullDecimal    `json:"defaultMinCharge"  bun:"default_min_charge,type:NUMERIC(19,4),nullzero"`
	DefaultMaxCharge  decimal.NullDecimal    `json:"defaultMaxCharge"  bun:"default_max_charge,type:NUMERIC(19,4),nullzero"`
	RoundingMode      ratetypes.RoundingMode `json:"roundingMode"      bun:"rounding_mode,type:rate_rounding_mode_enum,notnull,default:'HalfUp'"`
	RoundingPrecision int16                  `json:"roundingPrecision" bun:"rounding_precision,type:SMALLINT,notnull,default:2"`

	// BillToCustomerID redirects invoicing on a customer agreement. It is only
	// meaningful on the sell side and a check constraint keeps it there.
	BillToCustomerID *pulid.ID `json:"billToCustomerId" bun:"bill_to_customer_id,type:VARCHAR(100),nullzero"`

	// MarginFloorPercent and MaxPayPercentOfSell are the buy side guardrails a
	// brokerage sets against a carrier, and are likewise constrained to it.
	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent"  bun:"margin_floor_percent,type:NUMERIC(9,4),nullzero"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" bun:"max_pay_percent_of_sell,type:NUMERIC(9,4),nullzero"`

	SubmittedByID *pulid.ID `json:"submittedById" bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt   *int64    `json:"submittedAt"   bun:"submitted_at,type:BIGINT,nullzero"`
	ApprovedByID  *pulid.ID `json:"approvedById"  bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt    *int64    `json:"approvedAt"    bun:"approved_at,type:BIGINT,nullzero"`
	ReviewComment string    `json:"reviewComment" bun:"review_comment,type:TEXT,nullzero"`

	CurrentVersionNumber int64  `json:"currentVersionNumber" bun:"current_version_number,type:BIGINT,notnull,default:1"`
	Version              int64  `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64  `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64  `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector         string `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-"                  bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                  bun:"rel:belongs-to,join:organization_id=id"`
	Customer     *customer.Customer   `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id"`
	Carrier      *carrier.Carrier     `json:"carrier,omitempty"  bun:"rel:belongs-to,join:carrier_id=id"`

	Rules        []*RateAgreementRule        `json:"rules,omitempty"        bun:"rel:has-many,join:id=rate_agreement_id"`
	Accessorials []*RateAgreementAccessorial `json:"accessorials,omitempty" bun:"rel:has-many,join:id=rate_agreement_id"`
	Versions     []*RateAgreementVersion     `json:"versions,omitempty"     bun:"rel:has-many,join:id=rate_agreement_id"`
	FuelBinding  *RateAgreementFuelBinding   `json:"fuelBinding,omitempty"  bun:"rel:has-one,join:id=rate_agreement_id"`
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
	if ra.RoundingMode == "" {
		ra.RoundingMode = ratetypes.RoundingModeHalfUp
	}
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
		validation.Field(&ra.RoundingMode,
			validation.Required.Error("Rounding mode is required"),
			domainvalidation.ValidEnum[ratetypes.RoundingMode]("Rounding mode is invalid"),
		),
		validation.Field(&ra.RoundingPrecision,
			validation.Min(int16(0)).Error("Rounding precision cannot be negative"),
			validation.Max(int16(maxRoundingPrecision)).
				Error("Rounding precision cannot exceed 6"),
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
	if ra.DefaultMinCharge.Valid && ra.DefaultMinCharge.Decimal.IsNegative() {
		multiErr.Add(
			"defaultMinCharge",
			errortypes.ErrInvalid,
			"Default minimum charge cannot be negative",
		)
	}

	if ra.DefaultMaxCharge.Valid && ra.DefaultMaxCharge.Decimal.IsNegative() {
		multiErr.Add(
			"defaultMaxCharge",
			errortypes.ErrInvalid,
			"Default maximum charge cannot be negative",
		)
	}

	// A ceiling below the floor clamps every charge twice and leaves a priced
	// shipment nobody can explain.
	if ra.DefaultMinCharge.Valid && ra.DefaultMaxCharge.Valid &&
		ra.DefaultMaxCharge.Decimal.LessThan(ra.DefaultMinCharge.Decimal) {
		multiErr.Add(
			"defaultMaxCharge",
			errortypes.ErrInvalid,
			"Default maximum charge cannot be less than the default minimum charge",
		)
	}

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
		multiErr.Add(field, errortypes.ErrInvalid, label+" cannot be negative")
	}
}

func (ra *RateAgreement) validateChildren(multiErr *errortypes.MultiError) {
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

func (ra *RateAgreement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	ra.applyDefaults()

	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ra.ID.IsNil() {
			ra.ID = pulid.MustNew("rag_")
		}
		ra.CreatedAt = now
		ra.UpdatedAt = now
	case *bun.UpdateQuery:
		ra.UpdatedAt = now
	}

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
