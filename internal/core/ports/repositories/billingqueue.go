package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type BillingQueueFilterOptions struct {
	IncludeShipmentDetails bool   `query:"includeShipmentDetails"`
	Status                 string `query:"status"`
	BillType               string `query:"billType"`
}

type ListBillingQueueRequest struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions BillingQueueFilterOptions `query:"filterOptions"`
}

type GetBillingQueueItemRequest struct {
	BillingQueueItemID pulid.ID
	OrgID              pulid.ID
	BuID               pulid.ID
	UserID             pulid.ID
	FilterOptions      BillingQueueFilterOptions `query:"filterOptions"`
}

type BulkTransferRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type BillingQueueRepository interface {
	List(ctx context.Context, req *ListBillingQueueRequest) (*ports.ListResult[*billingqueue.QueueItem], error)
	GetByID(ctx context.Context, req GetBillingQueueItemRequest) (*billingqueue.QueueItem, error)
	Create(ctx context.Context, qi *billingqueue.QueueItem) (*billingqueue.QueueItem, error)
	Update(ctx context.Context, qi *billingqueue.QueueItem) (*billingqueue.QueueItem, error)
	BulkTransfer(ctx context.Context, req *BulkTransferRequest) error
}
