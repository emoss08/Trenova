package repositories

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type StampExchangeRateRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Rate       decimal.Decimal
	Date       int64
}
