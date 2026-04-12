package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBankReceiptWorkItemByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptWorkItemRepository interface {
	GetByID(
		ctx context.Context,
		req GetBankReceiptWorkItemByIDRequest,
	) (*bankreceiptworkitem.WorkItem, error)
	GetActiveByReceiptID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		bankReceiptID pulid.ID,
	) (*bankreceiptworkitem.WorkItem, error)
	ListActive(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceiptworkitem.WorkItem, error)
	Create(
		ctx context.Context,
		entity *bankreceiptworkitem.WorkItem,
	) (*bankreceiptworkitem.WorkItem, error)
	Update(
		ctx context.Context,
		entity *bankreceiptworkitem.WorkItem,
	) (*bankreceiptworkitem.WorkItem, error)
}
