package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

// ListConsolidationCandidatesRequest windows the approved queue items a run may
// bill. The lookback reaches before the period start to sweep shipments that were
// delivered inside the period but only approved after it closed; without it those
// straggle into the next period and the customer's statement is wrong twice.
type ListConsolidationCandidatesRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	CustomerIDs  []pulid.ID            `json:"-"`
	PeriodStart  int64                 `json:"-"`
	PeriodEnd    int64                 `json:"-"`
	LookbackDays int16                 `json:"-"`
	Limit        int                   `json:"-"`
}

// ConsolidationCandidate is one approved billing-queue item flattened with every
// key a split rule can group on, so grouping needs one query rather than one per
// shipment.
type ConsolidationCandidate struct {
	BillingQueueItemID pulid.ID            `bun:"billing_queue_item_id"`
	ShipmentID         pulid.ID            `bun:"shipment_id"`
	OrderID            pulid.ID            `bun:"order_id"`
	CustomerID         pulid.ID            `bun:"customer_id"`
	CustomerName       string              `bun:"customer_name"`
	ProNumber          string              `bun:"pro_number"`
	ShipmentBOL        string              `bun:"shipment_bol"`
	OrderNumber        string              `bun:"order_number"`
	OrderPONumber      string              `bun:"order_po_number"`
	ServiceTypeCode    string              `bun:"service_type_code"`
	OriginKey          string              `bun:"origin_key"`
	DestinationKey     string              `bun:"destination_key"`
	ServiceDate        *int64              `bun:"service_date"`
	TotalChargeAmount  decimal.NullDecimal `bun:"total_charge_amount"`
	CurrencyCode       string              `bun:"currency_code"`

	// OrderEligibleLegs and OrderTotalLegs decide whether this run may carry the
	// order's own charges: only when it bills every billable leg of that order.
	// Otherwise a statement catching two of three legs would bill the order's
	// customs brokerage a period early, and stamp it so the completing invoice
	// never sees it.
	OrderEligibleLegs int `bun:"order_eligible_legs"`
	OrderTotalLegs    int `bun:"order_total_legs"`
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
	ListConsolidationCandidates(
		ctx context.Context,
		req *ListConsolidationCandidatesRequest,
	) ([]*ConsolidationCandidate, error)
	GetStatusCounts(
		ctx context.Context,
		req *GetBillingQueueStatsRequest,
	) (map[billingqueue.Status]int, error)
}
