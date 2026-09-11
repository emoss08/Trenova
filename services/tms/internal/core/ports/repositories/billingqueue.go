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

// MarkPostedForInvoiceRequest sweeps the billing-queue items an invoice actually
// billed. InvoiceID is the real predicate; OrderID and ShipmentIDs are a fallback for
// rows written before the invoice back-link existed and which the migration could not
// link with confidence. Canceled items are never swept.
type MarkPostedForInvoiceRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	InvoiceID   pulid.ID              `json:"-"`
	OrderID     pulid.ID              `json:"-"`
	ShipmentIDs []pulid.ID            `json:"-"`
}

// AttachInvoiceRequest links every billing-queue item an invoice bills back to it, so
// the posting sweep and the double-bill guard can both work from one exact predicate.
type AttachInvoiceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	ItemIDs    []pulid.ID            `json:"-"`
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
	MarkPostedForInvoice(
		ctx context.Context,
		req *MarkPostedForInvoiceRequest,
	) (int64, error)
	AttachInvoice(
		ctx context.Context,
		req *AttachInvoiceRequest,
	) (int64, error)
	GetStatusCounts(
		ctx context.Context,
		req *GetBillingQueueStatsRequest,
	) (map[billingqueue.Status]int, error)
}
