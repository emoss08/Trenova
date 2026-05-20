package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type RateConversionResult struct {
	FromCurrency string          `json:"fromCurrency"`
	ToCurrency   string          `json:"toCurrency"`
	Amount       decimal.Decimal `json:"amount"`
	Rate         decimal.Decimal `json:"rate"`
	Converted    decimal.Decimal `json:"converted"`
	Date         string          `json:"date"`
}

type LatestRatesResult struct {
	BaseCurrency string             `json:"baseCurrency"`
	Date         string             `json:"date"`
	Rates        map[string]float64 `json:"rates"`
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
	RefreshRates(ctx context.Context, tenantInfo pagination.TenantInfo, baseCurrency string) error
}
