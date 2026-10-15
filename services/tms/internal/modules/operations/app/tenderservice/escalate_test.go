package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func exhaustedWaterfall(
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
) *tender.Tender {
	entity := &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		Mode:           tender.ModeWaterfall,
		Status:         tender.StatusExhausted,
	}
	for idx, status := range []tender.OfferStatus{
		tender.OfferStatusDeclined,
		tender.OfferStatusRevoked,
		tender.OfferStatusExpired,
	} {
		entity.Offers = append(entity.Offers, &tender.TenderOffer{
			ID:              pulid.MustNew("tof_"),
			TenderID:        entity.ID,
			CarrierID:       pulid.MustNew("car_"),
			Rank:            int16(idx + 1),
			RateMethod:      shipment.CarrierRateMethodFlat,
			Rate:            decimal.NewFromInt(int64(1500 + 100*idx)),
			OfferTTLSeconds: 7200,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "dispatch@carrier.example",
			Status:          status,
		})
	}
	return entity
}

func setupEscalation(
	t *testing.T,
	policy func(*dispatchcontrol.DispatchControl),
	sell int64,
) (*createTestDeps, pagination.TenantInfo, *tender.Tender) {
	t.Helper()
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	control := &dispatchcontrol.DispatchControl{
		TenderCounterOffersEnabled:          true,
		TenderCounterMarginFloorPercent:     decimal.NewFromInt(10),
		TenderExhaustedAction:               dispatchcontrol.TenderExhaustedActionRebroadcast,
		TenderEscalationRateIncreasePercent: decimal.NewFromInt(10),
		TenderEscalationOfferTTLMinutes:     45,
		TenderMaxEscalations:                2,
	}
	if policy != nil {
		policy(control)
	}
	deps.svc.dispatchControls = &stubDispatchControls{control: control}
	allowNotifications(deps)

	deps.assignmentRepo.On("GetMoveByID", mock.Anything, tenantInfo, move.ID).
		Return(move, nil).Maybe()
	deps.holdRepo.On("HasActiveDispatchHold", mock.Anything, mock.Anything).
		Return(false, nil).Maybe()
	deps.assignmentRepo.On("GetByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil).Maybe()
	deps.carrierAssignmentRepo.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil).Maybe()
	deps.shipmentRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&shipment.Shipment{
			ID:                move.ShipmentID,
			TotalChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(sell)),
		}, nil).Maybe()

	return deps, tenantInfo, exhaustedWaterfall(tenantInfo, move)
}

func TestAnExhaustedTenderIsRebroadcastAtTheEscalationRate(t *testing.T) {
	deps, tenantInfo, exhausted := setupEscalation(t, nil, 2000)

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhausted)

	created := deps.tenderRepo.created
	require.NotNil(t, created)
	assert.Equal(t, tender.ModeSpotBroadcast, created.Mode)
	assert.Equal(t, int16(1), created.EscalationRound)
	require.NotNil(t, created.EscalatedFromID)
	assert.Equal(t, exhausted.ID, *created.EscalatedFromID)
	assert.Nil(t, created.RoutingGuideID)

	require.Len(t, created.Offers, 2, "the carrier that bounced is not asked again")
	assert.Equal(t, exhausted.Offers[0].CarrierID, created.Offers[0].CarrierID)
	assert.Equal(t, "1650", created.Offers[0].Rate.String())
	assert.Equal(t, exhausted.Offers[2].CarrierID, created.Offers[1].CarrierID)
	assert.Equal(t, "1800", created.Offers[1].Rate.String(), "capped at the margin floor")
	for _, offer := range created.Offers {
		assert.Equal(t, int32(45*60), offer.OfferTTLSeconds)
		assert.Equal(t, tender.OfferStatusPending, offer.Status)
	}

	require.Len(t, deps.workflows.started, 1)
	assert.Equal(t, "carrier-tender-"+created.ID.String(), deps.workflows.started[0].ID)
	escalated := deps.events.eventsOfType(shipmentevent.TypeTenderEscalated)
	require.Len(t, escalated, 1)
	assert.Equal(t, 2, escalated[0].Metadata["carriers"])
	assert.Equal(t, "10", escalated[0].Metadata["rateIncreasePercent"])
}

func TestRewaterfallWalksTheCarriersInRankOrder(t *testing.T) {
	deps, tenantInfo, exhausted := setupEscalation(t, func(control *dispatchcontrol.DispatchControl) {
		control.TenderExhaustedAction = dispatchcontrol.TenderExhaustedActionRewaterfall
	}, 2000)

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhausted)

	created := deps.tenderRepo.created
	require.NotNil(t, created)
	assert.Equal(t, tender.ModeSpotSequential, created.Mode)
	assert.Equal(t, int16(1), created.Offers[0].Rank)
	assert.Equal(t, int16(2), created.Offers[1].Rank)
}

func TestNotifyOnlyNeverEscalates(t *testing.T) {
	deps, tenantInfo, exhausted := setupEscalation(t, func(control *dispatchcontrol.DispatchControl) {
		control.TenderExhaustedAction = dispatchcontrol.TenderExhaustedActionNotify
	}, 2000)

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhausted)

	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
}

func TestEscalationStopsAtTheRoundLimit(t *testing.T) {
	deps, tenantInfo, exhausted := setupEscalation(t, nil, 2000)
	exhausted.EscalationRound = 2
	from := pulid.MustNew("tnd_")
	exhausted.EscalatedFromID = &from

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhausted)

	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
}

func TestEscalationWithoutASellPriceKeepsTheRates(t *testing.T) {
	deps, tenantInfo, exhausted := setupEscalation(t, nil, 0)

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhausted)

	created := deps.tenderRepo.created
	require.NotNil(t, created)
	assert.Equal(t, "1500", created.Offers[0].Rate.String())
	assert.Equal(t, "1700", created.Offers[1].Rate.String())
}

func TestEscalationLeavesAMoveSomebodyCoveredAlone(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	deps.svc.dispatchControls = &stubDispatchControls{control: &dispatchcontrol.DispatchControl{
		TenderExhaustedAction: dispatchcontrol.TenderExhaustedActionRebroadcast,
		TenderMaxEscalations:  1,
	}}
	allowNotifications(deps)
	deps.assignmentRepo.On("GetMoveByID", mock.Anything, tenantInfo, move.ID).Return(move, nil)
	deps.holdRepo.On("HasActiveDispatchHold", mock.Anything, mock.Anything).Return(false, nil)
	deps.assignmentRepo.On("GetByMoveID", mock.Anything, tenantInfo, move.ID).Return(nil, nil)
	deps.carrierAssignmentRepo.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(&shipment.CarrierAssignment{ID: pulid.MustNew("casn_")}, nil)

	deps.svc.escalateExhausted(t.Context(), tenantInfo, exhaustedWaterfall(tenantInfo, move))

	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
}

func allowNotifications(deps *createTestDeps) {
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *notification.Notification) (*notification.Notification, error) {
			return entity, nil
		}).Maybe()
}
