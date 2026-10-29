package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func unheldShipments(t *testing.T) *shipmentportsmock.MockShipmentRepository {
	t.Helper()

	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return &shipment.Shipment{ID: req.ID}, nil
		}).
		Maybe()
	return shipments
}

func reviewedShipment(id pulid.ID, reviews ...string) *shipment.Shipment {
	bill := &tarifftypes.Bill{Currency: "USD"}
	for _, text := range reviews {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &shipment.Shipment{
		ID:        id,
		ProNumber: "PRO-77",
		RatingDetail: &shipment.RatingDetail{
			Bill:                bill,
			RatedAt:             1_800_000_000,
			FormulaTemplateName: "Dock rates",
		},
	}
}

func (f *holdFixture) heldForReview(t *testing.T, reviews ...string) *shipment.Shipment {
	t.Helper()

	held := reviewedShipment(f.item.ShipmentID, reviews...)
	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == f.item.ShipmentID && req.TenantInfo.OrgID == f.tenantInfo.OrgID
		})).
		Return(held, nil).
		Maybe()
	f.svc.shipmentRepo = shipments
	return held
}

func TestUpdateStatus_RefusesApprovalWhileTheTariffHoldsTheRatingForReview(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectRead()
	f.expectHolds()
	f.heldForReview(t, "Holiday pickup")

	updated, err := f.approve(t)

	require.Error(t, err)
	assert.Nil(t, updated)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, f.item.ShipmentID.String(), business.Params["shipmentIds"])
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	f.invoices.AssertNotCalled(
		t, "CreateFromApprovedBillingQueueItem", mock.Anything, mock.Anything, mock.Anything,
	)
}

func TestUpdateStatus_ApprovesOnceTheReviewedRatingIsApproved(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectRead()
	f.expectHolds()
	held := f.heldForReview(t, "Holiday pickup")
	_, err := shipmentrating.ApproveReview(held.RatingDetail, pulid.MustNew("usr_"), 1_800_000_100)
	require.NoError(t, err)
	f.repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()
	f.invoices.EXPECT().
		CreateFromApprovedBillingQueueItem(mock.Anything, mock.Anything, mock.Anything).
		Return(&invoiceports.CreateInvoiceFromBillingQueueResult{}, nil).
		Once()

	updated, err := f.approve(t)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusApproved, updated.Status)
}

func TestUpdateStatus_CreditMemoApprovalIsNeverHeldForARatingReview(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeCreditMemo)
	f.expectRead()
	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	f.svc.shipmentRepo = shipments
	f.repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()
	f.invoices.EXPECT().
		CreateFromApprovedBillingQueueItem(mock.Anything, mock.Anything, mock.Anything).
		Return(&invoiceports.CreateInvoiceFromBillingQueueResult{}, nil).
		Once()

	updated, err := f.approve(t)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusApproved, updated.Status)
	shipments.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestAutoApprove_LeavesARatingHeldForReviewInReview(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectHolds()
	f.heldForReview(t, "Holiday pickup")

	result := f.svc.autoApprove(t.Context(), f.item, f.tenantInfo, userActor(f.tenantInfo))

	assert.Same(t, f.item, result)
	assert.Equal(t, billingqueue.StatusInReview, result.Status)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestGetByID_ShowsTheRatingReviewHoldingTheItem(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.expectRead()
	f.expectHolds()
	f.heldForReview(t, "Holiday pickup", "Liftgate on a dock")

	item, err := f.svc.GetByID(t.Context(), &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                f.item.ID,
		TenantInfo:            f.tenantInfo,
		ExpandShipmentDetails: true,
	})

	require.NoError(t, err)
	require.NotNil(t, item.RatingReviewHold)
	assert.Equal(t, &billingqueue.RatingReviewHold{
		ShipmentID:          f.item.ShipmentID,
		Reasons:             []string{"Holiday pickup", "Liftgate on a dock"},
		RatedAt:             1_800_000_000,
		FormulaTemplateName: "Dock rates",
	}, item.RatingReviewHold)
}
