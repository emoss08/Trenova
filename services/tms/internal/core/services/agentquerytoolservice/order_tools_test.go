package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOrders struct {
	entity *order.Order
}

func (f *fakeOrders) Get(context.Context, repositories.GetOrderByIDRequest) (*order.Order, error) {
	return f.entity, nil
}

func TestGetOrder_BoundsShipmentsAndCharges(t *testing.T) {
	t.Parallel()

	shipments := make([]*shipment.Shipment, 0, maxOrderShipments+3)
	for range maxOrderShipments + 3 {
		shipments = append(shipments, &shipment.Shipment{ID: pulid.MustNew("shp_")})
	}
	entity := &order.Order{
		ID:          pulid.MustNew("ord_"),
		OrderNumber: "PO-88213",
		TotalAmount: decimal.NewNullDecimal(decimal.RequireFromString("900")),
		Shipments:   shipments,
		Charges: []*order.OrderCharge{
			{ID: pulid.MustNew("ordchg_"), Description: "Lumper", InvoicedAt: 1_700_000_000},
		},
	}
	tool := newGetOrderTool(&fakeOrders{entity: entity}, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"orderId": entity.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(orderView)
	assert.Len(t, view.Shipments, maxOrderShipments)
	assert.Equal(t, 3, view.ShipmentsOmitted)
	require.Len(t, view.Charges, 1)
	assert.True(t, view.Charges[0].Invoiced)
	assert.Equal(t, "900.00", view.TotalAmount, "an order's amounts are Internal in the registry")
}
