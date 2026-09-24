package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/shopspring/decimal"
)

const (
	maxOrderShipments = 50
	maxOrderCharges   = 100
)

var orderStatuses = []string{
	string(order.StatusDraft),
	string(order.StatusConfirmed),
	string(order.StatusInProgress),
	string(order.StatusCompleted),
	string(order.StatusBilled),
	string(order.StatusClosed),
	string(order.StatusCanceled),
}

func orderToolProviders() []any {
	return []any{
		newListOrdersTool,
		provideGetOrderTool,
	}
}

type orderReader interface {
	Get(ctx context.Context, req repositories.GetOrderByIDRequest) (*order.Order, error)
}

func provideGetOrderTool(
	orders *orderservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetOrderTool(orders, permissions)
}

func nullDecimalMoney(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.StringFixed(2)
}

type orderRow struct {
	ID           string       `json:"id"`
	OrderNumber  string       `json:"orderNumber"`
	Status       string       `json:"status"`
	CustomerID   string       `json:"customerId"`
	Customer     string       `json:"customer,omitempty"`
	PONumber     string       `json:"poNumber,omitempty"`
	BOL          string       `json:"bol,omitempty"`
	Currency     string       `json:"currency"`
	QuotedAmount string       `json:"quotedAmount,omitempty"`
	BaseAmount   string       `json:"baseAmount,omitempty"`
	TotalAmount  string       `json:"totalAmount,omitempty"`
	CreatedAt    optionalDate `json:"createdAt"`
}

func orderRowFrom(entity *order.Order, gate *fieldGate) orderRow {
	row := orderRow{
		ID:          entity.ID.String(),
		OrderNumber: entity.OrderNumber,
		Status:      string(entity.Status),
		CustomerID:  entity.CustomerID.String(),
		PONumber:    entity.PONumber,
		BOL:         entity.BOL,
		Currency:    entity.CurrencyCode,
		CreatedAt:   recordedDate(entity.CreatedAt),
	}
	if entity.Customer != nil {
		row.Customer = entity.Customer.Name
	}
	if gate.show("quotedAmount", "quotedAmount") {
		row.QuotedAmount = nullDecimalMoney(entity.QuotedAmount)
	}
	if gate.show("baseAmount", "baseAmount") {
		row.BaseAmount = nullDecimalMoney(entity.BaseAmount)
	}
	if gate.show("totalAmount", "totalAmount") {
		row.TotalAmount = nullDecimalMoney(entity.TotalAmount)
	}

	return row
}

func newListOrdersTool(
	repo repositories.OrderRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_orders",
		entityPlural: "orders",
		summary: "List customer orders, the commercial record that groups one or more " +
			"shipments and their charges under a customer's order number, PO or BOL. " +
			"get_order opens one with its shipments and charges.",
		resource: permission.ResourceOrder,
		config:   querybuilder.GetFieldConfiguration((*order.Order)(nil)),
		fields: []listField{
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: orderStatuses,
				Note:   "Billed means invoiced; Closed means nothing more will be added",
			},
			{Name: "orderNumber", Kind: filterText, Sortable: true},
			{Name: "poNumber", Kind: filterText},
			{Name: "bol", Kind: filterText},
			{Name: "totalAmount", Kind: filterNumber, Sortable: true},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListOrdersRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *order.Order) any {
				return orderRowFrom(item, gate)
			}), nil
		},
	})
}

type getOrderTool struct {
	orders orderReader
	access fieldAccess
}

func newGetOrderTool(
	orders orderReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getOrderTool{orders: orders, access: newFieldAccess(permissions)}
}

func (t *getOrderTool) Name() string { return "get_order" }

func (t *getOrderTool) Description() string {
	return "Retrieve one customer order by id with the shipments it groups and its " +
		"order-level charges, including which charges are already invoiced. Use " +
		"list_orders first when you have an order number, PO or BOL rather than an id."
}

func (t *getOrderTool) ParamSchema() map[string]any {
	return idSchema("orderId", "The order's id, from list_orders, the orderId on "+
		"get_shipment or get_invoice, or the page you are on.")
}

func (t *getOrderTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceOrder})
}

type orderShipmentRow struct {
	ID        string `json:"id"`
	ProNumber string `json:"proNumber"`
	BOL       string `json:"bol,omitempty"`
	Status    string `json:"status"`
}

type orderChargeRow struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Amount      string       `json:"amount,omitempty"`
	Invoiced    bool         `json:"invoiced"`
	InvoiceID   string       `json:"invoiceId,omitempty"`
	InvoicedAt  optionalDate `json:"invoicedAt"`
	SplitBill   bool         `json:"splitBill"`
}

type orderView struct {
	orderRow

	ShipmentCount    int                `json:"shipmentCount"`
	Shipments        []orderShipmentRow `json:"shipments"`
	ShipmentsOmitted int                `json:"shipmentsOmitted,omitempty"`
	ChargeCount      int                `json:"chargeCount"`
	Charges          []orderChargeRow   `json:"charges"`
	ChargesOmitted   int                `json:"chargesOmitted,omitempty"`
	Withheld         []string           `json:"withheldByAccess,omitempty"`
}

func (t *getOrderTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "orderId")
	if err != nil {
		return nil, err
	}

	entity, err := t.orders.Get(ctx, repositories.GetOrderByIDRequest{
		ID:              id,
		TenantInfo:      tenantOf(params),
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceOrder)
	view := orderView{
		orderRow:         orderRowFrom(entity, gate),
		ShipmentCount:    len(entity.Shipments),
		ShipmentsOmitted: max(len(entity.Shipments)-maxOrderShipments, 0),
		ChargeCount:      len(entity.Charges),
		ChargesOmitted:   max(len(entity.Charges)-maxOrderCharges, 0),
		Shipments:        make([]orderShipmentRow, 0, min(len(entity.Shipments), maxOrderShipments)),
		Charges:          make([]orderChargeRow, 0, min(len(entity.Charges), maxOrderCharges)),
	}
	for _, item := range entity.Shipments {
		if item == nil || len(view.Shipments) == maxOrderShipments {
			continue
		}
		view.Shipments = append(view.Shipments, orderShipmentRow{
			ID:        item.ID.String(),
			ProNumber: item.ProNumber,
			BOL:       item.BOL,
			Status:    string(item.Status),
		})
	}

	showAmount := gate.show("amount", "charges.amount")
	for _, charge := range entity.Charges {
		if charge == nil || len(view.Charges) == maxOrderCharges {
			continue
		}
		row := orderChargeRow{
			ID:          charge.ID.String(),
			Description: charge.Description,
			Invoiced:    charge.IsFullyInvoiced(),
			InvoiceID:   pulidString(charge.InvoiceID),
			InvoicedAt:  expectedDate(charge.InvoicedAt, "not invoiced"),
			SplitBill:   charge.HasAllocations(),
		}
		if showAmount {
			row.Amount = charge.Amount.StringFixed(2)
		}
		view.Charges = append(view.Charges, row)
	}
	view.Withheld = gate.Withheld()

	return view, nil
}
