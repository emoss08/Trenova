//go:build integration

package telematicsrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/telematicsrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stopReviewFixture struct {
	ctx        context.Context
	db         *keel.DB
	repo       telematicsports.TelematicsRepository
	tenantInfo pagination.TenantInfo
	shipmentID pulid.ID
	locationID pulid.ID
}

func setupStopReview(t *testing.T) *stopReviewFixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	type seededStop struct {
		OrganizationID pulid.ID
		BusinessUnitID pulid.ID
		ShipmentID     pulid.ID
		LocationID     pulid.ID
	}
	st := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	seeded, err := keel.IntoOne[seededStop](
		dbscope.WithSystem(ctx, "test"),
		db,
		st.AllTenants().
			Join(sm, keel.And(
				sm.ID.EqCol(st.ShipmentMoveID),
				sm.OrganizationID.EqCol(st.OrganizationID),
				sm.BusinessUnitID.EqCol(st.BusinessUnitID),
			)).
			Pick(st.OrganizationID, st.BusinessUnitID, sm.ShipmentID, st.LocationID).
			Order(st.ID.Asc()).
			Limit(1),
	)
	require.NoError(t, err)
	tenantInfo := pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID}

	return &stopReviewFixture{
		ctx: dbscope.WithTenant(ctx, tenantInfo.DBTenant()),
		db:  db,
		repo: telematicsrepository.New(
			telematicsrepository.Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()},
		),
		tenantInfo: tenantInfo,
		shipmentID: seeded.ShipmentID,
		locationID: seeded.LocationID,
	}
}

func (f *stopReviewFixture) insertMove(
	t *testing.T,
	status shipmentvocab.MoveStatus,
	arrivals ...bool,
) (*shipment.ShipmentMove, []*shipment.Stop) {
	t.Helper()

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: f.tenantInfo.BuID,
		OrganizationID: f.tenantInfo.OrgID,
		ShipmentID:     f.shipmentID,
		Status:         status,
		Loaded:         true,
		Sequence:       90,
	}
	require.NoError(t, shipmentdb.ShipmentMoves.Create(f.ctx, f.db, move))

	stops := make([]*shipment.Stop, 0, len(arrivals))
	for i, arrived := range arrivals {
		stop := &shipment.Stop{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       f.tenantInfo.BuID,
			OrganizationID:       f.tenantInfo.OrgID,
			ShipmentMoveID:       move.ID,
			LocationID:           f.locationID,
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopTypePickup,
			Sequence:             int64(i),
			ScheduledWindowStart: 1_700_000_000,
		}
		if arrived {
			at := int64(1_700_000_100)
			stop.ActualArrival = &at
			stop.Status = shipmentvocab.StopStatusInTransit
		}
		stops = append(stops, stop)
	}
	if len(stops) > 0 {
		require.NoError(t, shipmentdb.Stops.CreateMany(f.ctx, f.db, stops))
	}

	return move, stops
}

func (f *stopReviewFixture) insertOutcome(
	t *testing.T,
	occurredAt int64,
	result telematics.StopVisitResult,
) *telematics.TelematicsEvent {
	t.Helper()

	event := &telematics.TelematicsEvent{
		ID:             telematics.NewEventID(),
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Provider:       "Samsara",
		EventID:        pulid.MustNew("evt_").String(),
		EventType:      telematics.EventTypeGeofenceEntry,
		OccurredAt:     occurredAt,
		LocationID:     f.locationID,
		AddressName:    "Acme DC",
		CreatedAt:      occurredAt,
	}
	inserted, err := f.repo.InsertEvent(f.ctx, event)
	require.NoError(t, err)
	require.True(t, inserted)

	event.RecordStopOutcome(&result)
	require.NoError(t, f.repo.RecordStopOutcome(f.ctx, event))

	return event
}

func TestListOpenStopReviews_Integration(t *testing.T) {
	f := setupStopReview(t)

	move, stops := f.insertMove(t, shipmentvocab.MoveStatusInTransit, false, true)
	unarrived, arrived := stops[0], stops[1]
	finished, _ := f.insertMove(t, shipmentvocab.MoveStatusCompleted)

	refused := func(stopID pulid.ID) telematics.StopVisitResult {
		return telematics.StopVisitResult{
			Outcome: telematics.StopOutcomeRefused,
			Visit:   shipmentvocab.VisitArrival,
			MoveID:  move.ID,
			StopID:  stopID,
			Reason:  "Complete the earlier stops on this load first",
		}
	}
	unmatched := func(moveID pulid.ID) telematics.StopVisitResult {
		return telematics.StopVisitResult{
			Outcome: telematics.StopOutcomeUnmatched,
			Visit:   shipmentvocab.VisitDeparture,
			MoveID:  moveID,
			Reason:  "Not a stop on the load",
		}
	}

	openRefusal := f.insertOutcome(t, 2_000_000_300, refused(unarrived.ID))
	openUnmatched := f.insertOutcome(t, 2_000_000_200, unmatched(move.ID))
	f.insertOutcome(t, 2_000_000_100, refused(arrived.ID))
	f.insertOutcome(t, 2_000_000_100, unmatched(finished.ID))
	f.insertOutcome(t, 1_000, refused(unarrived.ID))
	f.insertOutcome(t, 2_000_000_100, telematics.StopVisitResult{
		Outcome: telematics.StopOutcomeRecorded,
		Visit:   shipmentvocab.VisitArrival,
		MoveID:  move.ID,
		StopID:  unarrived.ID,
	})

	reviews, err := f.repo.ListOpenStopReviews(f.ctx, &telematicsports.ListOpenStopReviewsRequest{
		TenantInfo: f.tenantInfo,
		Since:      2_000_000_000,
		Limit:      50,
	})
	require.NoError(t, err)

	require.Len(t, reviews, 2)
	assert.Equal(t, openRefusal.ID, reviews[0].Event.ID)
	assert.Equal(t, openUnmatched.ID, reviews[1].Event.ID)
	assert.Equal(t, f.shipmentID, reviews[0].ShipmentID)
	assert.Equal(t, telematics.StopOutcomeRefused, reviews[0].Event.StopOutcome)
	assert.Equal(t, shipmentvocab.VisitArrival, reviews[0].Event.StopVisit)
	assert.Equal(t, unarrived.ID, reviews[0].Event.StopID)
	assert.Equal(
		t,
		"Complete the earlier stops on this load first",
		reviews[0].Event.StopOutcomeReason,
	)
}

func TestStopOutcomeCheck_RefusesAReviewWithoutAReason_Integration(t *testing.T) {
	f := setupStopReview(t)

	event := &telematics.TelematicsEvent{
		ID:             telematics.NewEventID(),
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Provider:       "Samsara",
		EventID:        pulid.MustNew("evt_").String(),
		EventType:      telematics.EventTypeGeofenceEntry,
		OccurredAt:     2_000_000_000,
		CreatedAt:      2_000_000_000,
	}
	_, err := f.repo.InsertEvent(f.ctx, event)
	require.NoError(t, err)

	event.StopOutcome = telematics.StopOutcomeUnmatched
	event.StopVisit = shipmentvocab.VisitArrival
	event.ShipmentMoveID = pulid.MustNew("sm_")
	require.Error(t, f.repo.RecordStopOutcome(f.ctx, event))
}
