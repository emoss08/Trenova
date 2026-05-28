package exchangerate

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

	ID                 pulid.ID        `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID        `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Provider           Provider        `json:"provider"           bun:"provider,type:VARCHAR(32),notnull"`
	FromCurrency       string          `json:"fromCurrency"       bun:"from_currency,type:VARCHAR(3),notnull"`
	ToCurrency         string          `json:"toCurrency"         bun:"to_currency,type:VARCHAR(3),notnull"`
	RateType           RateType        `json:"rateType"           bun:"rate_type,type:VARCHAR(16),notnull"`
	Bid                decimal.Decimal `json:"bid"                bun:"bid,type:NUMERIC(24,12),notnull"`
	Ask                decimal.Decimal `json:"ask"                bun:"ask,type:NUMERIC(24,12),notnull"`
	Mid                decimal.Decimal `json:"mid"                bun:"mid,type:NUMERIC(24,12),notnull"`
	SelectedRate       decimal.Decimal `json:"selectedRate"       bun:"selected_rate,type:NUMERIC(24,12),notnull"`
	Date               string          `json:"date"               bun:"date,type:DATE,notnull"`
	SourceTimestamp    time.Time       `json:"sourceTimestamp"    bun:"source_timestamp,type:TIMESTAMPTZ,notnull"`
	FetchedAt          time.Time       `json:"fetchedAt"          bun:"fetched_at,type:TIMESTAMPTZ,notnull,default:current_timestamp"`
	SettlementEligible bool            `json:"settlementEligible" bun:"settlement_eligible,type:BOOLEAN,notnull,default:false"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (e *ExchangeRate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(e,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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

	ID              pulid.ID        `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID        `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Provider        Provider        `json:"provider"        bun:"provider,type:VARCHAR(32),notnull"`
	FromCurrency    string          `json:"fromCurrency"    bun:"from_currency,type:VARCHAR(3),notnull"`
	ToCurrency      string          `json:"toCurrency"      bun:"to_currency,type:VARCHAR(3),notnull"`
	Amount          decimal.Decimal `json:"amount"          bun:"amount,type:NUMERIC(24,8),notnull"`
	Rate            decimal.Decimal `json:"rate"            bun:"rate,type:NUMERIC(24,12),notnull"`
	ConvertedAmount decimal.Decimal `json:"convertedAmount" bun:"converted_amount,type:NUMERIC(24,8),notnull"`
	RateType        RateType        `json:"rateType"        bun:"rate_type,type:VARCHAR(16),notnull"`
	SourceTimestamp time.Time       `json:"sourceTimestamp" bun:"source_timestamp,type:TIMESTAMPTZ,notnull"`
	FetchedAt       time.Time       `json:"fetchedAt"       bun:"fetched_at,type:TIMESTAMPTZ,notnull,default:current_timestamp"`
	ExpiresAt       time.Time       `json:"expiresAt"       bun:"expires_at,type:TIMESTAMPTZ,notnull"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (q *SettlementQuote) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(q,
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
		validation.Field(&q.ConvertedAmount, validation.Required.Error("Converted amount is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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
