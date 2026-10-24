// Package ratequote records what a rating decided and why.
//
// A quote is a table rather than a blob on the shipment for three reasons. It
// has to exist before a shipment does, for pre-booking pricing and rate
// shopping. It has to be queryable across shipments — which rules have never
// won, which lanes keep falling through to a fallback, what margin looks like
// by customer. And it has to accumulate: shipments are re-rated on create, on
// update, on every move edit, on assignment and on every fuel price job, and a
// column that gets overwritten each time destroys the history a dispute needs.
package ratequote

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*RateQuote)(nil)
)

// How each shipment was priced, what it cost, and what an override gave away
type RateQuote struct {
	keel.Table `                                           keel:"rate_quotes,report=Billing"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:rqt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ShipmentID     *pulid.ID `json:"shipmentId" keel:"shipment_id"`
	ShipmentMoveID *pulid.ID `json:"shipmentMoveId" keel:"shipment_move_id"`

	PartyType rateagreement.PartyType `json:"partyType" keel:"party_type"`
	PartyID   pulid.ID                `json:"partyId" keel:"party_id"`

	Purpose Purpose `json:"purpose" keel:"purpose"`
	Outcome Outcome `json:"outcome" keel:"outcome"`
	Status  Status  `json:"status" keel:"status"`

	RateAgreementID              *pulid.ID `json:"rateAgreementId" keel:"rate_agreement_id"`
	RateAgreementRuleID          *pulid.ID `json:"rateAgreementRuleId" keel:"rate_agreement_rule_id"`
	AgreementVersionNumber       *int64    `json:"agreementVersionNumber" keel:"agreement_version_number,label=Agreement Version,dimension"`
	FormulaTemplateID            *pulid.ID `json:"formulaTemplateId" keel:"formula_template_id"`
	FormulaTemplateVersionNumber *int64    `json:"formulaTemplateVersionNumber" keel:"formula_template_version_number,dimension"`
	SpecificityScore             int32     `json:"specificityScore" keel:"specificity_score,dimension"`

	Currency          string              `json:"currency" keel:"currency"`
	LinehaulAmount    decimal.Decimal     `json:"linehaulAmount" keel:"linehaul_amount,label=Linehaul,format=money"`
	FuelAmount        decimal.Decimal     `json:"fuelAmount" keel:"fuel_amount,label=Fuel,format=money"`
	AccessorialAmount decimal.Decimal     `json:"accessorialAmount" keel:"accessorial_amount,label=Accessorials,format=money"`
	TotalAmount       decimal.Decimal     `json:"totalAmount" keel:"total_amount,label=Total,format=money"`
	CostAmount        decimal.NullDecimal `json:"costAmount" keel:"cost_amount,label=Cost,format=money"`
	MarginAmount      decimal.NullDecimal `json:"marginAmount" keel:"margin_amount,label=Margin,format=money"`
	MarginPercent     decimal.NullDecimal `json:"marginPercent" keel:"margin_percent,label=Margin (%)"`

	BillingCurrency string              `json:"billingCurrency" keel:"billing_currency"`
	BillingAmount   decimal.Decimal     `json:"billingAmount" keel:"billing_amount,format=money"`
	FxRate          decimal.NullDecimal `json:"fxRate" keel:"fx_rate"`
	ExchangeRateID  *pulid.ID           `json:"exchangeRateId" keel:"exchange_rate_id"`

	AsOf          int64     `json:"asOf" keel:"as_of,type=epoch"`
	RatedAt       int64     `json:"ratedAt" keel:"rated_at"`
	RatedByID     *pulid.ID `json:"ratedById" keel:"rated_by_id"`
	EngineVersion string    `json:"engineVersion" keel:"engine_version"`

	ContextHash string `json:"contextHash" keel:"context_hash,noreport"`

	OverrideReason string              `json:"overrideReason" keel:"override_reason,nullzero"`
	ForegoneAmount decimal.NullDecimal `json:"foregoneAmount" keel:"foregone_amount,format=money"`

	Trace *ratetypes.Trace `json:"trace" keel:"trace,jsonb,noreport"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"`
	Organization *tenant.Organization         `json:"-"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" keel:"belongs:rate_agreement_id,label=Rate Agreement"`
	RatedBy      *tenant.User                 `json:"ratedBy,omitempty" keel:"belongs:rated_by_id,noreport"`
}

func (rq *RateQuote) BeforeUpdate(_ context.Context) error {
	rq.applyDefaults()

	return nil
}

// Explanation is the one line answer to "why this rate", assembled from what
// the quote already knows.
func (rq *RateQuote) Explanation() string {
	switch rq.Outcome {
	case OutcomeRated:
		if winner := rq.winner(); winner != nil && winner.AgreementName != "" {
			if winner.RuleLabel != "" {
				return winner.AgreementName + " — " + winner.RuleLabel
			}
			return winner.AgreementName
		}
		return "Priced from a rate agreement"
	case OutcomeFormulaFallback:
		return "No agreement covered this lane; priced from a formula template"
	case OutcomeManualOverride:
		if rq.OverrideReason != "" {
			return "Rate set by hand: " + rq.OverrideReason
		}
		return "Rate set by hand"
	case OutcomeNoRateFound:
		return "No rate covered this lane"
	case OutcomeError:
		if rq.Trace != nil && rq.Trace.Error != "" {
			return "Rating failed: " + rq.Trace.Error
		}
		return "Rating failed"
	default:
		return ""
	}
}

func (rq *RateQuote) winner() *ratetypes.Candidate {
	if rq.Trace == nil {
		return nil
	}

	return rq.Trace.Winner()
}

func (rq *RateQuote) applyDefaults() {
	if rq.Purpose == "" {
		rq.Purpose = PurposeRating
	}
	if rq.Status == "" {
		if rq.ShipmentID == nil || rq.ShipmentID.IsNil() {
			rq.Status = StatusQuoted
		} else {
			rq.Status = StatusApplied
		}
	}
	if rq.BillingCurrency == "" {
		rq.BillingCurrency = rq.Currency
	}
}

func (rq *RateQuote) Validate(multiErr *errortypes.MultiError) {
	rq.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rq,
		validation.Field(&rq.PartyType,
			validation.Required.Error("Party type is required"),
			domainvalidation.ValidEnum[rateagreement.PartyType]("Party type is invalid"),
		),
		validation.Field(&rq.PartyID, validation.Required.Error("Party is required")),
		validation.Field(&rq.Purpose,
			validation.Required.Error("Purpose is required"),
			domainvalidation.ValidEnum[Purpose]("Purpose is invalid"),
		),
		validation.Field(&rq.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[Outcome]("Outcome is invalid"),
		),
		validation.Field(&rq.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&rq.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&rq.BillingCurrency,
			validation.Required.Error("Billing currency is required"),
			validation.Length(3, 3).Error("Billing currency must be a three letter code"),
		),
		validation.Field(&rq.AsOf, validation.Required.Error("As of date is required")),
		validation.Field(&rq.EngineVersion,
			validation.Required.Error("Engine version is required"),
			validation.Length(1, 32).
				Error("Engine version cannot be longer than 32 characters"),
		),
		validation.Field(&rq.ContextHash,
			validation.Required.Error("Context hash is required"),
			validation.Length(1, 64).
				Error("Context hash cannot be longer than 64 characters"),
		),
	))

	// A quote that says it was priced from an agreement without naming one
	// cannot be explained or replayed, which is the whole reason it exists.
	if rq.Outcome == OutcomeRated &&
		(rq.RateAgreementID == nil || rq.RateAgreementID.IsNil()) {
		multiErr.Add(
			"rateAgreementId",
			errortypes.ErrRequired,
			"A rated quote must name the agreement that priced it",
		)
	}

	if rq.Outcome == OutcomeFormulaFallback &&
		(rq.FormulaTemplateID == nil || rq.FormulaTemplateID.IsNil()) {
		multiErr.Add(
			"formulaTemplateId",
			errortypes.ErrRequired,
			"A fallback quote must name the formula template that priced it",
		)
	}
}

func (rq *RateQuote) BeforeCreate(ctx context.Context) error {
	if rq.Purpose == "" {
		rq.Purpose = PurposeRating
	}
	if rq.Status == "" {
		rq.Status = StatusApplied
	}
	rq.applyDefaults()
	now := timeutils.NowUnix()
	if rq.RatedAt == 0 {
		rq.RatedAt = now
	}

	return nil
}

func (rq *RateQuote) GetID() pulid.ID {
	return rq.ID
}

func (rq *RateQuote) GetCreatedAt() int64 {
	return rq.CreatedAt
}

func (rq *RateQuote) GetOrganizationID() pulid.ID {
	return rq.OrganizationID
}

func (rq *RateQuote) GetBusinessUnitID() pulid.ID {
	return rq.BusinessUnitID
}

func (rq *RateQuote) GetTableName() string {
	return "rate_quotes"
}

// GetPostgresSearchConfig declares no search vector: a quote is reached through
// the shipment, the party, the agreement or the rule that produced it, never by
// free text. The listed fields are what a quote list filters on.
func (rq *RateQuote) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rqt",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "purpose", Type: domaintypes.FieldTypeEnum},
			{Name: "outcome", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "party_type", Type: domaintypes.FieldTypeEnum},
		},
	}
}
