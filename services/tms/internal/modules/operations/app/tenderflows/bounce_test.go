package tenderflows

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryTenderRepo struct {
	tenderports.TenderRepository

	tenders    map[pulid.ID]*tender.Tender
	live       *tender.Tender
	revoked    []pulid.ID
	workflowID string
	reopens    int
}

func (m *memoryTenderRepo) GetByID(
	_ context.Context,
	req tenderports.GetTenderByIDRequest,
) (*tender.Tender, error) {
	entity, ok := m.tenders[req.TenderID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Tender not found")
	}
	return tendereffects.CopyTender(entity, func(*tender.TenderOffer) {}), nil
}

func (m *memoryTenderRepo) GetOfferByID(
	_ context.Context,
	req tenderports.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	for _, entity := range m.tenders {
		if offer := entity.FindOffer(req.OfferID); offer != nil {
			copied := *offer
			copied.Tender = tendereffects.CopyTender(entity, func(*tender.TenderOffer) {})
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Offer not found")
}

func (m *memoryTenderRepo) GetLiveByMoveID(
	_ context.Context,
	_ tenderports.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return m.live, nil
}

func (m *memoryTenderRepo) Reopen(
	_ context.Context,
	req *tenderports.ReopenTenderRequest,
) (bool, error) {
	entity, ok := m.tenders[req.TenderID]
	if !ok || entity.AcceptedOfferID == nil || *entity.AcceptedOfferID != req.RevokedOfferID {
		return false, nil
	}
	offer := entity.FindOffer(req.RevokedOfferID)
	if offer == nil || offer.Status != tender.OfferStatusAccepted {
		return false, nil
	}
	m.reopens++
	entity.Status = tender.StatusActive
	entity.BounceCount++
	entity.LastBouncedAt = &req.At
	entity.AcceptedOfferID = nil
	entity.AcceptedAt = nil
	offer.Status = tender.OfferStatusRevoked
	offer.RevokedAt = &req.At
	offer.RevokeReason = req.Reason
	for _, reoffer := range req.Reoffers {
		reoffer.ID = pulid.MustNew("tof_")
		entity.Offers = append(entity.Offers, reoffer)
	}
	return true, nil
}

func (m *memoryTenderRepo) RevokeTokensForOffer(
	_ context.Context,
	_ pagination.TenantInfo,
	offerID pulid.ID,
	_ int64,
) error {
	m.revoked = append(m.revoked, offerID)
	return nil
}

func (m *memoryTenderRepo) SetWorkflowID(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	workflowID string,
) error {
	m.workflowID = workflowID
	return nil
}

func (m *memoryTenderRepo) UpdateTenderStatus(
	_ context.Context,
	req *tenderports.UpdateTenderStatusRequest,
) (bool, error) {
	entity := m.tenders[req.TenderID]
	entity.Status = req.ToStatus
	entity.CancellationReason = req.CancellationReason
	return true, nil
}

func (m *memoryTenderRepo) BulkUpdateOfferStatus(
	_ context.Context,
	_ *tenderports.BulkOfferStatusRequest,
) (int64, error) {
	return 0, nil
}

type cancelingAssigner struct {
	tenderports.CarrierMoveAssigner

	canceled []*carrierassignmentports.CancelCarrierAssignmentRequest
	err      error
}

func (c *cancelingAssigner) Cancel(
	_ context.Context,
	req *carrierassignmentports.CancelCarrierAssignmentRequest,
) error {
	if c.err != nil {
		return c.err
	}
	c.canceled = append(c.canceled, req)
	return nil
}

type bounceDeps struct {
	repo        *memoryTenderRepo
	assignments *stubCarrierAssignmentRepo
	assigner    *cancelingAssigner
	workflows   *fakeWorkflowStarter
	events      *recordingEventService
	audit       *recordingAuditService
	svc         *rig
	tenant      pagination.TenantInfo
	entity      *tender.Tender
}

func acceptedWaterfall(tenantInfo pagination.TenantInfo) *tender.Tender {
	entity := &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentID:     pulid.MustNew("shp_"),
		ShipmentMoveID: pulid.MustNew("smv_"),
		Mode:           tender.ModeWaterfall,
		Status:         tender.StatusAccepted,
	}
	sentAt := int64(1_800_000_000)
	statuses := []tender.OfferStatus{
		tender.OfferStatusDeclined,
		tender.OfferStatusAccepted,
		tender.OfferStatusSkipped,
		tender.OfferStatusSkipped,
	}
	names := []string{"Alpha Freight", "Bravo Lines", "Charlie Haul", "Delta Express"}
	for idx, status := range statuses {
		offer := &tender.TenderOffer{
			ID:              pulid.MustNew("tof_"),
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			TenderID:        entity.ID,
			CarrierID:       pulid.MustNew("car_"),
			Rank:            int16(idx + 1),
			Rate:            decimal.NewFromInt(int64(1500 + 100*idx)),
			OfferTTLSeconds: 3600,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "dispatch@carrier.example",
			Status:          status,
			Carrier:         &carrier.Carrier{Name: names[idx]},
		}
		if status != tender.OfferStatusSkipped {
			offer.SentAt = &sentAt
		}
		if status == tender.OfferStatusAccepted {
			id := offer.ID
			entity.AcceptedOfferID = &id
			entity.AcceptedAt = &sentAt
		}
		entity.Offers = append(entity.Offers, offer)
	}
	return entity
}

func setupBounce(t *testing.T) *bounceDeps {
	t.Helper()
	tenantInfo := createTestTenant()
	entity := acceptedWaterfall(tenantInfo)
	accepted := entity.FindOffer(*entity.AcceptedOfferID)

	deps := &bounceDeps{
		repo: &memoryTenderRepo{tenders: map[pulid.ID]*tender.Tender{entity.ID: entity}},
		assignments: &stubCarrierAssignmentRepo{active: &shipment.CarrierAssignment{
			ID:             pulid.MustNew("casn_"),
			CarrierID:      accepted.CarrierID,
			ShipmentMoveID: entity.ShipmentMoveID,
		}},
		assigner:  &cancelingAssigner{},
		workflows: &fakeWorkflowStarter{},
		events:    &recordingEventService{},
		audit:     &recordingAuditService{},
		tenant:    tenantInfo,
		entity:    entity,
	}
	deps.svc = &rig{
		l:                     zap.NewNop(),
		db:                    dbtest.NopConnection{},
		repo:                  deps.repo,
		carrierAssignmentRepo: deps.assignments,
		assigner:              deps.assigner,
		workflows:             deps.workflows,
		eventService:          deps.events,
		auditService:          deps.audit,
		invalidator:           noopInvalidator{},
	}
	deps.svc.assemble()
	return deps
}

type noopInvalidator struct{}

func (noopInvalidator) InvalidateShipments(
	context.Context,
	*shipmentinvalidationports.ShipmentInvalidation,
) {
}

func TestBounceTakesTheCarrierOffAndSendsTheNextRanks(t *testing.T) {
	deps := setupBounce(t)
	revokedID := *deps.entity.AcceptedOfferID

	got, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down in Joliet",
	})

	require.NoError(t, err)
	require.Len(t, deps.assigner.canceled, 1)
	assert.Equal(t, deps.entity.ShipmentMoveID, deps.assigner.canceled[0].ShipmentMoveID)
	assert.Equal(t,
		"Carrier backed out of the accepted tender: Truck broke down in Joliet",
		deps.assigner.canceled[0].Reason,
	)

	assert.Equal(t, tender.StatusActive, got.Status)
	assert.Equal(t, int16(1), got.BounceCount)
	assert.Nil(t, got.AcceptedOfferID)
	revoked := got.FindOffer(revokedID)
	require.NotNil(t, revoked)
	assert.Equal(t, tender.OfferStatusRevoked, revoked.Status)
	assert.Equal(t, "Truck broke down in Joliet", revoked.RevokeReason)

	pending := make([]*tender.TenderOffer, 0)
	for _, offer := range got.Offers {
		if offer.Status == tender.OfferStatusPending {
			pending = append(pending, offer)
		}
	}
	require.Len(t, pending, 2)
	assert.Equal(t, int16(3), pending[0].Rank)
	assert.Equal(t, int16(4), pending[1].Rank)

	require.Len(t, deps.workflows.started, 1)
	assert.Equal(t, "carrier-tender-"+deps.entity.ID.String()+"-r1", deps.workflows.started[0].ID)
	assert.Equal(t, deps.workflows.started[0].ID, deps.repo.workflowID)
	assert.Contains(t, deps.repo.revoked, revokedID)

	bounced := deps.events.eventsOfType(shipmentevent.TypeTenderBounced)
	require.Len(t, bounced, 1)
	assert.Equal(t, "Bravo Lines backed out of the accepted tender", bounced[0].Summary)
	assert.Equal(t, 2, bounced[0].Metadata["reoffers"])
	assert.Equal(t, string(tender.ResponseSourceManual), bounced[0].Metadata["source"])
	require.Len(t, deps.audit.entries, 1)
	assert.Contains(t, deps.audit.entries[0].params.ResourceID, deps.entity.ID.String())
}

func TestBounceRetryAfterTheCarrierCameOffCarriesOn(t *testing.T) {
	deps := setupBounce(t)
	deps.assignments.active = nil

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Driver no-show",
	})

	require.NoError(t, err)
	assert.Empty(t, deps.assigner.canceled)
	assert.Equal(t, 1, deps.repo.reopens)
	require.Len(t, deps.workflows.started, 1)
}

func TestBounceRefusesATenderNobodyAccepted(t *testing.T) {
	deps := setupBounce(t)
	deps.entity.Status = tender.StatusActive

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Changed their mind",
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, deps.assigner.canceled)
	assert.Zero(t, deps.repo.reopens)
	assert.Empty(t, deps.workflows.started)
}

func TestBounceRefusesAMoveSomebodyElseCovers(t *testing.T) {
	deps := setupBounce(t)
	deps.assignments.active.CarrierID = pulid.MustNew("car_")

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "covered by a different carrier")
	assert.Empty(t, deps.assigner.canceled)
	assert.Zero(t, deps.repo.reopens)
}

func TestBounceRefusesWhileAnotherTenderIsLiveOnTheMove(t *testing.T) {
	deps := setupBounce(t)
	deps.repo.live = &tender.Tender{ID: pulid.MustNew("tnd_"), Status: tender.StatusActive}

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "another tender in progress")
	assert.Empty(t, deps.assigner.canceled)
}

func TestBounceStopsWhenTheCarrierCannotComeOff(t *testing.T) {
	deps := setupBounce(t)
	deps.assigner.err = errortypes.NewBusinessError(
		"Only fresh assigned shipment moves can have their carrier assignment canceled",
	)

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})

	require.Error(t, err)
	assert.Zero(t, deps.repo.reopens, "a move under way keeps its acceptance")
	assert.Empty(t, deps.workflows.started)
}

func TestBounceRequiresAReason(t *testing.T) {
	deps := setupBounce(t)

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "   ",
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Zero(t, deps.repo.reopens)
}

func TestBounceWithNobodyLeftStillStartsTheRoundSoItExhausts(t *testing.T) {
	deps := setupBounce(t)
	for _, offer := range deps.entity.Offers {
		if offer.Status == tender.OfferStatusSkipped {
			offer.DeclineReason = "Carrier became ineligible before the offer was sent: insurance lapsed"
		}
	}

	preview, err := deps.svc.bouncer.PreviewBounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})
	require.NoError(t, err)
	assert.True(t, preview.WillExhaust)
	assert.True(t, preview.Unassigns)
	assert.Empty(t, preview.Reoffers)
	assert.Equal(t, tender.StatusAccepted, deps.entity.Status, "a preview writes nothing")

	_, err = deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})
	require.NoError(t, err)
	require.Len(t, deps.workflows.started, 1)
	bounced := deps.events.eventsOfType(shipmentevent.TypeTenderBounced)
	require.Len(t, bounced, 1)
	assert.Equal(t, 0, bounced[0].Metadata["reoffers"])
}

func TestBounceThatCannotStartItsRoundCancelsTheTender(t *testing.T) {
	deps := setupBounce(t)
	deps.workflows.err = errors.New("temporal unavailable")

	_, err := deps.svc.bouncer.Bounce(t.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: deps.tenant,
		TenderID:   deps.entity.ID,
		Reason:     "Truck broke down",
	})

	require.Error(t, err)
	assert.Equal(t, tender.StatusCanceled, deps.entity.Status)
	assert.Equal(t, "Tender workflow could not be started", deps.entity.CancellationReason)
}

func TestAnEDIDeclineOfAnAcceptedOfferBouncesTheTender(t *testing.T) {
	deps := setupBounce(t)
	accepted := *deps.entity.AcceptedOfferID

	err := deps.svc.responder.RecordResponse(t.Context(), &tenderports.TenderResponseRequest{
		TenantInfo:    deps.tenant,
		OfferID:       accepted,
		Action:        tender.ResponseActionDecline,
		Source:        tender.ResponseSourceEDI,
		DeclineReason: "Equipment unavailable",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, deps.repo.reopens)
	bounced := deps.events.eventsOfType(shipmentevent.TypeTenderBounced)
	require.Len(t, bounced, 1)
	assert.Equal(t, "Equipment unavailable", bounced[0].Metadata["reason"])
	assert.Equal(t, string(tender.ResponseSourceEDI), bounced[0].Metadata["source"])
}

func TestOnlyAnEDIDeclineOfTheAcceptedOfferIsABounce(t *testing.T) {
	deps := setupBounce(t)
	accepted := *deps.entity.AcceptedOfferID
	declined := deps.entity.Offers[0].ID

	bounced, err := deps.svc.bouncer.BounceOnCarrierDecline(
		t.Context(),
		&tenderports.TenderResponseRequest{
			TenantInfo: deps.tenant,
			OfferID:    accepted,
			Action:     tender.ResponseActionDecline,
			Source:     tender.ResponseSourceManual,
		},
	)
	require.NoError(t, err)
	assert.False(t, bounced, "a person bounces a tender explicitly")

	bounced, err = deps.svc.bouncer.BounceOnCarrierDecline(
		t.Context(),
		&tenderports.TenderResponseRequest{
			TenantInfo: deps.tenant,
			OfferID:    declined,
			Action:     tender.ResponseActionDecline,
			Source:     tender.ResponseSourceEDI,
		},
	)
	require.NoError(t, err)
	assert.False(t, bounced)
	assert.Zero(t, deps.repo.reopens)
}
