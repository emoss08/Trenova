package exchangerate

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
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
	keel.Table `keel:"exchange_rates"`

	ID                 pulid.ID        `json:"id"                 keel:"id,pk,autoid:fxr_"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID        `json:"organizationId"     keel:"organization_id,pk,tenant"`
	Provider           Provider        `json:"provider"           keel:"provider"`
	FromCurrency       string          `json:"fromCurrency"       keel:"from_currency"`
	ToCurrency         string          `json:"toCurrency"         keel:"to_currency"`
	RateType           RateType        `json:"rateType"           keel:"rate_type"`
	Bid                decimal.Decimal `json:"bid"                keel:"bid"`
	Ask                decimal.Decimal `json:"ask"                keel:"ask"`
	Mid                decimal.Decimal `json:"mid"                keel:"mid"`
	SelectedRate       decimal.Decimal `json:"selectedRate"       keel:"selected_rate"`
	Date               string          `json:"date"               keel:"date"`
	SourceTimestamp    time.Time       `json:"sourceTimestamp"    keel:"source_timestamp"`
	FetchedAt          time.Time       `json:"fetchedAt"          keel:"fetched_at"`
	SettlementEligible bool            `json:"settlementEligible" keel:"settlement_eligible"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (e *ExchangeRate) BeforeUpdate(_ context.Context) error {
	now := time.Now().UTC()
	e.FetchedAt = now

	return nil
}

func (e *ExchangeRate) BeforeCreate(_ context.Context) error {
	now := time.Now().UTC()
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
	if e.FetchedAt.IsZero() {
		e.FetchedAt = timeutils.Now()
	}

	return nil
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

type SettlementQuote struct {
	keel.Table `keel:"exchange_rate_settlement_quotes"`

	ID              pulid.ID        `json:"id"              keel:"id,pk,autoid:fxq_"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID        `json:"organizationId"  keel:"organization_id,pk,tenant"`
	Provider        Provider        `json:"provider"        keel:"provider"`
	FromCurrency    string          `json:"fromCurrency"    keel:"from_currency"`
	ToCurrency      string          `json:"toCurrency"      keel:"to_currency"`
	Amount          decimal.Decimal `json:"amount"          keel:"amount"`
	Rate            decimal.Decimal `json:"rate"            keel:"rate"`
	ConvertedAmount decimal.Decimal `json:"convertedAmount" keel:"converted_amount"`
	RateType        RateType        `json:"rateType"        keel:"rate_type"`
	SourceTimestamp time.Time       `json:"sourceTimestamp" keel:"source_timestamp"`
	FetchedAt       time.Time       `json:"fetchedAt"       keel:"fetched_at"`
	ExpiresAt       time.Time       `json:"expiresAt"       keel:"expires_at"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (q *SettlementQuote) BeforeCreate(_ context.Context) error {
	now := time.Now().UTC()
	if q.Provider == "" {
		q.Provider = ProviderOANDA
	}
	if q.RateType == "" {
		q.RateType = RateTypeMid
	}
	if q.SourceTimestamp.IsZero() {
		q.SourceTimestamp = now
	}
	if q.ExpiresAt.IsZero() {
		q.ExpiresAt = q.FetchedAt.Add(15 * time.Minute)
	}
	if q.FetchedAt.IsZero() {
		q.FetchedAt = timeutils.Now()
	}

	return nil
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
