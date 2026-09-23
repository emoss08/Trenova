package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMarkInvoicedLegsKeepsEachShipmentsAdditionalCharges(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("shp_")
	charges := []*shipment.AdditionalCharge{
		{
			ID:                  pulid.MustNew("ac_"),
			ShipmentID:          shipmentID,
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.RequireFromString("112.50"),
			Unit:                1,
		},
		{
			ID:                  pulid.MustNew("ac_"),
			ShipmentID:          shipmentID,
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.RequireFromString("450"),
			Unit:                1,
		},
	}
	loaded := &shipment.Shipment{
		ID:                shipmentID,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Status:            shipment.StatusReadyToInvoice,
		AdditionalCharges: charges,
	}

	repo := mocks.NewMockShipmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
		return req.ID == shipmentID && req.ShipmentOptions.ExpandShipmentDetails
	})).
		Return(loaded, nil).
		Once()

	var persisted *shipment.Shipment
	repo.On("UpdateDerivedState", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*shipment.Shipment)
			require.True(t, ok)
			persisted = entity
		}).
		Return(loaded, nil).
		Once()

	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().
		ListActiveInvoiceItemsByShipmentIDs(mock.Anything, &repositories.ListActiveInvoiceItemsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: []pulid.ID{shipmentID},
		}).
		Return(nil, nil).
		Once()

	svc := &Service{l: zap.NewNop(), shipmentRepo: repo, billingQueueRepo: billingQueueRepo}
	entity := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentID:     shipmentID,
		Lines: []*invoice.InvoiceLine{
			{ShipmentID: shipmentID, Type: invoice.InvoiceLineTypeFreight},
			{ShipmentID: shipmentID, Type: invoice.InvoiceLineTypeAccessorial},
		},
	}

	legs, err := svc.markInvoicedLegs(t.Context(), entity, 1_789_339_461, tenantInfo)
	require.NoError(t, err)

	require.Len(t, legs, 1)
	require.NotNil(t, persisted)
	assert.Equal(t, shipment.StatusInvoiced, persisted.Status)
	require.NotNil(t, persisted.BilledAt)
	assert.Equal(t, int64(1_789_339_461), *persisted.BilledAt)
	assert.Equal(t, charges, persisted.AdditionalCharges)
}
