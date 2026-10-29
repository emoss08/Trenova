package operationsquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
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

func NullDecimalMoney(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.StringFixed(2)
}

type orderRow struct {
	ID           string                 `json:"id"`
	OrderNumber  string                 `json:"orderNumber"`
	Status       string                 `json:"status"`
	CustomerID   string                 `json:"customerId"`
	Customer     string                 `json:"customer,omitempty"`
	PONumber     string                 `json:"poNumber,omitempty"`
	BOL          string                 `json:"bol,omitempty"`
	Currency     string                 `json:"currency"`
	QuotedAmount string                 `json:"quotedAmount,omitempty"`
	BaseAmount   string                 `json:"baseAmount,omitempty"`
	TotalAmount  string                 `json:"totalAmount,omitempty"`
	CreatedAt    querytool.OptionalDate `json:"createdAt"`
}

func orderRowFrom(entity *order.Order, gate *fieldgate.Gate) orderRow {
	row := orderRow{
		ID:          entity.ID.String(),
		OrderNumber: entity.OrderNumber,
		Status:      string(entity.Status),
		CustomerID:  entity.CustomerID.String(),
		PONumber:    entity.PONumber,
		BOL:         entity.BOL,
		Currency:    entity.CurrencyCode,
		CreatedAt:   querytool.RecordedDate(entity.CreatedAt),
	}
	if entity.Customer != nil {
		row.Customer = entity.Customer.Name
	}
	if gate.Show("quotedAmount", "quotedAmount") {
		row.QuotedAmount = NullDecimalMoney(entity.QuotedAmount)
	}
	if gate.Show("baseAmount", "baseAmount") {
		row.BaseAmount = NullDecimalMoney(entity.BaseAmount)
	}
	if gate.Show(querytool.FieldTotalAmount, querytool.FieldTotalAmount) {
		row.TotalAmount = NullDecimalMoney(entity.TotalAmount)
	}

	return row
}

func NewListOrdersTool(
	repo orderports.OrderRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return listtool.New(listtool.Spec{
		Name:         "order_list",
		Row:          orderRow{},
		EntityPlural: "orders",
		SearchTerms:  []string{"po number", "order number"},
		Summary: "List customer orders, the commercial record that groups one or more " +
			"shipments and their charges under a customer's order number, PO or BOL. " +
			"order_get opens one with its shipments and charges.",
		Resource: permission.ResourceOrder,
		Config:   listquery.Describe((*order.Order)(nil), orderdb.OrdersSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ParamStatus,
				Kind:   filtercatalog.KindEnum,
				Values: orderStatuses,
				Note:   "Billed means invoiced; Closed means nothing more will be added",
			},
			{Name: "orderNumber", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "poNumber", Kind: filtercatalog.KindText},
			{Name: "bol", Kind: filtercatalog.KindText},
			{Name: querytool.FieldTotalAmount, Kind: filtercatalog.KindNumber, Sortable: true},
			{Name: "createdAt", Kind: filtercatalog.KindDate, Sortable: true},
		},
		Access: fieldgate.NewAccess(permissions),
		FetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			_ pagination.CursorInfo,
			gate *fieldgate.Gate,
		) (listtool.Page, error) {
			result, err := repo.List(ctx, &orderports.ListOrdersRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *order.Order) any {
				return orderRowFrom(item, gate)
			}), nil
		},
	})
}
