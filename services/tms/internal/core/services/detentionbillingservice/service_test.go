package detentionbillingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/ports"
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

type fakeOccurrences struct {
	repositories.DetentionOccurrenceRepository

	holds       []*detention.DetentionOccurrence
	holdsReq    *repositories.ListDetentionBillingHoldsRequest
	byCharge    []*detention.DetentionOccurrence
	byChargeReq *repositories.ListOccurrencesByChargesRequest
	updated     []*detention.DetentionOccurrence
}

func (f *fakeOccurrences) ListBillingHoldsByShipments(
	_ context.Context,
	req *repositories.ListDetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	f.holdsReq = req
	return f.holds, nil
}

func (f *fakeOccurrences) ListByAdditionalCharges(
	_ context.Context,
	req *repositories.ListOccurrencesByChargesRequest,
) ([]*detention.DetentionOccurrence, error) {
	f.byChargeReq = req
	return f.byCharge, nil
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
	repositories.DetentionEvidenceRepository

	appended []*detention.DetentionEvidence
}

func (f *fakeEvidence) Append(
	_ context.Context,
	req *repositories.AppendEvidenceRequest,
) (*detention.DetentionEvidence, error) {
	f.appended = append(f.appended, req.Entry)
	return req.Entry, nil
}

type recordingAudit struct {
	mocks.NoopAuditService

	comments []string
}

func (a *recordingAudit) LogAction(
	params *services.LogActionParams,
	_ ...services.LogOption,
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
	invoices    *mocks.MockInvoiceRepository
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
		invoices:    mocks.NewMockInvoiceRepository(t),
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
	f.invoices.EXPECT().
		ListLineCharges(mock.Anything, &repositories.ListInvoiceLineChargesRequest{
			TenantInfo: f.tenant,
			InvoiceIDs: []pulid.ID{f.invoiceID},
		}).
		Return([]repositories.InvoiceLineCharge{
			{AdditionalChargeID: f.chargeID, ShipmentID: f.shipmentID},
		}, nil).
		Once()
	f.invoices.EXPECT().
		NetBilledByCharge(mock.Anything, &repositories.NetBilledByChargeRequest{
			TenantInfo: f.tenant,
			ChargeIDs:  []pulid.ID{f.chargeID},
		}).
		Return(map[pulid.ID]decimal.Decimal{f.chargeID: net}, nil).
		Once()
}

func (f *fixture) sync(
	t *testing.T,
	event services.DetentionBillingEvent,
) (*ports.AfterCommitHooks, context.Context) {
	t.Helper()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	require.NoError(t, f.svc.SyncInvoiceBilling(ctx, &services.SyncDetentionInvoiceBillingRequest{
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

	hooks, ctx := f.sync(t, services.DetentionBillingInvoiceCreated)

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

	f.sync(t, services.DetentionBillingInvoiceVoided)

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

	f.sync(t, services.DetentionBillingInvoiceVoided)

	assert.Empty(t, f.occurrences.updated)
	assert.Empty(t, f.evidence.appended)
	assert.Equal(t, detention.OccurrenceStatusBilled, billed.Status)
}

func TestSyncInvoiceBillingDoesNothingForAnInvoiceWithoutDetention(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.invoices.EXPECT().
		ListLineCharges(mock.Anything, mock.Anything).
		Return([]repositories.InvoiceLineCharge{}, nil).
		Once()

	f.sync(t, services.DetentionBillingInvoiceCreated)

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

	err := f.svc.GuardShipments(t.Context(), &services.DetentionBillingHoldsRequest{
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

	require.NoError(t, f.svc.GuardShipments(t.Context(), &services.DetentionBillingHoldsRequest{
		TenantInfo:  f.tenant,
		ShipmentIDs: []pulid.ID{f.shipmentID},
	}))
}
