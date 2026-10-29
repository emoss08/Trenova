package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func (m *memoryTenderRepo) UpdateOfferStatus(
	_ context.Context,
	req *tenderports.UpdateOfferStatusRequest,
) (bool, error) {
	for _, entity := range m.tenders {
		offer := entity.FindOffer(req.OfferID)
		if offer == nil {
			continue
		}
		allowed := false
		for _, from := range req.FromStatus {
			allowed = allowed || offer.Status == from
		}
		if !allowed {
			return false, nil
		}
		offer.Status = req.ToStatus
		if req.ExpiresAt != nil {
			offer.ExpiresAt = req.ExpiresAt
		}
		if req.RespondedAt != nil {
			offer.RespondedAt = req.RespondedAt
		}
		if req.ResponseSource != "" {
			offer.ResponseSource = req.ResponseSource
		}
		if req.DeclineReason != "" {
			offer.DeclineReason = req.DeclineReason
		}
		if counter := req.Counter; counter != nil {
			offer.CounterRate = decimal.NewNullDecimal(counter.Rate)
			offer.CounterNote = counter.Note
			offer.CounteredAt = &counter.At
			offer.CounterMargin = counter.MarginPercent
			offer.CounterNeedsApproval = counter.NeedsApproval
		}
		if decided := req.CounterDecided; decided != nil {
			offer.CounterDecidedAt = &decided.At
			offer.CounterDecidedByID = decided.UserID
		}
		return true, nil
	}
	return false, nil
}

func (m *memoryTenderRepo) RecordLateOfferResponse(
	_ context.Context,
	_ *tenderports.RecordLateOfferResponseRequest,
) error {
	return nil
}

type stubDispatchControls struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
}

func (s *stubDispatchControls) GetByOrgID(
	_ context.Context,
	_ dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	return s.control, nil
}

type stubPermissions struct {
	permissionports.PermissionEngine

	allowApprove bool
	checked      []*permissionports.PermissionCheckRequest
}

func (s *stubPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	s.checked = append(s.checked, req)
	return &permissionports.PermissionCheckResult{
		Allowed: req.Operation != permission.OpApprove || s.allowApprove,
	}, nil
}

type signal struct {
	workflowID string
	payload    tendercontract.Signal
}

type signalingWorkflows struct {
	fakeWorkflowStarter

	signals []signal
}

func (s *signalingWorkflows) SignalWorkflow(
	_ context.Context,
	workflowID, _, _ string,
	arg any,
) error {
	payload, _ := arg.(tendercontract.Signal)
	s.signals = append(s.signals, signal{workflowID: workflowID, payload: payload})
	return nil
}

func (s *signalingWorkflows) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	wf any,
	args ...any,
) (client.WorkflowRun, error) {
	return s.fakeWorkflowStarter.StartWorkflow(ctx, options, wf, args...)
}

type counterDeps struct {
	repo        *memoryTenderRepo
	moves       *assignmentportsmock.MockAssignmentRepository
	shipments   *shipmentportsmock.MockShipmentRepository
	controls    *stubDispatchControls
	permissions *stubPermissions
	workflows   *signalingWorkflows
	events      *recordingEventService
	svc         *Service
	tenant      pagination.TenantInfo
	entity      *tender.Tender
	offer       *tender.TenderOffer
}

func sentWaterfall(tenantInfo pagination.TenantInfo) *tender.Tender {
	entity := &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentID:     pulid.MustNew("shp_"),
		ShipmentMoveID: pulid.MustNew("smv_"),
		Mode:           tender.ModeWaterfall,
		Status:         tender.StatusActive,
		BounceCount:    2,
	}
	sentAt := timeutils.NowUnix() - 60
	expiresAt := sentAt + 3600
	for idx, status := range []tender.OfferStatus{tender.OfferStatusSent, tender.OfferStatusPending} {
		offer := &tender.TenderOffer{
			ID:              pulid.MustNew("tof_"),
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			TenderID:        entity.ID,
			CarrierID:       pulid.MustNew("car_"),
			Rank:            int16(idx + 1),
			RateMethod:      shipment.CarrierRateMethodFlat,
			Rate:            decimal.NewFromInt(1600),
			OfferTTLSeconds: 3600,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "dispatch@carrier.example",
			Status:          status,
			Carrier:         &carrier.Carrier{Name: "Bravo Lines"},
		}
		if status == tender.OfferStatusSent {
			offer.SentAt = &sentAt
			offer.ExpiresAt = &expiresAt
		}
		entity.Offers = append(entity.Offers, offer)
	}
	return entity
}

func setupCounter(t *testing.T, policy func(*dispatchcontrol.DispatchControl)) *counterDeps {
	t.Helper()
	tenantInfo := createTestTenant()
	entity := sentWaterfall(tenantInfo)
	control := &dispatchcontrol.DispatchControl{
		TenderCounterOffersEnabled:      true,
		TenderCounterMarginFloorPercent: decimal.NewNullDecimal(decimal.NewFromInt(10)),
		TenderCounterDecisionMinutes:    60,
		TenderExhaustedAction:           dispatchcontrol.TenderExhaustedActionNotify,
	}
	if policy != nil {
		policy(control)
	}

	deps := &counterDeps{
		repo:        &memoryTenderRepo{tenders: map[pulid.ID]*tender.Tender{entity.ID: entity}},
		moves:       assignmentportsmock.NewMockAssignmentRepository(t),
		shipments:   shipmentportsmock.NewMockShipmentRepository(t),
		controls:    &stubDispatchControls{control: control},
		permissions: &stubPermissions{},
		workflows:   &signalingWorkflows{},
		events:      &recordingEventService{},
		tenant:      tenantInfo,
		entity:      entity,
		offer:       entity.Offers[0],
	}
	deps.svc = &Service{
		l:                zap.NewNop(),
		db:               dbtest.NopConnection{},
		repo:             deps.repo,
		assignmentRepo:   deps.moves,
		shipmentRepo:     deps.shipments,
		dispatchControls: deps.controls,
		permissions:      deps.permissions,
		workflows:        deps.workflows,
		eventService:     deps.events,
		auditService:     &recordingAuditService{},
		invalidator:      noopInvalidator{},
	}
	deps.svc.assemble()
	return deps
}

func (d *counterDeps) expectSellPrice(sell int64) {
	distance := 500.0
	d.moves.On("GetMoveByID", mock.Anything, d.tenant, d.entity.ShipmentMoveID).
		Return(&shipment.ShipmentMove{ID: d.entity.ShipmentMoveID, Distance: &distance}, nil)
	d.shipments.On("GetByID", mock.Anything, mock.Anything).
		Return(&shipment.Shipment{
			ID:                d.entity.ShipmentID,
			TotalChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(sell)),
		}, nil)
}

func (d *counterDeps) counter(t *testing.T, rate int64) *tenderports.TenderCounterResult {
	t.Helper()
	result, err := d.svc.negotiator.RecordCounter(
		t.Context(),
		d.tenant,
		&tenderports.RecordCounterRequest{
			OfferID: d.offer.ID,
			Rate:    decimal.NewFromInt(rate),
			Note:    "  Fuel is up on this lane  ",
			Source:  tender.ResponseSourceEmail,
		},
	)
	require.NoError(t, err)
	return result
}

func TestACounterWithinTheFloorWaitsForAPersonByDefault(t *testing.T) {
	deps := setupCounter(t, nil)
	deps.expectSellPrice(2000)
	before := timeutils.NowUnix()

	result := deps.counter(t, 1750)

	assert.False(t, result.Ignored)
	assert.False(t, result.AutoAccept)
	assert.GreaterOrEqual(t, result.DecisionDeadline, before+3600)
	assert.Equal(t, tender.OfferStatusCountered, deps.offer.Status)
	assert.True(t, decimal.NewFromInt(1750).Equal(deps.offer.CounterRate.Decimal))
	assert.Equal(t, "Fuel is up on this lane", deps.offer.CounterNote)
	assert.False(t, deps.offer.CounterNeedsApproval)
	assert.True(t, decimal.NewFromFloat(12.5).Equal(deps.offer.CounterMargin.Decimal))
	assert.Equal(t, result.DecisionDeadline, *deps.offer.ExpiresAt)
	assert.Contains(
		t,
		deps.repo.revoked,
		deps.offer.ID,
		"the carrier's link closes once it counters",
	)

	countered := deps.events.eventsOfType(shipmentevent.TypeTenderCountered)
	require.Len(t, countered, 1)
	assert.Equal(t, "Bravo Lines countered at 1750.00", countered[0].Summary)
	assert.Equal(t, false, countered[0].Metadata["needsApproval"])
}

func TestACounterWithinTheFloorIsTakenWhenTheOrganizationAsks(t *testing.T) {
	deps := setupCounter(t, func(control *dispatchcontrol.DispatchControl) {
		control.TenderAutoAcceptCounters = true
	})
	deps.expectSellPrice(2000)

	result := deps.counter(t, 1750)

	assert.True(t, result.AutoAccept)
}

func TestACounterBelowTheFloorNeedsApprovalEvenWithAutoAccept(t *testing.T) {
	deps := setupCounter(t, func(control *dispatchcontrol.DispatchControl) {
		control.TenderAutoAcceptCounters = true
	})
	deps.expectSellPrice(2000)

	result := deps.counter(t, 1900)

	assert.False(t, result.AutoAccept)
	assert.True(t, deps.offer.CounterNeedsApproval)
	countered := deps.events.eventsOfType(shipmentevent.TypeTenderCountered)
	require.Len(t, countered, 1)
	assert.Equal(t, true, countered[0].Metadata["needsApproval"])
	assert.NotContains(t, countered[0].Metadata, "marginPercent",
		"every reader of the shipment reads its activity, so the event states no margin")
	assert.True(t, decimal.NewFromInt(5).Equal(deps.offer.CounterMargin.Decimal),
		"the offer keeps the margin for the margins field group")
}

func TestACounterThatArrivesAfterTheOfferMovedOnIsLate(t *testing.T) {
	deps := setupCounter(t, nil)
	deps.expectSellPrice(2000)
	deps.offer.Status = tender.OfferStatusExpired

	result := deps.counter(t, 1750)

	assert.True(t, result.Ignored)
	assert.Equal(t, tender.OfferStatusExpired, deps.offer.Status)
	assert.Empty(t, deps.events.eventsOfType(shipmentevent.TypeTenderCountered))
	assert.Len(t, deps.events.eventsOfType(shipmentevent.TypeTenderLateResponse), 1)
}

func TestCountersAreRefusedWhenTurnedOff(t *testing.T) {
	deps := setupCounter(t, func(control *dispatchcontrol.DispatchControl) {
		control.TenderCounterOffersEnabled = false
	})

	_, err := deps.svc.responder.PreviewResponse(t.Context(), &tenderports.TenderResponseRequest{
		TenantInfo:  deps.tenant,
		OfferID:     deps.offer.ID,
		Action:      tender.ResponseActionCounter,
		Source:      tender.ResponseSourceEmail,
		CounterRate: decimal.NewFromInt(1750),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Counter-offers are turned off")
}

func TestACounterNeedsAPositiveRate(t *testing.T) {
	deps := setupCounter(t, nil)

	_, err := deps.svc.responder.PreviewResponse(t.Context(), &tenderports.TenderResponseRequest{
		TenantInfo:  deps.tenant,
		OfferID:     deps.offer.ID,
		Action:      tender.ResponseActionCounter,
		Source:      tender.ResponseSourceEmail,
		CounterRate: decimal.Zero,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}

func countered(deps *counterDeps, needsApproval bool) {
	deadline := timeutils.NowUnix() + 600
	deps.offer.Status = tender.OfferStatusCountered
	deps.offer.CounterRate = decimal.NewNullDecimal(decimal.NewFromInt(1900))
	deps.offer.CounterNeedsApproval = needsApproval
	deps.offer.ExpiresAt = &deadline
}

func TestApprovingACounterBelowTheFloorNeedsTheApprovePermission(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, true)

	err := deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
		TenantInfo: deps.tenant,
		OfferID:    deps.offer.ID,
		Approve:    true,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, deps.workflows.signals)
	require.Len(t, deps.permissions.checked, 1)
	assert.Equal(t, permission.OpApprove, deps.permissions.checked[0].Operation)
	assert.Equal(t, permission.ResourceTender.String(), deps.permissions.checked[0].Resource)
	assert.Equal(t, deps.tenant.UserID, deps.permissions.checked[0].UserID)

	deps.permissions.allowApprove = true
	require.NoError(
		t,
		deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
			TenantInfo: deps.tenant,
			OfferID:    deps.offer.ID,
			Approve:    true,
		}),
	)
	require.Len(t, deps.workflows.signals, 1)
	sent := deps.workflows.signals[0]
	assert.Equal(t, "carrier-tender-"+deps.entity.ID.String()+"-r2", sent.workflowID)
	assert.Equal(t, tender.ResponseActionAccept, sent.payload.Action)
	assert.Equal(t, tender.ResponseSourceManual, sent.payload.Source)
	assert.Equal(t, deps.offer.ID, sent.payload.OfferID)
}

func TestACounterWithinTheFloorNeedsNoApprovePermission(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, false)

	require.NoError(
		t,
		deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
			TenantInfo: deps.tenant,
			OfferID:    deps.offer.ID,
			Approve:    true,
		}),
	)

	assert.Empty(t, deps.permissions.checked)
	require.Len(t, deps.workflows.signals, 1)
}

func TestRefusingACounterNeedsNoApprovePermission(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, true)

	require.NoError(
		t,
		deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
			TenantInfo: deps.tenant,
			OfferID:    deps.offer.ID,
			Reason:     "Too rich for this lane",
		}),
	)

	require.Len(t, deps.workflows.signals, 1)
	assert.Equal(t, tender.ResponseActionDecline, deps.workflows.signals[0].payload.Action)
	assert.Equal(t, "Too rich for this lane", deps.workflows.signals[0].payload.DeclineReason)
}

func TestOnlyAStandingCounterCanBeDecided(t *testing.T) {
	deps := setupCounter(t, nil)

	err := deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
		TenantInfo: deps.tenant,
		OfferID:    deps.offer.ID,
		Approve:    true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer waiting on a decision")

	countered(deps, false)
	lapsed := timeutils.NowUnix() - 1
	deps.offer.ExpiresAt = &lapsed
	err = deps.svc.negotiator.DecideCounter(t.Context(), &tendernegotiation.DecideCounterRequest{
		TenantInfo: deps.tenant,
		OfferID:    deps.offer.ID,
		Approve:    true,
	})
	require.Error(t, err)
	assert.Empty(t, deps.workflows.signals)
}

func TestRefusingACounterClosesTheOfferForTheNextCarrier(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, true)

	err := deps.svc.offerLifecycle.DeclineOffer(
		t.Context(), deps.tenant, deps.offer.ID, tender.ResponseSourceManual, "Too rich",
	)

	require.NoError(t, err)
	assert.Equal(t, tender.OfferStatusCounterRejected, deps.offer.Status)
	assert.Equal(t, "Too rich", deps.offer.DeclineReason)
	require.NotNil(t, deps.offer.CounterDecidedByID)
	assert.Equal(t, deps.tenant.UserID, *deps.offer.CounterDecidedByID)
	decided := deps.events.eventsOfType(shipmentevent.TypeTenderCounterDecided)
	require.Len(t, decided, 1)
	assert.Equal(t, false, decided[0].Metadata["approved"])
	assert.Empty(t, deps.events.eventsOfType(shipmentevent.TypeTenderDeclined))
}

func TestACounterNobodyDecidedLapses(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, false)

	require.NoError(t, deps.svc.offerLifecycle.ExpireOffer(t.Context(), deps.tenant, deps.offer.ID))

	assert.Equal(t, tender.OfferStatusExpired, deps.offer.Status)
	decided := deps.events.eventsOfType(shipmentevent.TypeTenderCounterDecided)
	require.Len(t, decided, 1)
	assert.Equal(
		t,
		"Nobody decided on the counter-offer before it lapsed",
		decided[0].Metadata["reason"],
	)
	assert.Empty(t, deps.events.eventsOfType(shipmentevent.TypeTenderExpired))
}

type recordingAssigner struct {
	tenderports.CarrierMoveAssigner

	assigned []*carrierassignmentports.AssignMoveToCarrierRequest
}

func (r *recordingAssigner) AssignToMove(
	_ context.Context,
	req *carrierassignmentports.AssignMoveToCarrierRequest,
) (*shipment.CarrierAssignment, error) {
	r.assigned = append(r.assigned, req)
	return &shipment.CarrierAssignment{ID: pulid.MustNew("casn_"), CarrierID: req.CarrierID}, nil
}

func (r *recordingAssigner) ConfirmAssignment(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) error {
	return nil
}

func TestTakingACounterPaysTheCarrierItsCounterRate(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, false)
	assigner := &recordingAssigner{}
	deps.svc.assigner = assigner
	deps.svc.carrierAssignmentRepo = &stubCarrierAssignmentRepo{}
	deps.svc.assemble()

	result, err := deps.svc.offerLifecycle.AcceptOffer(
		t.Context(),
		deps.tenant,
		deps.offer.ID,
		tender.ResponseSourceManual,
	)

	require.NoError(t, err)
	assert.False(t, result.Ignored)
	assert.False(t, result.NeedsReview)
	assert.Equal(t, tender.OfferStatusAccepted, deps.offer.Status)
	require.NotNil(t, deps.offer.CounterDecidedAt)
	require.Len(t, assigner.assigned, 1)
	assert.True(t, decimal.NewFromInt(1900).Equal(assigner.assigned[0].BaseRate))
	decided := deps.events.eventsOfType(shipmentevent.TypeTenderCounterDecided)
	require.Len(t, decided, 1)
	assert.Equal(t, true, decided[0].Metadata["approved"])
	assert.Equal(t, "1900.00", decided[0].Metadata["counterRate"])
}

func TestAPersonsAcceptOnACounteredOfferTakesTheCounter(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, false)

	require.NoError(
		t,
		deps.svc.responder.RecordResponse(t.Context(), &tenderports.TenderResponseRequest{
			TenantInfo: deps.tenant,
			OfferID:    deps.offer.ID,
			Action:     tender.ResponseActionAccept,
			Source:     tender.ResponseSourceManual,
		}),
	)

	require.Len(t, deps.workflows.signals, 1)
	assert.Equal(t, tender.ResponseActionAccept, deps.workflows.signals[0].payload.Action)
	assert.Empty(t, deps.events.eventsOfType(shipmentevent.TypeTenderLateResponse),
		"a decision on a counter is not a late answer")
}

func TestARecordedRefusalOfACounterBelowTheFloorNeedsNoApproval(t *testing.T) {
	deps := setupCounter(t, nil)
	countered(deps, true)

	preview, err := deps.svc.responder.PreviewResponse(
		t.Context(),
		&tenderports.TenderResponseRequest{
			TenantInfo:    deps.tenant,
			OfferID:       deps.offer.ID,
			Action:        tender.ResponseActionDecline,
			Source:        tender.ResponseSourceManual,
			DeclineReason: "Too rich",
		},
	)
	require.NoError(t, err)
	refused := preview.TenderAfter.FindOffer(deps.offer.ID)
	require.NotNil(t, refused)
	assert.Equal(t, tender.OfferStatusCounterRejected, refused.Status)
	require.NotNil(t, preview.NextOffer)
	assert.Equal(t, deps.entity.Offers[1].ID, preview.NextOffer.ID)
	assert.Empty(t, deps.permissions.checked)
}
