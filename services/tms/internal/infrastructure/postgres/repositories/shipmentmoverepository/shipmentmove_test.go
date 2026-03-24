package shipmentmoverepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, *bun.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, bunDB, mock
}

func TestSyncForShipment_InsertsNewMovesAndStops(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	entity.Moves[0].Loaded = false

	mock.ExpectQuery(`SELECT .*shipment_moves.*shipment_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_id", "status", "loaded", "sequence", "version"}))
	mock.ExpectQuery(`INSERT INTO .*shipment_moves.*loaded.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "distance", "updated_at"}).AddRow(0, nil, 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(0, nil, nil, nil, nil, "", 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(1, nil, nil, nil, nil, "", 0))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	assert.False(t, entity.Moves[0].ID.IsNil())
	assert.False(t, entity.Moves[0].Loaded)
	assert.False(t, entity.Moves[0].Stops[0].ID.IsNil())
	assert.False(t, entity.Moves[0].Stops[1].ID.IsNil())
}

func TestSyncForShipment_UpdatesExistingMoveAndStopsAndDeletesRemovedStop(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	moveID := pulid.MustNew("sm_")
	stopKeepID := pulid.MustNew("stp_")
	stopDeleteID := pulid.MustNew("stp_")

	entity.Moves[0].ID = moveID
	entity.Moves[0].Version = 1
	entity.Moves[0].Stops[0].ID = stopKeepID
	entity.Moves[0].Stops[0].Version = 1
	entity.Moves[0].Stops = entity.Moves[0].Stops[:1]

	mock.ExpectQuery(`SELECT .*shipment_moves.*shipment_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_id", "status", "loaded", "sequence", "version"}).
			AddRow(moveID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, entity.Moves[0].Status, entity.Moves[0].Loaded, 0, 1),
		)
	mock.ExpectQuery(`SELECT .*stops.*shipment_move_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version"}).
			AddRow(stopKeepID, entity.OrganizationID, entity.BusinessUnitID, moveID, entity.Moves[0].Stops[0].LocationID, entity.Moves[0].Stops[0].Status, entity.Moves[0].Stops[0].Type, 0, entity.Moves[0].Stops[0].ScheduledWindowStart, entity.Moves[0].Stops[0].ScheduledWindowEnd, 1).
			AddRow(stopDeleteID, entity.OrganizationID, entity.BusinessUnitID, moveID, pulid.MustNew("loc_"), shipment.StopStatusNew, shipment.StopTypeDelivery, 1, int64(2), int64(2), 1),
		)
	mock.ExpectExec(`UPDATE .*shipment_moves.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*stops.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM .*stops.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, int64(2), entity.Moves[0].Version)
	assert.Equal(t, int64(2), entity.Moves[0].Stops[0].Version)
}

func TestSyncForShipment_DeletesRemovedMoveWhenUnassigned(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	existingMoveID := pulid.MustNew("sm_")

	mock.ExpectQuery(`SELECT .*shipment_moves.*shipment_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_id", "status", "loaded", "sequence", "version"}).
			AddRow(existingMoveID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, shipment.MoveStatusNew, true, 0, 1),
		)
	mock.ExpectQuery(`SELECT .*stops.*shipment_move_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version"}))
	mock.ExpectQuery(`INSERT INTO .*shipment_moves.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "distance", "updated_at"}).AddRow(0, nil, 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(0, nil, nil, nil, nil, "", 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(1, nil, nil, nil, nil, "", 0))
	mock.ExpectQuery(`SELECT count\(\*\) FROM .*assignments.*shipment_move_id IN.*archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`DELETE FROM .*shipment_moves.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_RejectsDeletingAssignedMove(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	existingMoveID := pulid.MustNew("sm_")

	mock.ExpectQuery(`SELECT .*shipment_moves.*shipment_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_id", "status", "loaded", "sequence", "version"}).
			AddRow(existingMoveID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, shipment.MoveStatusNew, true, 0, 1),
		)
	mock.ExpectQuery(`SELECT .*stops.*shipment_move_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version"}))
	mock.ExpectQuery(`INSERT INTO .*shipment_moves.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "distance", "updated_at"}).AddRow(0, nil, 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(0, nil, nil, nil, nil, "", 0))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"sequence", "pieces", "weight", "actual_arrival", "actual_departure", "address_line", "updated_at"}).AddRow(1, nil, nil, nil, nil, "", 0))
	mock.ExpectQuery(`SELECT count\(\*\) FROM .*assignments.*shipment_move_id IN.*archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMovesByShipmentID_ExpandedDetailsKeepsUnassignedMoves(t *testing.T) {
	t.Parallel()

	repo, _, mock := newTestRepository(t)
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	assignedMoveID := pulid.MustNew("sm_")
	unassignedMoveID := pulid.MustNew("sm_")
	activeAssignmentID := pulid.MustNew("asn_")

	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.shipment_id = .*sm\.organization_id = .*sm\.business_unit_id = .*ORDER BY "sm"\."sequence" ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "version", "created_at", "updated_at",
		}).
			AddRow(assignedMoveID, buID, orgID, shipmentID, shipment.MoveStatusAssigned, true, 0, 1, 1, 1).
			AddRow(unassignedMoveID, buID, orgID, shipmentID, shipment.MoveStatusNew, true, 1, 1, 1, 1))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*shipment_move_id.*IN.*ORDER BY "stp"\."sequence" ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type", "schedule_type", "sequence", "scheduled_window_start", "scheduled_window_end", "version", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id IN.*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "primary_worker_id", "tractor_id", "trailer_id", "secondary_worker_id", "status", "version", "created_at", "updated_at",
		}).
			AddRow(activeAssignmentID, buID, orgID, assignedMoveID, nil, nil, nil, nil, shipment.AssignmentStatusNew, 1, 1, 1))

	entities, err := repo.GetMovesByShipmentID(t.Context(), &repositories.GetMovesByShipmentIDRequest{
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ExpandMoveDetails: true,
	})

	require.NoError(t, err)
	require.Len(t, entities, 2)
	require.NotNil(t, entities[0].Assignment)
	assert.Equal(t, activeAssignmentID, entities[0].Assignment.ID)
	assert.Nil(t, entities[1].Assignment)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateStatus_UpdatesMove(t *testing.T) {
	t.Parallel()

	repo, _, mock := newTestRepository(t)
	moveID := pulid.MustNew("sm_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.id = .*sm\.organization_id = .*sm\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "version", "created_at", "updated_at",
		}).AddRow(moveID, buID, orgID, pulid.MustNew("shp_"), shipment.MoveStatusAssigned, true, 0, 1, 1, 1))
	mock.ExpectExec(`UPDATE .*shipment_moves.*status.*version.*updated_at.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.id = .*sm\.organization_id = .*sm\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "version", "created_at", "updated_at",
		}).AddRow(moveID, buID, orgID, pulid.MustNew("shp_"), shipment.MoveStatusInTransit, true, 0, 2, 1, 2))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*shipment_move_id.*ORDER BY "stp"\."sequence" ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id IN.*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "primary_worker_id", "tractor_id", "trailer_id", "secondary_worker_id", "status", "version", "created_at", "updated_at",
		}))

	entity, err := repo.UpdateStatus(t.Context(), &repositories.UpdateMoveStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		MoveID: moveID,
		Status: shipment.MoveStatusInTransit,
	})

	require.NoError(t, err)
	assert.Equal(t, shipment.MoveStatusInTransit, entity.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSplitMove_CreatesDownstreamMoveAndUpdatesOriginalStop(t *testing.T) {
	t.Parallel()

	repo, _, mock := newTestRepository(t)
	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	pickupLocationID := pulid.MustNew("loc_")
	bridgeLocationID := pulid.MustNew("loc_")
	newDeliveryLocationID := pulid.MustNew("loc_")
	newMoveID := pulid.MustNew("sm_")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.id = .*sm\.organization_id = .*sm\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "distance", "version", "created_at", "updated_at",
		}).AddRow(moveID, buID, orgID, shipmentID, shipment.MoveStatusAssigned, true, 0, nil, 1, 1, 1))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*shipment_move_id.*IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version", "created_at", "updated_at",
		}).
			AddRow(pulid.MustNew("stp_"), buID, orgID, moveID, pickupLocationID, shipment.StopStatusNew, shipment.StopTypePickup, 0, 1, 2, 1, 1, 1).
			AddRow(pulid.MustNew("stp_"), buID, orgID, moveID, bridgeLocationID, shipment.StopStatusNew, shipment.StopTypeDelivery, 1, 3, 4, 1, 1, 1))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id IN.*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "primary_worker_id", "tractor_id", "trailer_id", "secondary_worker_id", "status", "version", "created_at", "updated_at",
		}))
	mock.ExpectExec(`UPDATE .*shipment_moves.*sequence = sequence \+ 1.*`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE .*stops.*status.*type.*version.*updated_at.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO .*shipment_moves.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow(newMoveID, 1))
	mock.ExpectQuery(`INSERT INTO .*stops.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow(pulid.MustNew("stp_"), 1).AddRow(pulid.MustNew("stp_"), 1))
	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "version", "created_at", "updated_at",
		}).AddRow(moveID, buID, orgID, shipmentID, shipment.MoveStatusAssigned, true, 0, 1, 1, 1))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*shipment_move_id.*IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id IN.*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "primary_worker_id", "tractor_id", "trailer_id", "secondary_worker_id", "status", "version", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*sm\.id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "version", "created_at", "updated_at",
		}).AddRow(newMoveID, buID, orgID, shipmentID, shipment.MoveStatusNew, true, 1, 1, 1, 1))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*shipment_move_id.*IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type", "sequence", "scheduled_window_start", "scheduled_window_end", "version", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`SELECT .*FROM "assignments" AS "a".*a\.shipment_move_id IN.*a\.organization_id = .*a\.business_unit_id = .*a\.archived_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "primary_worker_id", "tractor_id", "trailer_id", "secondary_worker_id", "status", "version", "created_at", "updated_at",
		}))
	mock.ExpectCommit()

	response, err := repo.SplitMove(t.Context(), &repositories.SplitMoveRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		MoveID:                moveID,
		NewDeliveryLocationID: newDeliveryLocationID,
		SplitPickupTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: 5,
			ScheduledWindowEnd:   int64Ptr(6),
		},
		NewDeliveryTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: 7,
			ScheduledWindowEnd:   int64Ptr(8),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.OriginalMove)
	require.NotNil(t, response.NewMove)
	require.NoError(t, mock.ExpectationsWereMet())
}

func newShipmentEntity() *shipment.Shipment {
	stopOne := &shipment.Stop{
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipment.StopStatusNew,
		Type:                 shipment.StopTypePickup,
		ScheduleType:         shipment.StopScheduleTypeOpen,
		Sequence:             0,
		ScheduledWindowStart: 1,
		ScheduledWindowEnd:   int64Ptr(1),
	}
	stopTwo := &shipment.Stop{
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipment.StopStatusNew,
		Type:                 shipment.StopTypeDelivery,
		ScheduleType:         shipment.StopScheduleTypeOpen,
		Sequence:             1,
		ScheduledWindowStart: 2,
		ScheduledWindowEnd:   int64Ptr(2),
	}

	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Moves: []*shipment.ShipmentMove{
			{
				Status:   shipment.MoveStatusNew,
				Loaded:   true,
				Sequence: 0,
				Stops:    []*shipment.Stop{stopOne, stopTwo},
			},
		},
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
