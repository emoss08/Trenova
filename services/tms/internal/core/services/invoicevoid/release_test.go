package invoicevoid_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/invoicevoid"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type releaseFixture struct {
	tenantInfo   pagination.TenantInfo
	userID       pulid.ID
	inv          *invoice.Invoice
	shipmentID   pulid.ID
	orderID      pulid.ID
	queueRepo    *mocks.MockBillingQueueRepository
	orderRepo    *mocks.MockOrderRepository
	allocRepo    *mocks.MockChargeAllocationRepository
	shipmentRepo *mocks.MockShipmentRepository
	invoiceRepo  *mocks.MockInvoiceRepository
	derivation   *mocks.MockOrderDerivationService
}

func newReleaseFixture(t *testing.T) *releaseFixture {
	t.Helper()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	f := &releaseFixture{
		tenantInfo:   pagination.TenantInfo{OrgID: orgID, BuID: buID},
		userID:       pulid.MustNew("usr_"),
		shipmentID:   pulid.MustNew("shp_"),
		orderID:      pulid.MustNew("ord_"),
		queueRepo:    mocks.NewMockBillingQueueRepository(t),
		orderRepo:    mocks.NewMockOrderRepository(t),
		allocRepo:    mocks.NewMockChargeAllocationRepository(t),
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		invoiceRepo:  mocks.NewMockInvoiceRepository(t),
		derivation:   mocks.NewMockOrderDerivationService(t),
	}
	f.inv = &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         f.shipmentID,
		Number:             "INV-1",
		Status:             invoice.StatusVoided,
	}

	return f
}

func (f *releaseFixture) deps() invoicevoid.Deps {
	return invoicevoid.Deps{
		BillingQueueRepo:     f.queueRepo,
		OrderRepo:            f.orderRepo,
		ChargeAllocationRepo: f.allocRepo,
		ShipmentRepo:         f.shipmentRepo,
		InvoiceRepo:          f.invoiceRepo,
		OrderDerivation:      f.derivation,
		Renumber: func(context.Context, billingqueue.BillType) (string, error) {
			return "INV-2", nil
		},
	}
}

func (f *releaseFixture) expectCommonReleases(rebill bool, released ...*billingqueue.BillingQueueItem) {
	f.queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *repositories.ReleaseForInvoiceRequest) bool {
			return req.Rebill == rebill &&
				req.InvoiceID == f.inv.ID &&
				req.AnchorItemID == f.inv.BillingQueueItemID &&
				req.TenantInfo == f.tenantInfo &&
				req.CanceledByID != nil && *req.CanceledByID == f.userID &&
				req.CanceledAt == 1_700_000_000 &&
				req.RenumberFn != nil
		})).
		Return(released, nil).
		Once()
	f.orderRepo.EXPECT().
		ClearChargesInvoice(mock.Anything, &repositories.ClearOrderChargesInvoiceRequest{TenantInfo: f.tenantInfo, InvoiceID: f.inv.ID}).
		Return(1, nil).
		Once()
	f.allocRepo.EXPECT().
		ClearInvoice(mock.Anything, &repositories.ClearChargeAllocationsInvoiceRequest{TenantInfo: f.tenantInfo, InvoiceID: f.inv.ID}).
		Return(2, nil).
		Once()
}

func (f *releaseFixture) params(disposition invoice.VoidDisposition, wasPosted bool) invoicevoid.Params {
	return invoicevoid.Params{
		Invoice:     f.inv,
		Disposition: disposition,
		ActorUserID: f.userID,
		Reason:      "Wrong customer",
		WasPosted:   wasPosted,
		Now:         1_700_000_000,
	}
}

func TestReleaseNoopsOnANilInvoice(t *testing.T) {
	t.Parallel()

	released, err := invoicevoid.Release(t.Context(), invoicevoid.Deps{}, invoicevoid.Params{})

	require.NoError(t, err)
	assert.Nil(t, released)
}

func TestReleaseDraftRebillReleasesItemsWithoutTouchingShipments(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	first := pulid.MustNew("bqi_")
	second := pulid.MustNew("bqi_")
	f.expectCommonReleases(true,
		&billingqueue.BillingQueueItem{ID: first},
		nil,
		&billingqueue.BillingQueueItem{ID: second},
	)

	released, err := invoicevoid.Release(t.Context(), f.deps(), f.params(invoice.VoidDispositionRebill, false))

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{first, second}, released, "nil rows are skipped")
	f.shipmentRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestReleasePostedRebillReturnsShipmentsToReadyToInvoice(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(true)
	billedAt := int64(1_699_000_000)
	shp := &shipment.Shipment{ID: f.shipmentID, OrderID: f.orderID, Status: shipment.StatusInvoiced, BilledAt: &billedAt}
	f.invoiceRepo.EXPECT().
		ListByShipmentIDs(mock.Anything, repositories.ListInvoicesByShipmentIDsRequest{TenantInfo: f.tenantInfo, ShipmentIDs: []pulid.ID{f.shipmentID}}).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).
		Once()
	f.shipmentRepo.EXPECT().
		GetByID(mock.Anything, &repositories.GetShipmentByIDRequest{
			ID:              f.shipmentID,
			TenantInfo:      f.tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
		}).
		Return(shp, nil).
		Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.ID == f.shipmentID && updated.Status == shipment.StatusReadyToInvoice && updated.BilledAt == nil
		})).
		Return(shp, nil).
		Once()
	f.derivation.EXPECT().RecomputeOrder(mock.Anything, f.tenantInfo, f.orderID).Return(nil).Once()

	_, err := invoicevoid.Release(t.Context(), f.deps(), f.params(invoice.VoidDispositionRebill, true))

	require.NoError(t, err)
}

func TestReleasePostedDoNotRebillCompletesShipments(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(false)
	shp := &shipment.Shipment{ID: f.shipmentID, Status: shipment.StatusInvoiced}
	f.invoiceRepo.EXPECT().ListByShipmentIDs(mock.Anything, mock.Anything).Return(map[pulid.ID][]*invoice.Invoice{}, nil).Once()
	f.shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(shp, nil).Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.Status == shipment.StatusCompleted && updated.BilledAt == nil
		})).
		Return(shp, nil).
		Once()

	_, err := invoicevoid.Release(t.Context(), f.deps(), f.params(invoice.VoidDispositionDoNotRebill, true))

	require.NoError(t, err)
	f.derivation.AssertNotCalled(t, "RecomputeOrder", mock.Anything, mock.Anything, mock.Anything)
}

func TestReleaseDoNotRebillKeepsALegAnotherPayersInvoiceStillBills(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(false)
	shp := &shipment.Shipment{ID: f.shipmentID, Status: shipment.StatusInvoiced}
	f.invoiceRepo.EXPECT().ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{
			f.shipmentID: {
				f.inv,
				{ID: pulid.MustNew("inv_"), Status: invoice.StatusPosted, BillType: billingqueue.BillTypeCreditMemo},
				{ID: pulid.MustNew("inv_"), Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice},
			},
		}, nil).
		Once()
	f.shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(shp, nil).Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.Status == shipment.StatusInvoiced && updated.BilledAt == nil
		})).
		Return(shp, nil).
		Once()

	_, err := invoicevoid.Release(t.Context(), f.deps(), f.params(invoice.VoidDispositionDoNotRebill, true))

	require.NoError(t, err)
}

func TestReleaseCancelReasonIsPrefixedAndCapped(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	var got string
	f.queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.ReleaseForInvoiceRequest) ([]*billingqueue.BillingQueueItem, error) {
			got = req.CancelReason
			return nil, nil
		}).
		Once()

	params := f.params(invoice.VoidDispositionDoNotRebill, false)
	params.Reason = strings.Repeat("x", 200)
	_, err := invoicevoid.Release(t.Context(), invoicevoid.Deps{BillingQueueRepo: f.queueRepo}, params)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "Invoice voided: "))
	assert.Len(t, got, 100)
}
