package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCustomerLedgerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
}

type ListARAgingRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type AccountsReceivableRepository interface {
	ListCustomerLedger(
		ctx context.Context,
		req ListCustomerLedgerRequest,
	) ([]*accountsreceivable.LedgerEntry, error)
	ListARAging(
		ctx context.Context,
		req ListARAgingRequest,
	) ([]*accountsreceivable.CustomerAgingRow, error)
}
