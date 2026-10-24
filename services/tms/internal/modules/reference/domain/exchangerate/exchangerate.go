package exchangerate

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ExchangeRate)(nil)
	_ bun.BeforeAppendModelHook          = (*SettlementQuote)(nil)
	_ validationframework.TenantedEntity = (*ExchangeRate)(nil)
	_ validationframework.TenantedEntity = (*SettlementQuote)(nil)
)

type Provider string

const ProviderOANDA Provider = "OANDA"

type RateType string

const (
	RateTypeBid RateType = "bid"
	RateTypeAsk RateType = "ask"
	RateTypeMid RateType = "mid"
)

type ExchangeRate struct {
	bun.BaseModel `bun:"table:exchange_rates,alias:er" json:"-"`
	keel.Table    `keel:"exchange_rates"`

	ID                 pulid.ID        `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:fxr_"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID        `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	Provider           Provider        `json:"provider"           bun:"provider,type:VARCHAR(32),notnull" keel:"provider"`
	FromCurrency       string          `json:"fromCurrency"       bun:"from_currency,type:VARCHAR(3),notnull" keel:"from_currency"`
	ToCurrency         string          `json:"toCurrency"         bun:"to_currency,type:VARCHAR(3),notnull" keel:"to_currency"`
	RateType           RateType        `json:"rateType"           bun:"rate_type,type:VARCHAR(16),notnull" keel:"rate_type"`
	Bid                decimal.Decimal `json:"bid"                bun:"bid,type:NUMERIC(24,12),notnull" keel:"bid"`
	Ask                decimal.Decimal `json:"ask"                bun:"ask,type:NUMERIC(24,12),notnull" keel:"ask"`
	Mid                decimal.Decimal `json:"mid"                bun:"mid,type:NUMERIC(24,12),notnull" keel:"mid"`
	SelectedRate       decimal.Decimal `json:"selectedRate"       bun:"selected_rate,type:NUMERIC(24,12),notnull" keel:"selected_rate"`
	Date               string          `json:"date"               bun:"date,type:DATE,notnull" keel:"date"`
	SourceTimestamp    time.Time       `json:"sourceTimestamp"    bun:"source_timestamp,type:TIMESTAMPTZ,notnull" keel:"source_timestamp"`
	FetchedAt          time.Time       `json:"fetchedAt"          bun:"fetched_at,type:TIMESTAMPTZ,notnull,default:current_timestamp" keel:"fetched_at"`
	SettlementEligible bool            `json:"settlementEligible" bun:"settlement_eligible,type:BOOLEAN,notnull" keel:"settlement_eligible"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (e *ExchangeRate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.FromCurrency,
			validation.Required.Error("From currency is required"),
			validation.Length(3, 3).Error("From currency must be 3 characters"),
		),
		validation.Field(&e.ToCurrency,
			validation.Required.Error("To currency is required"),
			validation.Length(3, 3).Error("To currency must be 3 characters"),
		),
		validation.Field(&e.Provider,
			validation.Required.Error("Provider is required"),
		),
		validation.Field(&e.RateType,
			validation.Required.Error("Rate type is required"),
			validation.In(RateTypeBid, RateTypeAsk, RateTypeMid).Error("Rate type is invalid"),
		),
		validation.Field(&e.SelectedRate,
			validation.Required.Error("Rate is required"),
		),
		validation.Field(&e.Date,
			validation.Required.Error("Date is required"),
		),
	))
}

func (e *ExchangeRate) GetID() pulid.ID {
	return e.ID
}

func (e *ExchangeRate) GetTableName() string {
	return "exchange_rates"
}

func (e *ExchangeRate) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *ExchangeRate) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (e *ExchangeRate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().UTC()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("fxr_")
		}
		if e.Provider == "" {
			e.Provider = ProviderOANDA
		}
		if e.RateType == "" {
			e.RateType = RateTypeMid
		}
		if e.SourceTimestamp.IsZero() {
			e.SourceTimestamp = now
		}
		e.FetchedAt = now
	case *bun.UpdateQuery:
		e.FetchedAt = now
	}

	return nil
}

type SettlementQuote struct {
	bun.BaseModel `bun:"table:exchange_rate_settlement_quotes,alias:fxq" json:"-"`
	keel.Table    `keel:"exchange_rate_settlement_quotes"`

	ID              pulid.ID        `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:fxq_"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID        `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	Provider        Provider        `json:"provider"        bun:"provider,type:VARCHAR(32),notnull" keel:"provider"`
	FromCurrency    string          `json:"fromCurrency"    bun:"from_currency,type:VARCHAR(3),notnull" keel:"from_currency"`
	ToCurrency      string          `json:"toCurrency"      bun:"to_currency,type:VARCHAR(3),notnull" keel:"to_currency"`
	Amount          decimal.Decimal `json:"amount"          bun:"amount,type:NUMERIC(24,8),notnull" keel:"amount"`
	Rate            decimal.Decimal `json:"rate"            bun:"rate,type:NUMERIC(24,12),notnull" keel:"rate"`
	ConvertedAmount decimal.Decimal `json:"convertedAmount" bun:"converted_amount,type:NUMERIC(24,8),notnull" keel:"converted_amount"`
	RateType        RateType        `json:"rateType"        bun:"rate_type,type:VARCHAR(16),notnull" keel:"rate_type"`
	SourceTimestamp time.Time       `json:"sourceTimestamp" bun:"source_timestamp,type:TIMESTAMPTZ,notnull" keel:"source_timestamp"`
	FetchedAt       time.Time       `json:"fetchedAt"       bun:"fetched_at,type:TIMESTAMPTZ,notnull,default:current_timestamp" keel:"fetched_at"`
	ExpiresAt       time.Time       `json:"expiresAt"       bun:"expires_at,type:TIMESTAMPTZ,notnull" keel:"expires_at"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (q *SettlementQuote) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		q,
		validation.Field(&q.FromCurrency,
			validation.Required.Error("From currency is required"),
			validation.Length(3, 3).Error("From currency must be 3 characters"),
		),
		validation.Field(&q.ToCurrency,
			validation.Required.Error("To currency is required"),
			validation.Length(3, 3).Error("To currency must be 3 characters"),
		),
		validation.Field(&q.Provider, validation.Required.Error("Provider is required")),
		validation.Field(&q.RateType,
			validation.Required.Error("Rate type is required"),
			validation.In(RateTypeBid, RateTypeAsk, RateTypeMid).Error("Rate type is invalid"),
		),
		validation.Field(&q.Amount, validation.Required.Error("Amount is required")),
		validation.Field(&q.Rate, validation.Required.Error("Rate is required")),
		validation.Field(
			&q.ConvertedAmount,
			validation.Required.Error("Converted amount is required"),
		),
	))
}

func (q *SettlementQuote) GetID() pulid.ID {
	return q.ID
}

func (q *SettlementQuote) GetTableName() string {
	return "exchange_rate_settlement_quotes"
}

func (q *SettlementQuote) GetOrganizationID() pulid.ID {
	return q.OrganizationID
}

func (q *SettlementQuote) GetBusinessUnitID() pulid.ID {
	return q.BusinessUnitID
}

func (q *SettlementQuote) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().UTC()

	if _, ok := query.(*bun.InsertQuery); !ok {
		return nil
	}

	if q.ID.IsNil() {
		q.ID = pulid.MustNew("fxq_")
	}
	if q.Provider == "" {
		q.Provider = ProviderOANDA
	}
	if q.RateType == "" {
		q.RateType = RateTypeMid
	}
	if q.SourceTimestamp.IsZero() {
		q.SourceTimestamp = now
	}
	if q.FetchedAt.IsZero() {
		q.FetchedAt = now
	}
	if q.ExpiresAt.IsZero() {
		q.ExpiresAt = q.FetchedAt.Add(15 * time.Minute)
	}

	return nil
}
