package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetExchangeRateRequest struct {
	TenantInfo   pagination.TenantInfo
	Provider     exchangerate.Provider
	FromCurrency string
	ToCurrency   string
	RateType     exchangerate.RateType
	Date         time.Time
}

type UpsertExchangeRatesRequest struct {
	TenantInfo pagination.TenantInfo
	Rates      []*exchangerate.ExchangeRate
}

type CreateSettlementQuoteRequest struct {
	TenantInfo pagination.TenantInfo
	Quote      *exchangerate.SettlementQuote
}

type ExchangeRateRepository interface {
	GetRate(ctx context.Context, req *GetExchangeRateRequest) (*exchangerate.ExchangeRate, error)
	UpsertRates(ctx context.Context, req *UpsertExchangeRatesRequest) error
	CreateSettlementQuote(
		ctx context.Context,
		req *CreateSettlementQuoteRequest,
	) (*exchangerate.SettlementQuote, error)
	GetLatestDate(ctx context.Context, tenantInfo pagination.TenantInfo) (*time.Time, error)
}
