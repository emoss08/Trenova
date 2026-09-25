package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type holdFixture struct {
	tenantInfo pagination.TenantInfo
	item       *billingqueue.BillingQueueItem
	repo       *mocks.MockBillingQueueRepository
	invoices   *mocks.MockInvoiceService
	detention  *mocks.MockDetentionBillingService
	svc        *service
}

func newHoldFixture(t *testing.T, billType billingqueue.BillType) *holdFixture {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	billerID := pulid.MustNew("usr_")
	f := &holdFixture{
		tenantInfo: tenantInfo,
		item: &billingqueue.BillingQueueItem{
			ID:               pulid.MustNew("bqi_"),
			OrganizationID:   tenantInfo.OrgID,
			BusinessUnitID:   tenantInfo.BuID,
			ShipmentID:       pulid.MustNew("shp_"),
			BillToCustomerID: pulid.MustNew("cus_"),
			AssignedBillerID: &billerID,
			Status:           billingqueue.StatusInReview,
			BillType:         billType,
		},
		repo:      mocks.NewMockBillingQueueRepository(t),
		invoices:  mocks.NewMockInvoiceService(t),
		detention: mocks.NewMockDetentionBillingService(t),
	}

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()

	f.svc = &service{
		l:                zap.NewNop(),
		db:               passthroughDB{},
		repo:             f.repo,
		invoiceSvc:       f.invoices,
		auditService:     audit,
		realtime:         realtime,
		validator:        testValidator(),
		detentionBilling: f.detention,
	}

	return f
}

func (f *holdFixture) expectRead() {
	f.repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetBillingQueueItemByIDRequest) (*billingqueue.BillingQueueItem, error) {
			clone := *f.item
			return &clone, nil
		}).
		Once()
}

func (f *holdFixture) expectHolds(holds ...*detention.DetentionOccurrence) {
	f.detention.EXPECT().
		HoldsForShipments(mock.Anything, &services.DetentionBillingHoldsRequest{
			TenantInfo:  f.tenantInfo,
			ShipmentIDs: []pulid.ID{f.item.ShipmentID},
		}).
		Return(holds, nil).
		Once()
}

func (f *holdFixture) heldCharge() *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:               pulid.MustNew("dto_"),
		OrganizationID:   f.tenantInfo.OrgID,
		BusinessUnitID:   f.tenantInfo.BuID,
		ShipmentID:       f.item.ShipmentID,
		Status:           detention.OccurrenceStatusPending,
		RequiresApproval: true,
		BillableAmount:   decimal.NewFromInt(450),
		Currency:         "USD",
	}
}

func (f *holdFixture) approve(t *testing.T) (*billingqueue.BillingQueueItem, error) {
	t.Helper()

	return f.svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:     f.item.ID,
		NewStatus:  billingqueue.StatusApproved,
		TenantInfo: f.tenantInfo,
	}, userActor(f.tenantInfo))
}

func TestUpdateStatus_RefusesApprovalWhileADetentionChargeAwaitsApproval(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	held := f.heldCharge()
	f.expectRead()
	f.expectHolds(held)

	updated, err := f.approve(t)

	require.Error(t, err)
	assert.Nil(t, updated)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), "1 detention charge on this shipment still needs approval")
	assert.Equal(t, held.ID.String(), business.Params["detentionOccurrenceIds"])
	assert.Equal(t, f.item.ShipmentID.String(), business.Params["shipmentIds"])
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	f.invoices.AssertNotCalled(
		t, "CreateFromApprovedBillingQueueItem", mock.Anything, mock.Anything, mock.Anything,
	)
}

func TestUpdateStatus_ApprovesWhenNoDetentionChargeIsHeld(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectRead()
	f.expectHolds()
	f.repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()
	f.invoices.EXPECT().
		CreateFromApprovedBillingQueueItem(mock.Anything, mock.Anything, mock.Anything).
		Return(&services.CreateInvoiceFromBillingQueueResult{
			Invoice: &invoice.Invoice{ID: pulid.MustNew("inv_")},
		}, nil).
		Once()

	updated, err := f.approve(t)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusApproved, updated.Status)
}

func TestUpdateStatus_CreditMemoApprovalIsNeverHeldByDetention(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeCreditMemo)
	f.expectRead()
	f.repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()
	f.invoices.EXPECT().
		CreateFromApprovedBillingQueueItem(mock.Anything, mock.Anything, mock.Anything).
		Return(&services.CreateInvoiceFromBillingQueueResult{}, nil).
		Once()

	updated, err := f.approve(t)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusApproved, updated.Status)
	f.detention.AssertNotCalled(t, "HoldsForShipments", mock.Anything, mock.Anything)
}

func TestAutoApprove_LeavesAHeldItemInReviewWithoutFailing(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectHolds(f.heldCharge())

	result := f.svc.autoApprove(t.Context(), f.item, f.tenantInfo, userActor(f.tenantInfo))

	assert.Same(t, f.item, result)
	assert.Equal(t, billingqueue.StatusInReview, result.Status)
	f.repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestGetByID_ListsTheDetentionChargesHoldingTheItem(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	held := f.heldCharge()
	held.LocationName = "Riverside DC"
	f.expectRead()
	f.expectHolds(held)
	shipments := mocks.NewMockShipmentRepository(t)
	shipments.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("shipment not found")).
		Once()
	f.svc.shipmentRepo = shipments

	item, err := f.svc.GetByID(t.Context(), &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                f.item.ID,
		TenantInfo:            f.tenantInfo,
		ExpandShipmentDetails: true,
	})

	require.NoError(t, err)
	require.Len(t, item.DetentionHolds, 1)
	hold := item.DetentionHolds[0]
	assert.Equal(t, held.ID, hold.OccurrenceID)
	assert.Equal(t, "Riverside DC", hold.LocationName)
	assert.True(t, decimal.NewFromInt(450).Equal(hold.BillableAmount))
	assert.Equal(t, detention.BillingHoldReasonEscalated, hold.Reason)
}
