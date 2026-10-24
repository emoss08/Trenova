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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateQuote)(nil)
	_ validationframework.TenantedEntity = (*RateQuote)(nil)
)

// How each shipment was priced, what it cost, and what an override gave away
type RateQuote struct {
	bun.BaseModel             `bun:"table:rate_quotes,alias:rqt" json:"-"`
	keel.Table                `                                           keel:"rate_quotes,report=Billing"`
	pagination.CursorValueSet `bun:",embed"                      json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"               keel:"id,pk,autoid:rqt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"  keel:"organization_id,pk,tenant"`

	ShipmentID     *pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),nullzero"      keel:"shipment_id"`
	ShipmentMoveID *pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),nullzero" keel:"shipment_move_id"`

	PartyType rateagreement.PartyType `json:"partyType" bun:"party_type,type:rate_agreement_party_type_enum,notnull" keel:"party_type"`
	PartyID   pulid.ID                `json:"partyId"   bun:"party_id,type:VARCHAR(100),notnull"                     keel:"party_id"`

	Purpose Purpose `json:"purpose" bun:"purpose,type:rate_quote_purpose_enum,notnull,default:'Rating'" keel:"purpose"`
	Outcome Outcome `json:"outcome" bun:"outcome,type:rate_quote_outcome_enum,notnull"                  keel:"outcome"`
	Status  Status  `json:"status"  bun:"status,type:rate_quote_status_enum,notnull,default:'Applied'"  keel:"status"`

	RateAgreementID              *pulid.ID `json:"rateAgreementId"              bun:"rate_agreement_id,type:VARCHAR(100),nullzero"         keel:"rate_agreement_id"`
	RateAgreementRuleID          *pulid.ID `json:"rateAgreementRuleId"          bun:"rate_agreement_rule_id,type:VARCHAR(100),nullzero"    keel:"rate_agreement_rule_id"`
	AgreementVersionNumber       *int64    `json:"agreementVersionNumber"       bun:"agreement_version_number,type:BIGINT,nullzero"        keel:"agreement_version_number,label=Agreement Version,dimension"`
	FormulaTemplateID            *pulid.ID `json:"formulaTemplateId"            bun:"formula_template_id,type:VARCHAR(100),nullzero"       keel:"formula_template_id"`
	FormulaTemplateVersionNumber *int64    `json:"formulaTemplateVersionNumber" bun:"formula_template_version_number,type:BIGINT,nullzero" keel:"formula_template_version_number,dimension"`
	SpecificityScore             int32     `json:"specificityScore"             bun:"specificity_score,type:INTEGER,notnull,default:0"     keel:"specificity_score,dimension"`

	Currency          string              `json:"currency"          bun:"currency,type:VARCHAR(3),notnull"                        keel:"currency"`
	LinehaulAmount    decimal.Decimal     `json:"linehaulAmount"    bun:"linehaul_amount,type:NUMERIC(19,4),notnull,default:0"    keel:"linehaul_amount,label=Linehaul,format=money"`
	FuelAmount        decimal.Decimal     `json:"fuelAmount"        bun:"fuel_amount,type:NUMERIC(19,4),notnull,default:0"        keel:"fuel_amount,label=Fuel,format=money"`
	AccessorialAmount decimal.Decimal     `json:"accessorialAmount" bun:"accessorial_amount,type:NUMERIC(19,4),notnull,default:0" keel:"accessorial_amount,label=Accessorials,format=money"`
	TotalAmount       decimal.Decimal     `json:"totalAmount"       bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"       keel:"total_amount,label=Total,format=money"`
	CostAmount        decimal.NullDecimal `json:"costAmount"        bun:"cost_amount,type:NUMERIC(19,4),nullzero"                 keel:"cost_amount,label=Cost,format=money"`
	MarginAmount      decimal.NullDecimal `json:"marginAmount"      bun:"margin_amount,type:NUMERIC(19,4),nullzero"               keel:"margin_amount,label=Margin,format=money"`
	MarginPercent     decimal.NullDecimal `json:"marginPercent"     bun:"margin_percent,type:NUMERIC(9,4),nullzero"               keel:"margin_percent,label=Margin (%)"`

	BillingCurrency string              `json:"billingCurrency" bun:"billing_currency,type:VARCHAR(3),notnull"            keel:"billing_currency"`
	BillingAmount   decimal.Decimal     `json:"billingAmount"   bun:"billing_amount,type:NUMERIC(19,4),notnull,default:0" keel:"billing_amount,format=money"`
	FxRate          decimal.NullDecimal `json:"fxRate"          bun:"fx_rate,type:NUMERIC(24,12),nullzero"                keel:"fx_rate"`
	ExchangeRateID  *pulid.ID           `json:"exchangeRateId"  bun:"exchange_rate_id,type:VARCHAR(100),nullzero"         keel:"exchange_rate_id"`

	AsOf          int64     `json:"asOf"          bun:"as_of,type:BIGINT,notnull"               keel:"as_of,type=epoch"`
	RatedAt       int64     `json:"ratedAt"       bun:"rated_at,type:BIGINT,notnull"            keel:"rated_at"`
	RatedByID     *pulid.ID `json:"ratedById"     bun:"rated_by_id,type:VARCHAR(100),nullzero"  keel:"rated_by_id"`
	EngineVersion string    `json:"engineVersion" bun:"engine_version,type:VARCHAR(32),notnull" keel:"engine_version"`

	ContextHash string `json:"contextHash" bun:"context_hash,type:VARCHAR(64),notnull" keel:"context_hash,noreport"`

	OverrideReason string              `json:"overrideReason" bun:"override_reason,type:TEXT,nullzero"          keel:"override_reason,nullzero"`
	ForegoneAmount decimal.NullDecimal `json:"foregoneAmount" bun:"foregone_amount,type:NUMERIC(19,4),nullzero" keel:"foregone_amount,format=money"`

	Trace *ratetypes.Trace `json:"trace" bun:"trace,type:JSONB,nullzero" keel:"trace,jsonb,noreport"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization         `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" bun:"rel:belongs-to,join:rate_agreement_id=id" keel:"belongs:rate_agreement_id,label=Rate Agreement"`
	RatedBy      *tenant.User                 `json:"ratedBy,omitempty"   bun:"rel:belongs-to,join:rated_by_id=id"       keel:"belongs:rated_by_id,noreport"`
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

func (rq *RateQuote) BeforeAppendModel(_ context.Context, query bun.Query) error {
	rq.applyDefaults()

	if _, ok := query.(*bun.InsertQuery); ok {
		if rq.ID.IsNil() {
			rq.ID = pulid.MustNew("rqt_")
		}
		now := timeutils.NowUnix()
		rq.CreatedAt = now
		if rq.RatedAt == 0 {
			rq.RatedAt = now
		}
	}

	return nil
}

func (rq *RateQuote) BeforeCreate(ctx context.Context) error {
	return rq.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
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
