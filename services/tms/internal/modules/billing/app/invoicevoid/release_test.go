package invoicevoid_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevoid"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/chargeallocationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/detentionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/orderportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
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
	queueRepo    *billingqueueportsmock.MockBillingQueueRepository
	orderRepo    *orderportsmock.MockOrderRepository
	allocRepo    *chargeallocationportsmock.MockChargeAllocationRepository
	shipmentRepo *shipmentportsmock.MockShipmentRepository
	invoiceRepo  *invoiceportsmock.MockInvoiceRepository
	derivation   *orderportsmock.MockOrderDerivationService
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
		queueRepo:    billingqueueportsmock.NewMockBillingQueueRepository(t),
		orderRepo:    orderportsmock.NewMockOrderRepository(t),
		allocRepo:    chargeallocationportsmock.NewMockChargeAllocationRepository(t),
		shipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		invoiceRepo:  invoiceportsmock.NewMockInvoiceRepository(t),
		derivation:   orderportsmock.NewMockOrderDerivationService(t),
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
		Renumber: func(context.Context, *billingqueue.BillingQueueItem) (string, error) {
			return "INV-2", nil
		},
	}
}

func (f *releaseFixture) expectCommonReleases(
	rebill bool,
	released ...*billingqueue.BillingQueueItem,
) {
	f.queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *billingqueueports.ReleaseForInvoiceRequest) bool {
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
	f.invoiceRepo.EXPECT().
		RevokeDocumentShareTokens(mock.Anything, invoiceports.RevokeInvoiceDocumentShareTokensRequest{
			InvoiceID:  f.inv.ID,
			TenantInfo: f.tenantInfo,
		}).
		Return(int64(1), nil).
		Once()
	f.orderRepo.EXPECT().
		ClearChargesInvoice(mock.Anything, &orderports.ClearOrderChargesInvoiceRequest{TenantInfo: f.tenantInfo, InvoiceID: f.inv.ID}).
		Return(1, nil).
		Once()
	f.allocRepo.EXPECT().
		ClearInvoice(mock.Anything, &chargeallocationports.ClearChargeAllocationsInvoiceRequest{TenantInfo: f.tenantInfo, InvoiceID: f.inv.ID}).
		Return(2, nil).
		Once()
}

func (f *releaseFixture) params(
	disposition invoice.VoidDisposition,
	wasPosted bool,
) invoicevoid.Params {
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

	released, err := invoicevoid.Release(
		t.Context(),
		f.deps(),
		f.params(invoice.VoidDispositionRebill, false),
	)

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{first, second}, released, "nil rows are skipped")
	f.shipmentRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestReleasePostedRebillReturnsShipmentsToReadyToInvoice(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(true)
	billedAt := int64(1_699_000_000)
	shp := &shipment.Shipment{
		ID:         f.shipmentID,
		OrderID:    f.orderID,
		Status:     shipmentvocab.StatusInvoiced,
		BilledAt:   &billedAt,
		RateLocked: true,
	}
	f.invoiceRepo.EXPECT().
		ListByShipmentIDs(mock.Anything, invoiceports.ListInvoicesByShipmentIDsRequest{TenantInfo: f.tenantInfo, ShipmentIDs: []pulid.ID{f.shipmentID}}).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).
		Once()
	f.shipmentRepo.EXPECT().
		GetByID(mock.Anything, &shipmentports.GetShipmentByIDRequest{
			ID:              f.shipmentID,
			TenantInfo:      f.tenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
		}).
		Return(shp, nil).
		Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.ID == f.shipmentID && updated.Status == shipmentvocab.StatusReadyToInvoice &&
				updated.BilledAt == nil &&
				!updated.RateLocked
		})).
		Return(shp, nil).
		Once()
	f.derivation.EXPECT().RecomputeOrder(mock.Anything, f.tenantInfo, f.orderID).Return(nil).Once()

	_, err := invoicevoid.Release(
		t.Context(),
		f.deps(),
		f.params(invoice.VoidDispositionRebill, true),
	)

	require.NoError(t, err)
}

func TestReleasePostedDoNotRebillCompletesShipments(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(false)
	shp := &shipment.Shipment{
		ID:         f.shipmentID,
		Status:     shipmentvocab.StatusInvoiced,
		RateLocked: true,
	}
	f.invoiceRepo.EXPECT().
		ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).
		Once()
	f.shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(shp, nil).Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.Status == shipmentvocab.StatusCompleted && updated.BilledAt == nil &&
				!updated.RateLocked
		})).
		Return(shp, nil).
		Once()

	_, err := invoicevoid.Release(
		t.Context(),
		f.deps(),
		f.params(invoice.VoidDispositionDoNotRebill, true),
	)

	require.NoError(t, err)
	f.derivation.AssertNotCalled(t, "RecomputeOrder", mock.Anything, mock.Anything, mock.Anything)
}

func TestReleaseDoNotRebillKeepsALegAnotherPayersInvoiceStillBills(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(false)
	shp := &shipment.Shipment{
		ID:         f.shipmentID,
		Status:     shipmentvocab.StatusInvoiced,
		RateLocked: true,
	}
	f.invoiceRepo.EXPECT().ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{
			f.shipmentID: {
				f.inv,
				{
					ID:       pulid.MustNew("inv_"),
					Status:   invoice.StatusPosted,
					BillType: billingqueue.BillTypeCreditMemo,
				},
				{
					ID:       pulid.MustNew("inv_"),
					Status:   invoice.StatusPosted,
					BillType: billingqueue.BillTypeInvoice,
				},
			},
		}, nil).
		Once()
	f.shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(shp, nil).Once()
	f.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(updated *shipment.Shipment) bool {
			return updated.Status == shipmentvocab.StatusInvoiced && updated.BilledAt == nil &&
				updated.RateLocked
		})).
		Return(shp, nil).
		Once()

	_, err := invoicevoid.Release(
		t.Context(),
		f.deps(),
		f.params(invoice.VoidDispositionDoNotRebill, true),
	)

	require.NoError(t, err)
}

func TestReleaseCancelReasonIsPrefixedAndCapped(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	var got string
	f.queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *billingqueueports.ReleaseForInvoiceRequest) ([]*billingqueue.BillingQueueItem, error) {
			got = req.CancelReason
			return nil, nil
		}).
		Once()

	params := f.params(invoice.VoidDispositionDoNotRebill, false)
	params.Reason = strings.Repeat("x", 200)
	_, err := invoicevoid.Release(
		t.Context(),
		invoicevoid.Deps{BillingQueueRepo: f.queueRepo},
		params,
	)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "Invoice voided: "))
	assert.Len(t, got, 100)
}

func TestReleaseSettlesTheDetentionChargesTheVoidedInvoiceBilled(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(false)
	detention := detentionportsmock.NewMockDetentionBillingService(t)
	detention.EXPECT().
		SyncInvoiceBilling(mock.Anything, &detentionports.SyncDetentionInvoiceBillingRequest{
			TenantInfo:    f.tenantInfo,
			InvoiceIDs:    []pulid.ID{f.inv.ID},
			InvoiceNumber: f.inv.Number,
			Event:         detentionports.DetentionBillingInvoiceVoided,
			ActorUserID:   f.userID,
		}).
		Return(nil).
		Once()
	deps := f.deps()
	deps.DetentionBilling = detention

	_, err := invoicevoid.Release(
		t.Context(),
		deps,
		f.params(invoice.VoidDispositionDoNotRebill, false),
	)

	require.NoError(t, err)
}

func TestReleaseFailsWhenTheDetentionChargesCannotBeSettled(t *testing.T) {
	t.Parallel()

	f := newReleaseFixture(t)
	f.expectCommonReleases(true)
	detention := detentionportsmock.NewMockDetentionBillingService(t)
	detention.EXPECT().
		SyncInvoiceBilling(mock.Anything, mock.Anything).
		Return(errors.New("occurrence changed underneath")).
		Once()
	deps := f.deps()
	deps.DetentionBilling = detention

	_, err := invoicevoid.Release(
		t.Context(),
		deps,
		f.params(invoice.VoidDispositionRebill, true),
	)

	require.Error(t, err)
	f.shipmentRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}
