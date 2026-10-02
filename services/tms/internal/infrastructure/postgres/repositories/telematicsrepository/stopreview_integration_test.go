//go:build integration

package telematicsrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/telematicsrepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type stopReviewFixture struct {
	ctx        context.Context
	db         *bun.DB
	repo       repositories.TelematicsRepository
	tenantInfo pagination.TenantInfo
	shipmentID pulid.ID
	locationID pulid.ID
}

func setupStopReview(t *testing.T) *stopReviewFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)

	var seeded struct {
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
		ShipmentID     pulid.ID `bun:"shipment_id"`
		LocationID     pulid.ID `bun:"location_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("stops AS st").
		ColumnExpr("st.organization_id, st.business_unit_id, sm.shipment_id, st.location_id").
		Join("JOIN shipment_moves AS sm ON sm.id = st.shipment_move_id").
		Limit(1).
		Scan(ctx, &seeded))

	return &stopReviewFixture{
		ctx:        ctx,
		db:         db,
		repo:       telematicsrepository.New(telematicsrepository.Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		tenantInfo: pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID},
		shipmentID: seeded.ShipmentID,
		locationID: seeded.LocationID,
	}
}

func (f *stopReviewFixture) insertMove(
	t *testing.T,
	status shipment.MoveStatus,
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
	_, err := f.db.NewInsert().Model(move).Exec(f.ctx)
	require.NoError(t, err)

	stops := make([]*shipment.Stop, 0, len(arrivals))
	for i, arrived := range arrivals {
		stop := &shipment.Stop{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       f.tenantInfo.BuID,
			OrganizationID:       f.tenantInfo.OrgID,
			ShipmentMoveID:       move.ID,
			LocationID:           f.locationID,
			Status:               shipment.StopStatusNew,
			Type:                 shipment.StopTypePickup,
			Sequence:             int64(i),
			ScheduledWindowStart: 1_700_000_000,
		}
		if arrived {
			at := int64(1_700_000_100)
			stop.ActualArrival = &at
			stop.Status = shipment.StopStatusInTransit
		}
		stops = append(stops, stop)
	}
	if len(stops) > 0 {
		_, err = f.db.NewInsert().Model(&stops).Exec(f.ctx)
		require.NoError(t, err)
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

	move, stops := f.insertMove(t, shipment.MoveStatusInTransit, false, true)
	unarrived, arrived := stops[0], stops[1]
	finished, _ := f.insertMove(t, shipment.MoveStatusCompleted)

	refused := func(stopID pulid.ID) telematics.StopVisitResult {
		return telematics.StopVisitResult{
			Outcome: telematics.StopOutcomeRefused,
			Visit:   shipment.VisitArrival,
			MoveID:  move.ID,
			StopID:  stopID,
			Reason:  "Complete the earlier stops on this load first",
		}
	}
	unmatched := func(moveID pulid.ID) telematics.StopVisitResult {
		return telematics.StopVisitResult{
			Outcome: telematics.StopOutcomeUnmatched,
			Visit:   shipment.VisitDeparture,
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
		Visit:   shipment.VisitArrival,
		MoveID:  move.ID,
		StopID:  unarrived.ID,
	})

	reviews, err := f.repo.ListOpenStopReviews(f.ctx, &repositories.ListOpenStopReviewsRequest{
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
	assert.Equal(t, shipment.VisitArrival, reviews[0].Event.StopVisit)
	assert.Equal(t, unarrived.ID, reviews[0].Event.StopID)
	assert.Equal(t, "Complete the earlier stops on this load first", reviews[0].Event.StopOutcomeReason)
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
	event.StopVisit = shipment.VisitArrival
	event.ShipmentMoveID = pulid.MustNew("sm_")
	require.Error(t, f.repo.RecordStopOutcome(f.ctx, event))
}
