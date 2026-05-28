package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type RateConversionResult struct {
	FromCurrency       string          `json:"fromCurrency"`
	ToCurrency         string          `json:"toCurrency"`
	Amount             decimal.Decimal `json:"amount"`
	Rate               decimal.Decimal `json:"rate"`
	Converted          decimal.Decimal `json:"converted"`
	Date               string          `json:"date"`
	Provider           string          `json:"provider,omitempty"`
	RateType           string          `json:"rateType,omitempty"`
	SourceTimestamp    *time.Time      `json:"sourceTimestamp,omitempty"`
	FetchedAt          *time.Time      `json:"fetchedAt,omitempty"`
	SettlementEligible bool            `json:"settlementEligible"`
	SettlementQuoteID  string          `json:"settlementQuoteId,omitempty"`
}

type LatestRatesResult struct {
	BaseCurrency string                     `json:"baseCurrency"`
	Date         string                     `json:"date"`
	Provider     string                     `json:"provider"`
	RateType     string                     `json:"rateType"`
	Rates        map[string]decimal.Decimal `json:"rates"`
}

type CreateSettlementQuoteRequest struct {
	FromCurrency string                `json:"fromCurrency"`
	ToCurrency   string                `json:"toCurrency"`
	Amount       decimal.Decimal       `json:"amount"`
	RateType     exchangerate.RateType `json:"rateType,omitempty"`
	Date         string                `json:"date,omitempty"`
}

type ExchangeRateService interface {
	Convert(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		fromCurrency, toCurrency string,
		amount decimal.Decimal,
		date time.Time,
	) (*RateConversionResult, error)
	GetLatestRates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		baseCurrency string,
	) (*LatestRatesResult, error)
	CreateSettlementQuote(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		req *CreateSettlementQuoteRequest,
	) (*exchangerate.SettlementQuote, error)
	RefreshRates(ctx context.Context, tenantInfo pagination.TenantInfo, baseCurrency string) error
}
