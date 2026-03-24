package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newDuplicateTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{
		db:        postgres.NewTestConnection(bunDB),
		l:         zap.NewNop(),
		generator: testutil.TestSequenceGenerator{BatchValues: []string{"PRO-1", "PRO-2"}},
	}, mock
}

func TestBulkDuplicate_CreatesShipmentMoveAndStopCopies(t *testing.T) {
	t.Parallel()

	repo, mock := newDuplicateTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	moveID := pulid.MustNew("sm_")

	req := &repositories.BulkDuplicateShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  orgID,
			BuID:   buID,
			UserID: pulid.MustNew("usr_"),
		},
		ShipmentID: shipmentID,
		Count:      2,
	}

	mock.ExpectQuery(`SELECT .*FROM "shipments" AS "sp".*sp\.id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "service_type_id", "customer_id",
			"formula_template_id", "status", "pro_number", "bol", "rating_unit",
		}).AddRow(
			shipmentID, buID, orgID, pulid.MustNew("svc_"), pulid.MustNew("cus_"),
			pulid.MustNew("fmt_"), shipment.StatusAssigned, "PRO-OLD", "BOL-OLD", 1,
		))
	mock.ExpectQuery(`SELECT .*FROM "shipment_moves" AS "sm".*"shipment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "status", "loaded", "sequence", "distance",
		}).AddRow(
			moveID, buID, orgID, shipmentID, shipment.MoveStatusAssigned, true, 0, 123.4,
		))
	mock.ExpectQuery(`SELECT .*FROM "stops" AS "stp".*"shipment_move_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_move_id", "location_id", "status", "type",
			"sequence", "scheduled_window_start", "scheduled_window_end",
		}).
			AddRow(pulid.MustNew("stp_"), buID, orgID, moveID, pulid.MustNew("loc_"), shipment.StopStatusCompleted, shipment.StopTypePickup, 0, 100, 110).
			AddRow(pulid.MustNew("stp_"), buID, orgID, moveID, pulid.MustNew("loc_"), shipment.StopStatusInTransit, shipment.StopTypeDelivery, 1, 200, 210),
		)
	mock.ExpectQuery(`SELECT .*FROM "additional_charges" AS "ac".*"shipment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit",
		}).AddRow(
			pulid.MustNew("ac_"), buID, orgID, shipmentID, pulid.MustNew("acc_"), "Flat", "10.0000", 1,
		))
	mock.ExpectQuery(`SELECT .*FROM "shipment_commodities" AS "sc".*"shipment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "commodity_id", "weight", "pieces",
		}).AddRow(
			pulid.MustNew("sc_"), buID, orgID, shipmentID, pulid.MustNew("com_"), 1000, 10,
		))
	mock.ExpectQuery(`SELECT .*code.*FROM "business_units"`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("BU"))
	mock.ExpectQuery(`SELECT .*code.*FROM locations`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("LOC"))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "shipments"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO "shipment_moves"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO "stops"`).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(`INSERT INTO "additional_charges"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO "shipment_commodities"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	entities, err := repo.BulkDuplicate(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, entities, 2)
	assert.Equal(t, "PRO-1", entities[0].ProNumber)
	assert.Equal(t, shipment.StatusNew, entities[0].Status)
	assert.Len(t, entities[0].Moves, 1)
	assert.Len(t, entities[0].Moves[0].Stops, 2)
	assert.Len(t, entities[0].AdditionalCharges, 1)
	assert.Len(t, entities[0].Commodities, 1)
	assert.Equal(t, shipment.MoveStatusNew, entities[0].Moves[0].Status)
	assert.Equal(t, shipment.StopStatusNew, entities[0].Moves[0].Stops[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}
