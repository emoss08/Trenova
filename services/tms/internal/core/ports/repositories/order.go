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
}

type UpdateOrderStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OrderID    pulid.ID              `json:"-"`
	Status     order.Status          `json:"-"`
	Version    int64                 `json:"-"`
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
	AttachShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentIDs []pulid.ID,
	) (int64, error)
	DetachShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentID pulid.ID,
	) (int64, error)
	// CountShipmentsWithDifferentCustomer returns how many of the given shipments belong
	// to a customer other than customerID (invariant #4 — one customer per order).
	CountShipmentsWithDifferentCustomer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		shipmentIDs []pulid.ID,
	) (int64, error)
	AddCharge(
		ctx context.Context,
		entity *order.OrderCharge,
	) (*order.OrderCharge, error)
	RemoveCharge(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		chargeID pulid.ID,
	) (int64, error)
	ListCharges(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) ([]*order.OrderCharge, error)
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
