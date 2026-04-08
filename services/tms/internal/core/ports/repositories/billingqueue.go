package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListBillingQueueItemsRequest struct {
	Filter        *pagination.QueryOptions `json:"filter"`
	IncludePosted bool                     `json:"includePosted"`
}

type GetBillingQueueItemByIDRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	ItemID                pulid.ID              `json:"itemId"`
	ExpandShipmentDetails bool                  `json:"expandShipmentDetails"`
}

type GetBillingQueueStatsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

type BillingQueueRepository interface {
	List(
		ctx context.Context,
		req *ListBillingQueueItemsRequest,
	) (*pagination.ListResult[*billingqueue.BillingQueueItem], error)
	GetByID(
		ctx context.Context,
		req *GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	Create(
		ctx context.Context,
		entity *billingqueue.BillingQueueItem,
	) (*billingqueue.BillingQueueItem, error)
	Update(
		ctx context.Context,
		entity *billingqueue.BillingQueueItem,
	) (*billingqueue.BillingQueueItem, error)
	ExistsByShipmentAndType(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		billType billingqueue.BillType,
	) (bool, error)
	GetStatusCounts(
		ctx context.Context,
		req *GetBillingQueueStatsRequest,
	) (map[billingqueue.Status]int, error)
}
