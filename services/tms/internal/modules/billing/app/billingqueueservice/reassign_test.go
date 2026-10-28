package billingqueueservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/chargeallocationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// splitShipment is the shipment from the field report: $2,850 freight split by
// amount between Peak ($1,500) and Acme ($1,350), and a $1,154.38 detention
// charge that stays whole with Acme, the shipment's customer.
type splitShipment struct {
	tenantInfo pagination.TenantInfo
	shipment   *shipment.Shipment
	acme       pulid.ID
	peak       pulid.ID
	detention  *shipment.AdditionalCharge
	acmeItem   *billingqueue.BillingQueueItem
	peakItem   *billingqueue.BillingQueueItem
}

func newSplitShipment() *splitShipment {
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	shipmentID := pulid.MustNew("shp_")
	detention := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.RequireFromString("1154.38"),
		Unit:                1,
		AccessorialCharge: &accessorialcharge.AccessorialCharge{
			Code:        "DET",
			Description: "Detention Fee",
		},
	}
	shp := &shipment.Shipment{
		ID:                  shipmentID,
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		CustomerID:          acme,
		Status:              shipment.StatusReadyToInvoice,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(2850)),
		TariffInputs:        tarifftypes.Inputs{"rate": rules.MoneyValue(decimal.NewFromInt(2850))},
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("1154.38")),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("4004.38")),
		AdditionalCharges:   []*shipment.AdditionalCharge{detention},
		ChargeAllocations: []*shipment.ChargeAllocation{
			{
				ID:               pulid.MustNew("chal_"),
				OrganizationID:   tenantInfo.OrgID,
				BusinessUnitID:   tenantInfo.BuID,
				ShipmentID:       &shipmentID,
				ChargeKind:       shipment.ChargeAllocationKindFreight,
				BillToCustomerID: peak,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(decimal.NewFromInt(1500)),
			},
			{
				ID:               pulid.MustNew("chal_"),
				OrganizationID:   tenantInfo.OrgID,
				BusinessUnitID:   tenantInfo.BuID,
				ShipmentID:       &shipmentID,
				ChargeKind:       shipment.ChargeAllocationKindFreight,
				BillToCustomerID: acme,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(decimal.NewFromInt(1350)),
				Sequence:         1,
			},
		},
	}

	item := func(payer pulid.ID, number string) *billingqueue.BillingQueueItem {
		return &billingqueue.BillingQueueItem{
			ID:               pulid.MustNew("bqi_"),
			OrganizationID:   tenantInfo.OrgID,
			BusinessUnitID:   tenantInfo.BuID,
			ShipmentID:       shipmentID,
			BillToCustomerID: payer,
			Status:           billingqueue.StatusInReview,
			BillType:         billingqueue.BillTypeInvoice,
			Number:           number,
		}
	}

	return &splitShipment{
		tenantInfo: tenantInfo,
		shipment:   shp,
		acme:       acme,
		peak:       peak,
		detention:  detention,
		acmeItem:   item(acme, "INV-22"),
		peakItem:   item(peak, "INV-23"),
	}
}

func testActor(tenantInfo pagination.TenantInfo) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func activeCustomers(tenantInfo pagination.TenantInfo, ids ...pulid.ID) []*customer.Customer {
	out := make([]*customer.Customer, 0, len(ids))
	for _, id := range ids {
		out = append(out, &customer.Customer{
			ID:             id,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         domaintypes.StatusActive,
		})
	}
	return out
}

func TestReconcileQueueItems_CreatesForNewPayersAndCancelsForPayersWithNothing(t *testing.T) {
	t.Parallel()

	f := newSplitShipment()
	carrier := pulid.MustNew("cus_")
	resolution := &shipment.ShareResolution{Shares: []*shipment.PayerShare{
		{PayerID: f.acme},
		{
			PayerID: f.peak,
			Charges: []shipment.AllocatedCharge{{Kind: shipment.ChargeAllocationKindFreight}},
		},
		{
			PayerID: carrier,
			Charges: []shipment.AllocatedCharge{{Kind: shipment.ChargeAllocationKindAccessorial}},
		},
	}}

	toCreate, toCancel := reconcileQueueItems(
		resolution,
		[]*billingqueue.BillingQueueItem{f.acmeItem, f.peakItem},
	)

	require.Len(t, toCreate, 1)
	assert.Equal(t, carrier, toCreate[0].PayerID)
	require.Len(t, toCancel, 1)
	assert.Equal(t, f.acmeItem.ID, toCancel[0].ID, "a payer whose share is empty loses their item")
}

func TestReassignRows_WholeChargeToTheShipmentPayerIsNoRows(t *testing.T) {
	t.Parallel()

	f := newSplitShipment()
	rows, err := reassignRows(t.Context(), f.shipment, &billingqueueports.ReassignChargeRequest{
		ChargeKind: shipment.ChargeAllocationKindFreight,
		Allocations: []*shipment.ChargeAllocation{{
			BillToCustomerID: f.acme,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
		}},
	}, nil)

	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestReassignRows_RefusesARowIDFromAnotherCharge(t *testing.T) {
	t.Parallel()

	f := newSplitShipment()
	_, err := reassignRows(t.Context(), f.shipment, &billingqueueports.ReassignChargeRequest{
		ChargeKind:         shipment.ChargeAllocationKindAccessorial,
		AdditionalChargeID: f.detention.ID,
		Allocations: []*shipment.ChargeAllocation{{
			ID:               f.shipment.ChargeAllocations[0].ID,
			BillToCustomerID: f.peak,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
		}},
	}, f.detention)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this charge")
}

func TestReassignRows_KeepsTheIdentityOfARowAlreadyOnTheCharge(t *testing.T) {
	t.Parallel()

	f := newSplitShipment()
	existing := f.shipment.ChargeAllocations[0]
	existing.Version = 4
	rows, err := reassignRows(t.Context(), f.shipment, &billingqueueports.ReassignChargeRequest{
		ChargeKind: shipment.ChargeAllocationKindFreight,
		Allocations: []*shipment.ChargeAllocation{
			{
				ID:               existing.ID,
				BillToCustomerID: f.peak,
				Method:           shipment.ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(50)),
			},
			{
				BillToCustomerID: f.acme,
				Method:           shipment.ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(50)),
			},
		},
	}, nil)

	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, existing.ID, rows[0].ID)
	assert.Equal(t, int64(4), rows[0].Version)
	assert.True(t, rows[1].ID.IsNil())
	assert.Equal(t, int16(1), rows[1].Sequence)
	assert.Equal(t, f.shipment.ID, pulid.ConvertFromPtr(rows[1].ShipmentID))
}

type reassignHarness struct {
	f            *splitShipment
	svc          *service
	repo         *billingqueueportsmock.MockBillingQueueRepository
	shipmentRepo *shipmentportsmock.MockShipmentRepository
	customerRepo *customerportsmock.MockCustomerRepository
	invoiceRepo  *invoiceportsmock.MockInvoiceRepository
	allocRepo    *chargeallocationportsmock.MockChargeAllocationRepository
}

func newReassignHarness(t *testing.T) *reassignHarness {
	t.Helper()
	f := newSplitShipment()
	h := &reassignHarness{
		f:            f,
		repo:         billingqueueportsmock.NewMockBillingQueueRepository(t),
		shipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		customerRepo: customerportsmock.NewMockCustomerRepository(t),
		invoiceRepo:  invoiceportsmock.NewMockInvoiceRepository(t),
		allocRepo:    chargeallocationportsmock.NewMockChargeAllocationRepository(t),
	}
	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()

	h.svc = &service{
		l:                    zap.NewNop(),
		db:                   passthroughDB{},
		repo:                 h.repo,
		shipmentRepo:         h.shipmentRepo,
		customerRepo:         h.customerRepo,
		invoiceRepo:          h.invoiceRepo,
		chargeAllocationRepo: h.allocRepo,
		generator:            &countingGenerator{},
		auditService:         audit,
		realtime:             realtime,
		validator:            testValidator(),
	}

	h.invoiceRepo.EXPECT().
		ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).
		Maybe()
	h.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			return cloneShipment(f.shipment), nil
		}).
		Maybe()
	h.customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return(activeCustomers(f.tenantInfo, f.acme, f.peak), nil).
		Maybe()
	h.customerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error) {
			return &customer.Customer{
				ID:             req.ID,
				BillingProfile: &customer.CustomerBillingProfile{},
			}, nil
		}).
		Maybe()
	h.allocRepo.EXPECT().LockedIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(map[pulid.ID]struct{}{}, nil).
		Maybe()

	return h
}

func cloneShipment(src *shipment.Shipment) *shipment.Shipment {
	copied := *src
	copied.AdditionalCharges = append([]*shipment.AdditionalCharge(nil), src.AdditionalCharges...)
	copied.ChargeAllocations = make([]*shipment.ChargeAllocation, 0, len(src.ChargeAllocations))
	for _, row := range src.ChargeAllocations {
		clone := *row
		copied.ChargeAllocations = append(copied.ChargeAllocations, &clone)
	}
	return &copied
}

func (h *reassignHarness) expectItem(item *billingqueue.BillingQueueItem) {
	h.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *billingqueueports.GetBillingQueueItemByIDRequest) bool {
			return req.ItemID == item.ID
		})).
		RunAndReturn(func(context.Context, *billingqueueports.GetBillingQueueItemByIDRequest) (*billingqueue.BillingQueueItem, error) {
			copied := *item
			return &copied, nil
		})
}

func (h *reassignHarness) expectActive(items ...*billingqueue.BillingQueueItem) {
	h.repo.EXPECT().
		ListActiveInvoiceItemsByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*billingqueue.BillingQueueItem{h.f.shipment.ID: items}, nil)
}

func TestReassignCharge_MovingDetentionToPeakKeepsBothItemsAndSyncsTheSplit(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	h.expectItem(f.acmeItem)
	h.expectActive(f.acmeItem, f.peakItem)

	var synced []*shipment.ChargeAllocation
	h.allocRepo.EXPECT().
		SyncForShipment(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, shp *shipment.Shipment) error {
			synced = shp.ChargeAllocations
			return nil
		}).
		Once()

	result, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:             f.acmeItem.ID,
		TenantInfo:         f.tenantInfo,
		ChargeKind:         shipment.ChargeAllocationKindAccessorial,
		AdditionalChargeID: f.detention.ID,
		Allocations: []*shipment.ChargeAllocation{{
			BillToCustomerID: f.peak,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
		}},
	}, testActor(f.tenantInfo))

	require.NoError(t, err)
	assert.Empty(t, result.CreatedItemIDs)
	assert.Empty(t, result.CanceledItemIDs)
	require.Len(t, synced, 3, "the two freight rows stay and detention gains one row")
	var detentionRow *shipment.ChargeAllocation
	for _, row := range synced {
		if row.ChargeKind == shipment.ChargeAllocationKindAccessorial {
			detentionRow = row
		}
	}
	require.NotNil(t, detentionRow)
	assert.Equal(t, f.peak, detentionRow.BillToCustomerID)
	assert.Equal(t, f.detention.ID, pulid.ConvertFromPtr(detentionRow.AdditionalChargeID))
	require.NotNil(t, result.Item.PayerShare)
}

func TestReassignCharge_GivingPeakFreightBackToAcmeCancelsPeaksItem(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	h.expectItem(f.peakItem)
	h.expectActive(f.acmeItem, f.peakItem)
	h.allocRepo.EXPECT().
		SyncForShipment(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	var updated *billingqueue.BillingQueueItem
	h.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, item *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			updated = item
			return item, nil
		}).
		Once()

	result, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:      f.peakItem.ID,
		TenantInfo:  f.tenantInfo,
		ChargeKind:  shipment.ChargeAllocationKindFreight,
		Allocations: []*shipment.ChargeAllocation{},
	}, testActor(f.tenantInfo))

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, f.peakItem.ID, updated.ID)
	assert.Equal(t, billingqueue.StatusCanceled, updated.Status)
	assert.Equal(t, reassignCancelReason, updated.CancelReason)
	assert.Equal(t, []pulid.ID{f.peakItem.ID}, result.CanceledItemIDs)
	assert.Empty(t, result.CreatedItemIDs)
}

func TestReassignCharge_ANewPayerGetsTheirOwnQueueItem(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	carrier := pulid.MustNew("cus_")
	h.customerRepo.ExpectedCalls = nil
	h.customerRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		Return(activeCustomers(f.tenantInfo, f.acme, f.peak, carrier), nil).Maybe()
	h.customerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error) {
			return &customer.Customer{
				ID:             req.ID,
				BillingProfile: &customer.CustomerBillingProfile{},
			}, nil
		}).
		Maybe()
	h.expectItem(f.acmeItem)
	h.expectActive(f.acmeItem, f.peakItem)
	h.allocRepo.EXPECT().
		SyncForShipment(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	var created *billingqueue.BillingQueueItem
	h.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, item *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			item.ID = pulid.MustNew("bqi_")
			created = item
			return item, nil
		}).
		Once()

	result, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:             f.acmeItem.ID,
		TenantInfo:         f.tenantInfo,
		ChargeKind:         shipment.ChargeAllocationKindAccessorial,
		AdditionalChargeID: f.detention.ID,
		Allocations: []*shipment.ChargeAllocation{{
			BillToCustomerID: carrier,
			Method:           shipment.ChargeAllocationMethodAmount,
			Amount:           decimal.NewNullDecimal(decimal.RequireFromString("1154.38")),
		}},
	}, testActor(f.tenantInfo))

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, carrier, created.BillToCustomerID)
	assert.Equal(t, billingqueue.StatusReadyForReview, created.Status)
	assert.True(t, created.AllocatedTotalAmount.Equal(decimal.RequireFromString("1154.38")))
	assert.Equal(t, []pulid.ID{created.ID}, result.CreatedItemIDs)
	assert.Empty(t, result.CanceledItemIDs, "Acme still pays freight")
}

func TestReassignCharge_RefusesWhileASiblingIsApproved(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	approved := *f.peakItem
	approved.Status = billingqueue.StatusApproved
	h.expectItem(f.acmeItem)
	h.expectActive(f.acmeItem, &approved)

	_, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:             f.acmeItem.ID,
		TenantInfo:         f.tenantInfo,
		ChargeKind:         shipment.ChargeAllocationKindAccessorial,
		AdditionalChargeID: f.detention.ID,
		Allocations: []*shipment.ChargeAllocation{{
			BillToCustomerID: f.peak,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
		}},
	}, testActor(f.tenantInfo))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "INV-23 is Approved")
}

func TestReassignCharge_RefusesASplitThatDoesNotAddUp(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	h.expectItem(f.acmeItem)

	_, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:     f.acmeItem.ID,
		TenantInfo: f.tenantInfo,
		ChargeKind: shipment.ChargeAllocationKindFreight,
		Allocations: []*shipment.ChargeAllocation{
			{
				BillToCustomerID: f.peak,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(decimal.NewFromInt(1000)),
			},
			{
				BillToCustomerID: f.acme,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(decimal.NewFromInt(1000)),
			},
		},
	}, testActor(f.tenantInfo))

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr), "got %v", err)
	assert.Contains(t, multiErr.Error(), "must add up to 2850.00")
}

func TestReassignCharge_RefusesAnItemPastReview(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	posted := *f.acmeItem
	posted.Status = billingqueue.StatusApproved
	h.expectItem(&posted)

	_, err := h.svc.ReassignCharge(t.Context(), &billingqueueports.ReassignChargeRequest{
		ItemID:     posted.ID,
		TenantInfo: f.tenantInfo,
		ChargeKind: shipment.ChargeAllocationKindFreight,
	}, testActor(f.tenantInfo))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only while the item is ready for review")
}

// splitDetentionByAmount divides detention by amount, Peak $154.38 and Acme
// $1,000, so an edit to the detention amount leaves that split stale.
func splitDetentionByAmount(f *splitShipment) {
	chargeID := f.detention.ID
	shipmentID := f.shipment.ID
	f.shipment.ChargeAllocations = append(f.shipment.ChargeAllocations,
		&shipment.ChargeAllocation{
			ID:                 pulid.MustNew("chal_"),
			OrganizationID:     f.tenantInfo.OrgID,
			BusinessUnitID:     f.tenantInfo.BuID,
			ShipmentID:         &shipmentID,
			AdditionalChargeID: &chargeID,
			ChargeKind:         shipment.ChargeAllocationKindAccessorial,
			BillToCustomerID:   f.peak,
			Method:             shipment.ChargeAllocationMethodAmount,
			Amount:             decimal.NewNullDecimal(decimal.RequireFromString("154.38")),
		},
		&shipment.ChargeAllocation{
			ID:                 pulid.MustNew("chal_"),
			OrganizationID:     f.tenantInfo.OrgID,
			BusinessUnitID:     f.tenantInfo.BuID,
			ShipmentID:         &shipmentID,
			AdditionalChargeID: &chargeID,
			ChargeKind:         shipment.ChargeAllocationKindAccessorial,
			BillToCustomerID:   f.acme,
			Method:             shipment.ChargeAllocationMethodAmount,
			Amount:             decimal.NewNullDecimal(decimal.NewFromInt(1000)),
			Sequence:           1,
		},
	)
}

func editedDetention(f *splitShipment, amount string) []*shipment.AdditionalCharge {
	edited := *f.detention
	edited.Amount = decimal.RequireFromString(amount)
	return []*shipment.AdditionalCharge{&edited}
}

func TestUpdateCharges_RefusesAnEditThatLeavesAnAmountSplitStale(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	splitDetentionByAmount(f)
	h.expectItem(f.acmeItem)

	_, err := h.svc.UpdateCharges(t.Context(), &billingqueueports.UpdateChargesRequest{
		ItemID:            f.acmeItem.ID,
		TenantInfo:        f.tenantInfo,
		AdditionalCharges: editedDetention(f, "1200"),
	}, testActor(f.tenantInfo))

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr), "got %v", err)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, convertAmountSplitsField, multiErr.Errors[0].Field)
	assert.Contains(t, multiErr.Error(), "Detention Fee split no longer adds up")
	assert.Contains(t, multiErr.Error(), "total 1154.38 but the charge is now 1200.00")
}

func TestUpdateCharges_ConvertsTheStaleSplitAndSavesWhenAsked(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	splitDetentionByAmount(f)
	h.expectItem(f.acmeItem)

	var saved *shipment.Shipment
	h.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
			saved = shp
			return shp, nil
		}).
		Once()

	_, err := h.svc.UpdateCharges(t.Context(), &billingqueueports.UpdateChargesRequest{
		ItemID:                       f.acmeItem.ID,
		TenantInfo:                   f.tenantInfo,
		AdditionalCharges:            editedDetention(f, "1200"),
		ConvertAmountSplitsToPercent: true,
	}, testActor(f.tenantInfo))

	require.NoError(t, err)
	require.NotNil(t, saved)
	var percents []string
	for _, row := range saved.ChargeAllocations {
		if row.ChargeKind != shipment.ChargeAllocationKindAccessorial {
			assert.Equal(
				t,
				shipment.ChargeAllocationMethodAmount,
				row.Method,
				"the balanced freight split is left alone",
			)
			continue
		}
		assert.Equal(t, shipment.ChargeAllocationMethodPercent, row.Method)
		percents = append(percents, row.Percent.Decimal.StringFixed(6))
	}
	assert.Equal(t, []string{"13.373413", "86.626587"}, percents)
	assert.True(t, saved.TotalChargeAmount.Decimal.Equal(decimal.NewFromInt(4050)))
}

func TestUpdateCharges_ConvertsAStaleAmountSplitWhenAsked(t *testing.T) {
	t.Parallel()

	f := newSplitShipment()
	stale := cloneShipment(f.shipment)
	stale.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(3000))

	rebalance, err := shipment.RebalanceAmountSplits(t.Context(), stale, stale.ChargeAllocations)
	require.NoError(t, err)
	refused := staleAmountSplitError(rebalance.Stale, accessorialNames(stale.AdditionalCharges))
	require.Len(t, refused.Errors, 1)
	assert.Equal(t, convertAmountSplitsField, refused.Errors[0].Field)
	assert.Contains(t, refused.Error(), "total 2850.00 but the charge is now 3000.00")

	assert.Equal(t, 1, rebalance.ConvertToPercent())
	after, err := shipment.RebalanceAmountSplits(t.Context(), stale, stale.ChargeAllocations)
	require.NoError(t, err)
	assert.Empty(t, after.Stale)
	assert.Equal(
		t,
		"Charges updated from billing queue; 1 amount split(s) converted to percentages",
		chargesUpdatedComment(1),
	)
}

func TestGetByID_AttachesTheItemsOwnBill(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	h.expectItem(f.peakItem)

	item, err := h.svc.GetByID(t.Context(), &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                f.peakItem.ID,
		TenantInfo:            f.tenantInfo,
		ExpandShipmentDetails: true,
	})

	require.NoError(t, err)
	require.NotNil(t, item.PayerShare)
	assert.True(t, item.PayerShare.TotalAmount.Equal(decimal.NewFromInt(1500)))
	require.Len(t, item.PayerShare.OtherPayerLines, 1)
	assert.Equal(t, "Detention Fee", item.PayerShare.OtherPayerLines[0].Description)
}
