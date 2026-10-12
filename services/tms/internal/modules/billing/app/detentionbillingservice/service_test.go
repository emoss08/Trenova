package detentionbillingservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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

type fakeOccurrences struct {
	detentionports.DetentionOccurrenceRepository

	holds       []*detention.DetentionOccurrence
	holdsReq    *detentionports.ListDetentionBillingHoldsRequest
	byCharge    []*detention.DetentionOccurrence
	byChargeReq *detentionports.ListOccurrencesByChargesRequest
	updated     []*detention.DetentionOccurrence
}

func (f *fakeOccurrences) ListBillingHoldsByShipments(
	_ context.Context,
	req *detentionports.ListDetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	f.holdsReq = req
	return f.holds, nil
}

func (f *fakeOccurrences) ListByAdditionalCharges(
	_ context.Context,
	req *detentionports.ListOccurrencesByChargesRequest,
) ([]*detention.DetentionOccurrence, error) {
	f.byChargeReq = req
	listed := make([]*detention.DetentionOccurrence, 0, len(f.byCharge))
	for _, occurrence := range f.byCharge {
		if slices.Contains(req.Statuses, occurrence.Status) {
			listed = append(listed, occurrence)
		}
	}
	return listed, nil
}

func (f *fakeOccurrences) Update(
	_ context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	entity.Version++
	f.updated = append(f.updated, entity)
	return entity, nil
}

type fakeEvidence struct {
	detentionports.DetentionEvidenceRepository

	appended []*detention.DetentionEvidence
}

func (f *fakeEvidence) Append(
	_ context.Context,
	req *detentionports.AppendEvidenceRequest,
) (*detention.DetentionEvidence, error) {
	f.appended = append(f.appended, req.Entry)
	return req.Entry, nil
}

type recordingAudit struct {
	mocks.NoopAuditService

	comments []string
}

func (a *recordingAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.comments = append(a.comments, params.ResourceID)
	return nil
}

type fixture struct {
	tenant      pagination.TenantInfo
	shipmentID  pulid.ID
	chargeID    pulid.ID
	invoiceID   pulid.ID
	userID      pulid.ID
	occurrences *fakeOccurrences
	evidence    *fakeEvidence
	audit       *recordingAudit
	invoices    *invoiceportsmock.MockInvoiceRepository
	svc         *Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		tenant:      pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		shipmentID:  pulid.MustNew("shp_"),
		chargeID:    pulid.MustNew("ac_"),
		invoiceID:   pulid.MustNew("inv_"),
		userID:      pulid.MustNew("usr_"),
		occurrences: &fakeOccurrences{},
		evidence:    &fakeEvidence{},
		audit:       &recordingAudit{},
		invoices:    invoiceportsmock.NewMockInvoiceRepository(t),
	}
	f.svc = &Service{
		l:              zap.NewNop(),
		occurrenceRepo: f.occurrences,
		evidenceRepo:   f.evidence,
		invoiceRepo:    f.invoices,
		auditService:   f.audit,
		now:            func() int64 { return 1_700_000_000 },
	}

	return f
}

func (f *fixture) occurrence(status detention.OccurrenceStatus) *detention.DetentionOccurrence {
	chargeID := f.chargeID

	return &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		OrganizationID:     f.tenant.OrgID,
		BusinessUnitID:     f.tenant.BuID,
		ShipmentID:         f.shipmentID,
		Status:             status,
		BillableAmount:     decimal.NewFromInt(180),
		Currency:           "USD",
		AdditionalChargeID: &chargeID,
	}
}

func (f *fixture) expectCharges(net decimal.Decimal) {
	f.expectLineCharges()
	f.invoices.EXPECT().
		NetBilledByCharge(mock.Anything, &invoiceports.NetBilledByChargeRequest{
			TenantInfo: f.tenant,
			ChargeIDs:  []pulid.ID{f.chargeID},
		}).
		Return(map[pulid.ID]decimal.Decimal{f.chargeID: net}, nil).
		Once()
}

func (f *fixture) expectLineCharges() {
	f.invoices.EXPECT().
		ListLineCharges(mock.Anything, &invoiceports.ListInvoiceLineChargesRequest{
			TenantInfo: f.tenant,
			InvoiceIDs: []pulid.ID{f.invoiceID},
		}).
		Return([]invoiceports.InvoiceLineCharge{
			{AdditionalChargeID: f.chargeID, ShipmentID: f.shipmentID},
		}, nil).
		Once()
}

func (f *fixture) sync(
	t *testing.T,
	event detentionports.DetentionBillingEvent,
) (*kernelports.AfterCommitHooks, context.Context) {
	t.Helper()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	require.NoError(t, f.svc.SyncInvoiceBilling(ctx, &detentionports.SyncDetentionInvoiceBillingRequest{
		TenantInfo:    f.tenant,
		InvoiceIDs:    []pulid.ID{f.invoiceID, f.invoiceID},
		InvoiceNumber: "INV-1001",
		Event:         event,
		ActorUserID:   f.userID,
	}))

	return hooks, ctx
}

func TestSyncInvoiceBillingMarksTheChargesOnANewInvoiceBilled(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	pending := f.occurrence(detention.OccurrenceStatusPending)
	approved := f.occurrence(detention.OccurrenceStatusApproved)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{pending, approved}
	f.expectCharges(decimal.NewFromInt(360))

	hooks, ctx := f.sync(t, detentionports.DetentionBillingInvoiceCreated)

	require.NotNil(t, f.occurrences.byChargeReq)
	assert.Equal(t, []pulid.ID{f.chargeID}, f.occurrences.byChargeReq.ChargeIDs)
	assert.Equal(t, []pulid.ID{f.shipmentID}, f.occurrences.byChargeReq.ShipmentIDs)

	require.Len(t, f.occurrences.updated, 2)
	for _, occurrence := range f.occurrences.updated {
		assert.Equal(t, detention.OccurrenceStatusBilled, occurrence.Status)
	}

	require.Len(t, f.evidence.appended, 2)
	for _, entry := range f.evidence.appended {
		assert.Equal(t, detention.EvidenceKindStatusChange, entry.Kind)
		assert.Equal(t, detention.EvidenceSourceSystem, entry.Source)
		assert.Equal(t, "Billed on invoice INV-1001 at 180.00 USD", entry.Summary)
		require.NotNil(t, entry.RecordedByID)
		assert.Equal(t, f.userID, *entry.RecordedByID)
	}

	assert.Empty(t, f.audit.comments, "the audit waits for the invoice to commit")
	hooks.Run(ctx)
	assert.Len(t, f.audit.comments, 2)
}

func TestSyncInvoiceBillingReleasesBilledChargesWhenNoInvoiceStands(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	approver := pulid.MustNew("usr_")
	billed := f.occurrence(detention.OccurrenceStatusBilled)
	billed.ApprovedByID = &approver
	f.occurrences.byCharge = []*detention.DetentionOccurrence{billed}
	f.expectCharges(decimal.Zero)

	f.sync(t, detentionports.DetentionBillingInvoiceVoided)

	require.Len(t, f.occurrences.updated, 1)
	assert.Equal(t, detention.OccurrenceStatusApproved, f.occurrences.updated[0].Status)
	require.NotNil(t, f.occurrences.updated[0].ApprovedByID)
	assert.Equal(t, approver, *f.occurrences.updated[0].ApprovedByID)
	require.Len(t, f.evidence.appended, 1)
	assert.Equal(t,
		"Billing released: invoice INV-1001 was voided and no other invoice bills this charge",
		f.evidence.appended[0].Summary,
	)
}

func TestSyncInvoiceBillingKeepsAChargeAnotherPayersInvoiceStillBills(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	billed := f.occurrence(detention.OccurrenceStatusBilled)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{billed}
	f.expectCharges(decimal.NewFromInt(90))

	f.sync(t, detentionports.DetentionBillingInvoiceVoided)

	assert.Empty(t, f.occurrences.updated)
	assert.Empty(t, f.evidence.appended)
	assert.Equal(t, detention.OccurrenceStatusBilled, billed.Status)
}

func TestSyncInvoiceBillingDoesNothingForAnInvoiceWithoutDetention(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.invoices.EXPECT().
		ListLineCharges(mock.Anything, mock.Anything).
		Return([]invoiceports.InvoiceLineCharge{}, nil).
		Once()

	f.sync(t, detentionports.DetentionBillingInvoiceCreated)

	assert.Nil(t, f.occurrences.byChargeReq)
	assert.Empty(t, f.occurrences.updated)
}

func TestGuardShipmentsRefusesAndNamesEveryHeldCharge(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.occurrence(detention.OccurrenceStatusPending)
	first.RequiresApproval = true
	second := f.occurrence(detention.OccurrenceStatusPending)
	second.RequiresApproval = true
	f.occurrences.holds = []*detention.DetentionOccurrence{first, second}

	err := f.svc.GuardShipments(t.Context(), &detentionports.DetentionBillingHoldsRequest{
		TenantInfo:  f.tenant,
		ShipmentIDs: []pulid.ID{f.shipmentID, f.shipmentID, pulid.Nil},
	})

	require.Error(t, err)
	require.NotNil(t, f.occurrences.holdsReq)
	assert.Equal(t, []pulid.ID{f.shipmentID}, f.occurrences.holdsReq.ShipmentIDs)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t,
		"2 detention charges on this shipment still need approval before the shipment can be billed. "+
			"Approve or waive them on the detention desk first.",
		business.Error(),
	)
	assert.Equal(t, first.ID.String()+","+second.ID.String(),
		business.Params["detentionOccurrenceIds"])
	assert.Equal(t, f.shipmentID.String(), business.Params["shipmentIds"])
}

func TestGuardShipmentsPassesAPendingChargeThatNeedsNoApproval(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.occurrences.holds = []*detention.DetentionOccurrence{}

	require.NoError(t, f.svc.GuardShipments(t.Context(), &detentionports.DetentionBillingHoldsRequest{
		TenantInfo:  f.tenant,
		ShipmentIDs: []pulid.ID{f.shipmentID},
	}))
}

func (f *fixture) syncWaiving(t *testing.T, waivers ...detentionports.DetentionWaiver) {
	t.Helper()

	ctx, _ := kernelports.WithAfterCommitHooks(t.Context())
	require.NoError(t, f.svc.SyncInvoiceBilling(ctx, &detentionports.SyncDetentionInvoiceBillingRequest{
		TenantInfo:    f.tenant,
		InvoiceIDs:    []pulid.ID{f.invoiceID},
		InvoiceNumber: "INV-1001",
		Event:         detentionports.DetentionBillingInvoiceAdjusted,
		ActorUserID:   f.userID,
		Waivers:       waivers,
	}))
}

func TestSyncInvoiceBillingWaivesACreditedChargeTheAdjustmentWasForWaiving(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	requester := pulid.MustNew("usr_")
	billed := f.occurrence(detention.OccurrenceStatusBilled)
	other := f.occurrence(detention.OccurrenceStatusBilled)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{billed, other}
	f.expectCharges(decimal.Zero)

	f.syncWaiving(t, detentionports.DetentionWaiver{
		OccurrenceID: billed.ID,
		Note:         "Decided in Triage",
		RequestedBy:  requester,
	})

	require.Len(t, f.occurrences.updated, 2)
	assert.Equal(t, detention.OccurrenceStatusWaived, billed.Status)
	require.NotNil(t, billed.WaiverReason)
	assert.Equal(t, detention.WaiverReasonOther, *billed.WaiverReason)
	require.NotNil(t, billed.WaivedByID)
	assert.Equal(t, requester, *billed.WaivedByID)
	assert.True(t, billed.WaivedAmount.Equal(decimal.NewFromInt(180)))
	assert.True(t, billed.BillableAmount.IsZero())
	assert.Equal(t, detention.OccurrenceStatusApproved, other.Status)

	require.Len(t, f.evidence.appended, 2)
	assert.Equal(t, detention.EvidenceKindWaiver, f.evidence.appended[0].Kind)
	assert.Equal(t,
		"Waived 180.00 USD after invoice INV-1001 was credited: Decided in Triage",
		f.evidence.appended[0].Summary,
	)
	assert.Equal(t, detention.EvidenceKindStatusChange, f.evidence.appended[1].Kind)
}

func TestSyncInvoiceBillingLeavesAWaiverAloneWhileAnInvoiceStillBillsTheCharge(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	billed := f.occurrence(detention.OccurrenceStatusBilled)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{billed}
	f.expectCharges(decimal.NewFromInt(90))

	f.syncWaiving(t, detentionports.DetentionWaiver{OccurrenceID: billed.ID})

	assert.Empty(t, f.occurrences.updated)
	assert.Equal(t, detention.OccurrenceStatusBilled, billed.Status)
}

func TestSyncInvoiceBillingWaivesADisputedChargeOnceItIsCredited(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	disputed := f.occurrence(detention.OccurrenceStatusDisputed)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{disputed}
	f.expectCharges(decimal.Zero)

	f.syncWaiving(t, detentionports.DetentionWaiver{
		OccurrenceID: disputed.ID,
		Reason:       detention.WaiverReasonCustomerGoodwill,
	})

	require.Len(t, f.occurrences.updated, 1)
	assert.Equal(t, detention.OccurrenceStatusWaived, disputed.Status)
	assert.Equal(t, detention.WaiverReasonCustomerGoodwill, *disputed.WaiverReason)
}

func TestSyncInvoiceBillingKeepsADisputedChargeAnInvoiceStillBillsDisputed(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	disputed := f.occurrence(detention.OccurrenceStatusDisputed)
	disputed.DisputeNote = "Customer says the truck arrived late"
	f.occurrences.byCharge = []*detention.DetentionOccurrence{disputed}
	f.expectCharges(decimal.NewFromInt(180))

	hooks, ctx := f.sync(t, detentionports.DetentionBillingInvoiceAdjusted)
	hooks.Run(ctx)

	assert.Equal(t, detention.OccurrenceStatusDisputed, disputed.Status)
	assert.Equal(t, "Customer says the truck arrived late", disputed.DisputeNote)
	assert.True(t, disputed.BillableAmount.Equal(decimal.NewFromInt(180)))
	assert.Empty(t, f.occurrences.updated)
	assert.Empty(t, f.evidence.appended)
	assert.Empty(t, f.audit.comments)
}

func TestSyncInvoiceBillingStandByThenAdjustmentThenWaive(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	billed := f.occurrence(detention.OccurrenceStatusBilled)
	other := f.occurrence(detention.OccurrenceStatusApproved)
	f.occurrences.byCharge = []*detention.DetentionOccurrence{billed, other}

	require.NoError(t, billed.Dispute("Standing by the gate record", 1_699_999_000))

	f.expectCharges(decimal.NewFromInt(360))
	f.sync(t, detentionports.DetentionBillingInvoiceAdjusted)

	assert.Equal(t, detention.OccurrenceStatusDisputed, billed.Status)
	assert.Equal(t, detention.OccurrenceStatusBilled, other.Status)
	require.Len(t, f.occurrences.updated, 1)
	assert.Same(t, other, f.occurrences.updated[0])

	f.expectCharges(decimal.Zero)
	f.syncWaiving(t, detentionports.DetentionWaiver{
		OccurrenceID: billed.ID,
		Note:         "Waived in Triage after standing by",
		RequestedBy:  f.userID,
	})

	assert.Equal(t, detention.OccurrenceStatusWaived, billed.Status)
	assert.True(t, billed.WaivedAmount.Equal(decimal.NewFromInt(180)))
	assert.True(t, billed.BillableAmount.IsZero())
	require.NotNil(t, billed.WaiverReason)
	assert.Equal(t, detention.WaiverReasonOther, *billed.WaiverReason)
	assert.Equal(t, detention.OccurrenceStatusApproved, other.Status)

	f.expectCharges(decimal.NewFromInt(180))
	f.sync(t, detentionports.DetentionBillingInvoiceCreated)

	assert.Equal(t, detention.OccurrenceStatusWaived, billed.Status)
	assert.Equal(t, detention.OccurrenceStatusBilled, other.Status)
}

func TestSyncInvoiceBillingIsIdempotentForEveryStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   detention.OccurrenceStatus
		billed   bool
		want     detention.OccurrenceStatus
		recorded bool
	}{
		{detention.OccurrenceStatusAccruing, true, detention.OccurrenceStatusAccruing, false},
		{detention.OccurrenceStatusAccruing, false, detention.OccurrenceStatusAccruing, false},
		{detention.OccurrenceStatusPending, true, detention.OccurrenceStatusBilled, true},
		{detention.OccurrenceStatusPending, false, detention.OccurrenceStatusPending, false},
		{detention.OccurrenceStatusApproved, true, detention.OccurrenceStatusBilled, true},
		{detention.OccurrenceStatusApproved, false, detention.OccurrenceStatusApproved, false},
		{detention.OccurrenceStatusBilled, true, detention.OccurrenceStatusBilled, false},
		{detention.OccurrenceStatusBilled, false, detention.OccurrenceStatusApproved, true},
		{detention.OccurrenceStatusWaived, true, detention.OccurrenceStatusWaived, false},
		{detention.OccurrenceStatusWaived, false, detention.OccurrenceStatusWaived, false},
		{detention.OccurrenceStatusDisputed, true, detention.OccurrenceStatusDisputed, false},
		{detention.OccurrenceStatusDisputed, false, detention.OccurrenceStatusDisputed, false},
		{detention.OccurrenceStatusNotBillable, true, detention.OccurrenceStatusNotBillable, false},
		{detention.OccurrenceStatusNotBillable, false, detention.OccurrenceStatusNotBillable, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.status)+billedName(tc.billed), func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			occurrence := f.occurrence(tc.status)
			f.occurrences.byCharge = []*detention.DetentionOccurrence{occurrence}

			net := decimal.Zero
			if tc.billed {
				net = decimal.NewFromInt(180)
			}
			listed := slices.Contains(detention.InvoiceSettledStatuses(), tc.status)
			for range 2 {
				if listed {
					f.expectCharges(net)
				} else {
					f.expectLineCharges()
				}
				f.sync(t, detentionports.DetentionBillingInvoiceAdjusted)
				assert.Equal(t, tc.want, occurrence.Status)
			}

			want := 0
			if tc.recorded {
				want = 1
			}
			assert.Len(t, f.occurrences.updated, want)
			assert.Len(t, f.evidence.appended, want)
		})
	}
}

func billedName(billed bool) string {
	if billed {
		return "/invoiced"
	}

	return "/not-invoiced"
}

func TestSyncInvoiceBillingListsEveryOccurrenceAnInvoiceCanSettle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.expectCharges(decimal.NewFromInt(180))
	f.occurrences.byCharge = []*detention.DetentionOccurrence{
		f.occurrence(detention.OccurrenceStatusPending),
	}

	f.sync(t, detentionports.DetentionBillingInvoiceCreated)

	require.NotNil(t, f.occurrences.byChargeReq)
	assert.ElementsMatch(t, []detention.OccurrenceStatus{
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusDisputed,
	}, f.occurrences.byChargeReq.Statuses)
}
