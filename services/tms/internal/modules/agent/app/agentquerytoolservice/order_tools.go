package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/shopspring/decimal"
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
	if gate.show(fieldTotalAmount, fieldTotalAmount) {
		row.TotalAmount = nullDecimalMoney(entity.TotalAmount)
	}

	return row
}

func newListOrdersTool(
	repo orderports.OrderRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "order_list",
		entityPlural: "orders",
		searchTerms:  []string{"po number", "order number"},
		summary: "List customer orders, the commercial record that groups one or more " +
			"shipments and their charges under a customer's order number, PO or BOL. " +
			"order_get opens one with its shipments and charges.",
		resource: permission.ResourceOrder,
		config:   querybuilder.GetFieldConfiguration((*order.Order)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: orderStatuses,
				Note:   "Billed means invoiced; Closed means nothing more will be added",
			},
			{Name: "orderNumber", Kind: filterText, Sortable: true},
			{Name: "poNumber", Kind: filterText},
			{Name: "bol", Kind: filterText},
			{Name: fieldTotalAmount, Kind: filterNumber, Sortable: true},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			_ pagination.CursorInfo,
			gate *fieldGate,
		) (listPage, error) {
			result, err := repo.List(ctx, &orderports.ListOrdersRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			return offsetListPage(result.Items, func(item *order.Order) any {
				return orderRowFrom(item, gate)
			}), nil
		},
	})
}
