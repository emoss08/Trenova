package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type ListOrdersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListOrdersConnectionRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	Cursor       pagination.CursorInfo    `json:"-"`
	OrderColumns []string                 `json:"-"`
}

type GetOrderByIDRequest struct {
	ID              pulid.ID              `json:"id"              form:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"      form:"tenantInfo"`
	IncludeShipment bool                  `json:"includeShipment" form:"includeShipment"`
}

type GetOrdersByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OrderIDs   []pulid.ID            `json:"orderIds"`
}

type OrderSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	// AttachableOnly restricts options to orders that still accept new legs
	// (Billed/Closed/Canceled orders are excluded).
	AttachableOnly bool `json:"attachableOnly"`
	// CustomerID scopes options to a single customer (invariant #4 for the shipment
	// billing form's order picker).
	CustomerID pulid.ID `json:"customerId"`
}

// ShipmentAttachRef is the minimal shipment projection the order service needs to
// validate an attach: current parent order, customer, and leg status.
type ShipmentAttachRef struct {
	ID         pulid.ID        `bun:"id"`
	OrderID    pulid.ID        `bun:"order_id"`
	CustomerID pulid.ID        `bun:"customer_id"`
	Status     shipment.Status `bun:"status"`
}

type MarkOrderChargesInvoicedRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OrderID    pulid.ID              `json:"-"`
	ChargeIDs  []pulid.ID            `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	InvoicedAt int64                 `json:"-"`
}

type UpdateOrderStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OrderID    pulid.ID              `json:"-"`
	Status     order.Status          `json:"-"`
	Version    int64                 `json:"-"`
}

type RemoveOrderChargeRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OrderID    pulid.ID              `json:"-"`
	ChargeID   pulid.ID              `json:"-"`
}

type OrderRepository interface {
	List(
		ctx context.Context,
		req *ListOrdersRequest,
	) (*pagination.ListResult[*order.Order], error)
	ListConnection(
		ctx context.Context,
		req *ListOrdersConnectionRequest,
	) (*pagination.CursorListResult[*order.Order], error)
	GetByID(
		ctx context.Context,
		req GetOrderByIDRequest,
	) (*order.Order, error)
	GetByIDs(
		ctx context.Context,
		req GetOrdersByIDsRequest,
	) ([]*order.Order, error)
	Create(
		ctx context.Context,
		entity *order.Order,
	) (*order.Order, error)
	CreateInTx(
		ctx context.Context,
		tx bun.IDB,
		entity *order.Order,
	) error
	Update(
		ctx context.Context,
		entity *order.Order,
	) (*order.Order, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateOrderStatusRequest,
	) (*order.Order, error)
	GetShipmentStatuses(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) ([]shipment.Status, error)
	// AttachShipments points the given legs at the order. Canceled and Invoiced legs
	// are never attached (the WHERE clause excludes them); the caller compares the
	// affected count against the requested count.
	AttachShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentIDs []pulid.ID,
	) (int64, error)
	// DetachShipment moves a leg from the order onto newOrderID (its replacement
	// single-leg order) so the "every shipment has a commercial parent" invariant
	// holds. Invoiced legs are never detached.
	DetachShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentID pulid.ID,
		newOrderID pulid.ID,
	) (int64, error)
	// GetShipmentAttachRefs loads the attach-validation projection for the given
	// shipments (invariant #4 — one customer per order — plus leg-status and
	// source-order guards).
	GetShipmentAttachRefs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) ([]ShipmentAttachRef, error)
	// DeleteIfEmpty removes an order that has no legs, no charges, and no billing
	// artifacts — the leftover auto-order after its only leg was attached elsewhere.
	DeleteIfEmpty(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) (int64, error)
	AddCharge(
		ctx context.Context,
		entity *order.OrderCharge,
	) (*order.OrderCharge, error)
	RemoveCharge(
		ctx context.Context,
		req *RemoveOrderChargeRequest,
	) (int64, error)
	// UpdateCharge rewrites an uninvoiced charge's description/amount with an
	// optimistic version check; charges already carried on an invoice are immutable.
	UpdateCharge(
		ctx context.Context,
		entity *order.OrderCharge,
	) (int64, error)
	ListCharges(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) ([]*order.OrderCharge, error)
	// ListUninvoicedCharges returns the order's charges that have not yet been carried
	// on an invoice (invoice_id IS NULL). Order charges are billed exactly once.
	ListUninvoicedCharges(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) ([]*order.OrderCharge, error)
	MarkChargesInvoiced(
		ctx context.Context,
		req *MarkOrderChargesInvoicedRequest,
	) (int64, error)
	// RecalculateTotal recomputes an order's total_amount as the sum of its leg charges
	// plus its order-level charges (invariant #1 — money rolls up).
	RecalculateTotal(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) error
	SelectOptions(
		ctx context.Context,
		req *OrderSelectOptionsRequest,
	) (*pagination.ListResult[*order.Order], error)
}
