package shipmentrepository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newCancelTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
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
		generator: testutil.TestSequenceGenerator{BatchValues: []string{"PRO-1"}},
	}, mock
}

func TestCancel_UpdatesShipmentAndComponents(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	moveID := pulid.MustNew("sm_")
	userID := pulid.MustNew("usr_")

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE .*shipments.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "cancel_reason", "canceled_at", "canceled_by_id", "version",
		}).AddRow(shipmentID, orgID, buID, shipment.StatusCanceled, "customer request", 1700000000, userID, 2))
	mock.ExpectQuery(`SELECT "sm"\."id" FROM "shipment_moves" AS "sm" WHERE \(sm\.shipment_id = .*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(moveID))
	mock.ExpectExec(`UPDATE .*shipment_moves.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*assignments.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*stops.*`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	entity, err := repo.Cancel(t.Context(), &repositories.CancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID:   shipmentID,
		CanceledByID: userID,
		CanceledAt:   1700000000,
		CancelReason: "customer request",
	})

	require.NoError(t, err)
	assert.Equal(t, shipment.StatusCanceled, entity.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUncancel_UpdatesShipmentAndComponents(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	moveID := pulid.MustNew("sm_")

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE .*shipments.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "cancel_reason", "canceled_at", "canceled_by_id", "version",
		}).AddRow(shipmentID, orgID, buID, shipment.StatusNew, "", nil, "", 3))
	mock.ExpectQuery(`SELECT "sm"\."id" FROM "shipment_moves" AS "sm" WHERE \(sm\.shipment_id = .*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(moveID))
	mock.ExpectExec(`UPDATE .*shipment_moves.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*assignments.*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE .*stops.*`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	entity, err := repo.Uncancel(t.Context(), &repositories.UncancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID: shipmentID,
	})

	require.NoError(t, err)
	assert.Equal(t, shipment.StatusNew, entity.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransferOwnership_UpdatesOwner(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	ownerID := pulid.MustNew("usr_")

	mock.ExpectQuery(`UPDATE .*shipments.*owner_id = .*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "owner_id", "version",
		}).AddRow(shipmentID, orgID, buID, ownerID, 4))

	entity, err := repo.TransferOwnership(t.Context(), &repositories.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID: shipmentID,
		OwnerID:    ownerID,
	})

	require.NoError(t, err)
	assert.Equal(t, ownerID, entity.OwnerID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckForDuplicateBOLs_ExcludesCanceledAndCurrentShipment(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	excludeID := pulid.MustNew("shp_")

	mock.ExpectQuery(`SELECT .*FROM "shipments" AS "sp".*organization_id = .*business_unit_id = .*bol = .*status != .*id != .*`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pro_number"}).
			AddRow(pulid.MustNew("shp_"), "PRO-101"))

	results, err := repo.CheckForDuplicateBOLs(t.Context(), &repositories.DuplicateBOLCheckRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		BOL:        "BOL-1",
		ShipmentID: &excludeID,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "PRO-101", results[0].ProNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateDerivedState_UpdatesShipmentAndSyncsAdditionalCharges(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	repo.additionalChargeRepository = &mockShipmentAdditionalChargeRepository{}

	entity := &shipment.Shipment{
		ID:                  shipmentID,
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		Status:              shipment.StatusAssigned,
		Version:             3,
		FreightChargeAmount: decimalFromString(t, "100.00"),
		OtherChargeAmount:   decimalFromString(t, "10.00"),
		TotalChargeAmount:   decimalFromString(t, "110.00"),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE .*shipments.*status.*actual_ship_date.*actual_delivery_date.*freight_charge_amount.*other_charge_amount.*total_charge_amount.*version.*updated_at.*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "version",
		}).AddRow(shipmentID, orgID, buID, shipment.StatusAssigned, 4))
	mock.ExpectCommit()

	updated, err := repo.UpdateDerivedState(t.Context(), entity)

	require.NoError(t, err)
	assert.Equal(t, int64(4), updated.Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoDelayShipments_UpdatesEligibleShipments(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mock.ExpectBegin()
	mock.ExpectQuery(`WITH "delayed_cte" AS .*JOIN shipment_controls AS sc ON sc\.organization_id = sp\.organization_id AND sc\.business_unit_id = sp\.business_unit_id.*sc\.auto_delay_shipments = TRUE.*SELECT .* FROM "shipments" AS "sp".*sp\.id IN \(SELECT shipment_id FROM delayed_cte\)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "version",
		}).AddRow(shipmentID, orgID, buID, shipment.StatusAssigned, 1))
	mock.ExpectExec(`UPDATE .*shipments.*status = .*updated_at = .*sp\.id IN .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	entities, err := repo.AutoDelayShipments(t.Context())

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, shipment.StatusDelayed, entities[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

type mockShipmentAdditionalChargeRepository struct{}

func (m *mockShipmentAdditionalChargeRepository) SyncForShipment(
	_ context.Context,
	_ bun.IDB,
	_ *shipment.Shipment,
) error {
	return nil
}

func decimalFromString(t *testing.T, value string) decimal.NullDecimal {
	t.Helper()

	d, err := decimal.NewFromString(value)
	require.NoError(t, err)

	return decimal.NullDecimal{Decimal: d, Valid: true}
}
