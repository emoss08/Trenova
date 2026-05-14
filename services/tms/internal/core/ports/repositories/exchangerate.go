package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetExchangeRateRequest struct {
	TenantInfo   pagination.TenantInfo
	FromCurrency string
	ToCurrency   string
	Date         time.Time
}

type UpsertExchangeRatesRequest struct {
	TenantInfo pagination.TenantInfo
	Rates      []*exchangerate.ExchangeRate
}

type ExchangeRateRepository interface {
	GetRate(ctx context.Context, req *GetExchangeRateRequest) (*exchangerate.ExchangeRate, error)
	UpsertRates(ctx context.Context, req *UpsertExchangeRatesRequest) error
	GetLatestDate(ctx context.Context, tenantInfo pagination.TenantInfo) (*time.Time, error)
}
