package invoicerunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolveGroupLegsSkipsAnItemReallocatedToAnotherPayer(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	groupPayer := pulid.MustNew("cus_")
	otherPayer := pulid.MustNew("cus_")
	group := &invoicerun.InvoiceRunGroup{ID: pulid.MustNew("invrg_"), CustomerID: groupPayer}
	item := &invoicerun.InvoiceRunGroupItem{
		ID:                 pulid.MustNew("invri_"),
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
	}

	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().
		GetByID(mock.Anything, &repositories.GetBillingQueueItemByIDRequest{
			ItemID:     item.BillingQueueItemID,
			TenantInfo: tenantInfo,
		}).
		Return(&billingqueue.BillingQueueItem{
			ID:               item.BillingQueueItemID,
			ShipmentID:       item.ShipmentID,
			BillToCustomerID: otherPayer,
			Status:           billingqueue.StatusApproved,
		}, nil).
		Once()

	svc := &Service{l: zap.NewNop(), billingQueueRepo: billingQueueRepo}

	legs, items, reason, err := svc.resolveGroupLegs(
		t.Context(),
		tenantInfo,
		group,
		[]*invoicerun.InvoiceRunGroupItem{item},
	)
	require.NoError(t, err)
	assert.Nil(t, legs)
	assert.Nil(t, items)
	assert.Contains(t, reason, "re-allocated to another payer")
}

func TestResolveGroupLegsLoadsLegsForThePayersItems(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	payer := pulid.MustNew("cus_")
	group := &invoicerun.InvoiceRunGroup{ID: pulid.MustNew("invrg_"), CustomerID: payer}
	item := &invoicerun.InvoiceRunGroupItem{
		ID:                 pulid.MustNew("invri_"),
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
	}
	queueItem := &billingqueue.BillingQueueItem{
		ID:               item.BillingQueueItemID,
		ShipmentID:       item.ShipmentID,
		BillToCustomerID: payer,
		Status:           billingqueue.StatusApproved,
	}
	leg := &shipment.Shipment{ID: item.ShipmentID}

	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(queueItem, nil).Once()
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == item.ShipmentID && req.ShipmentOptions.ExpandShipmentDetails
		})).
		Return(leg, nil).
		Once()

	svc := &Service{l: zap.NewNop(), billingQueueRepo: billingQueueRepo, shipmentRepo: shipmentRepo}

	legs, items, reason, err := svc.resolveGroupLegs(
		t.Context(),
		tenantInfo,
		group,
		[]*invoicerun.InvoiceRunGroupItem{item},
	)
	require.NoError(t, err)
	assert.Empty(t, reason)
	require.Len(t, legs, 1)
	require.Len(t, items, 1)
	assert.Same(t, leg, legs[0])
	assert.Same(t, queueItem, items[0])
}
