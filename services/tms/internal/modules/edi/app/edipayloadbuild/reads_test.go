package edipayloadbuild

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/orderportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFreightInvoicePayload_QuotesEachShipmentsPurchaseOrderFromOneRead(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	first := pulid.MustNew("shp_")
	second := pulid.MustNew("shp_")
	third := pulid.MustNew("shp_")
	delivered := int64(1_768_219_200)

	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().
		ListSummariesByIDs(mock.Anything, &shipmentports.ListShipmentSummariesRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: []pulid.ID{first, second, third},
		}).
		Return([]*shipmentports.ShipmentSummary{
			{ShipmentID: second, PONumber: "PO-2"},
			{ShipmentID: first, PONumber: " PO-1 ", ServiceDate: &delivered},
			{ShipmentID: third},
		}, nil).
		Once()

	svc := New(Params{ShipmentRepo: shipments})
	payload, err := svc.FreightInvoicePayload(t.Context(), tenantInfo, &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Number:        "INV-4001",
		ShipmentCount: 3,
		Lines: []*invoice.InvoiceLine{
			{LineNumber: 1, ShipmentID: first, Amount: decimal.NewFromInt(100)},
			{LineNumber: 2, ShipmentID: first, Amount: decimal.NewFromInt(25)},
			{LineNumber: 3, ShipmentID: second, Amount: decimal.NewFromInt(200)},
			{LineNumber: 4, ShipmentID: third, Amount: decimal.NewFromInt(300)},
			{LineNumber: 5, Amount: decimal.NewFromInt(10)},
		},
	})
	require.NoError(t, err)

	invoiced := payload.FreightInvoice.Shipments
	require.Len(t, invoiced, 3)
	require.Equal(t, first, invoiced[0].ShipmentID)
	require.Equal(t, "PO-1", invoiced[0].PONumber)
	require.Equal(t, delivered, invoiced[0].DeliveryDate)
	require.Equal(t, "PO-2", invoiced[1].PONumber)
	require.Zero(t, invoiced[1].DeliveryDate)
	require.Empty(t, invoiced[2].PONumber, "a shipment without an order quotes no PO")
}

func TestFreightInvoicePayload_ASingleShipmentInvoiceReadsNothing(t *testing.T) {
	t.Parallel()

	svc := New(Params{ShipmentRepo: shipmentportsmock.NewMockShipmentRepository(t)})
	payload, err := svc.FreightInvoicePayload(
		t.Context(),
		pagination.TenantInfo{},
		&invoice.Invoice{
			ID:            pulid.MustNew("inv_"),
			Number:        "INV-4002",
			ShipmentID:    pulid.MustNew("shp_"),
			ShipmentCount: 1,
			PONumber:      "PO-9",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "PO-9", payload.FreightInvoice.PONumber)
	require.Empty(t, payload.FreightInvoice.Shipments)
}

func TestFreightInvoicePayload_AFailedReadFailsThePayload(t *testing.T) {
	t.Parallel()

	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().
		ListSummariesByIDs(mock.Anything, mock.Anything).
		Return(nil, errors.New("connection reset")).
		Once()

	svc := New(Params{ShipmentRepo: shipments})
	_, err := svc.FreightInvoicePayload(t.Context(), pagination.TenantInfo{}, &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		ShipmentCount: 2,
		Lines: []*invoice.InvoiceLine{
			{LineNumber: 1, ShipmentID: pulid.MustNew("shp_")},
			{LineNumber: 2, ShipmentID: pulid.MustNew("shp_")},
		},
	})
	require.ErrorContains(t, err, "connection reset")
}

func TestTenderPayload_QuotesTheSourceShipmentsPurchaseOrder(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	orderID := pulid.MustNew("ord_")

	orders := orderportsmock.NewMockOrderRepository(t)
	orders.EXPECT().
		GetByID(mock.Anything, orderports.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: tenantInfo,
		}).
		Return(&order.Order{ID: orderID, PONumber: " 8978300638 "}, nil).
		Once()

	svc := New(Params{OrderRepo: orders})
	payload, err := svc.TenderPayload(t.Context(), tenantInfo, &shipment.Shipment{
		ID:      pulid.MustNew("shp_"),
		OrderID: orderID,
		BOL:     "BOL-5",
	})
	require.NoError(t, err)
	require.Equal(t, "8978300638", payload.PONumber)
	require.Equal(t, "BOL-5", payload.BOL)
}

func TestTenderPayload_AShipmentWithoutAnOrderReadsNothing(t *testing.T) {
	t.Parallel()

	svc := New(Params{OrderRepo: orderportsmock.NewMockOrderRepository(t)})
	payload, err := svc.TenderPayload(t.Context(), pagination.TenantInfo{}, &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
	})
	require.NoError(t, err)
	require.Empty(t, payload.PONumber)
}

func TestTenderPayload_AMissingOrderQuotesNoPurchaseOrder(t *testing.T) {
	t.Parallel()

	orders := orderportsmock.NewMockOrderRepository(t)
	orders.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Order not found")).
		Once()

	svc := New(Params{OrderRepo: orders})
	payload, err := svc.TenderPayload(t.Context(), pagination.TenantInfo{}, &shipment.Shipment{
		ID:      pulid.MustNew("shp_"),
		OrderID: pulid.MustNew("ord_"),
	})
	require.NoError(t, err)
	require.Empty(t, payload.PONumber)
}

func TestShipmentPurchaseOrder_ReadsTheShipmentsOrder(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("shp_")
	orderID := pulid.MustNew("ord_")

	reader := shipmentserviceportsmock.NewMockShipmentReader(t)
	reader.EXPECT().
		Get(mock.Anything, &shipmentports.GetShipmentByIDRequest{
			ID:         shipmentID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipment.Shipment{ID: shipmentID, OrderID: orderID}, nil).
		Once()
	orders := orderportsmock.NewMockOrderRepository(t)
	orders.EXPECT().
		GetByID(mock.Anything, orderports.GetOrderByIDRequest{ID: orderID, TenantInfo: tenantInfo}).
		Return(&order.Order{ID: orderID, PONumber: "PO-77"}, nil).
		Once()

	svc := New(Params{OrderRepo: orders, ShipmentSvc: reader})
	poNumber, err := svc.ShipmentPurchaseOrder(t.Context(), tenantInfo, shipmentID)
	require.NoError(t, err)
	require.Equal(t, "PO-77", poNumber)
}
