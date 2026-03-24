package shipmentadditionalchargerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, bun.IDB, sqlmock.Sqlmock) {
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

func TestSyncForShipment_InsertsNewAdditionalCharges(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()

	mock.ExpectQuery(`SELECT .*FROM "additional_charges" AS "ac".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit", "version",
		}))
	mock.ExpectQuery(`INSERT INTO "additional_charges".*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(0))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.False(t, entity.AdditionalCharges[0].ID.IsNil())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_UpdatesExistingAdditionalChargeAndDeletesRemovedCharge(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	keepID := pulid.MustNew("ac_")
	deleteID := pulid.MustNew("ac_")

	entity.AdditionalCharges[0].ID = keepID
	entity.AdditionalCharges[0].Version = 1

	mock.ExpectQuery(`SELECT .*FROM "additional_charges" AS "ac".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit", "version",
		}).
			AddRow(keepID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, entity.AdditionalCharges[0].AccessorialChargeID, entity.AdditionalCharges[0].Method, entity.AdditionalCharges[0].Amount.String(), entity.AdditionalCharges[0].Unit, 1).
			AddRow(deleteID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, pulid.MustNew("acc_"), accessorialcharge.MethodFlat, decimal.NewFromInt(5).String(), 1, 1))
	mock.ExpectExec(`UPDATE "additional_charges" AS "ac".*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "additional_charges" AS "ac".*id IN .*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.Equal(t, int64(2), entity.AdditionalCharges[0].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_UpdatesExistingAccessorialChargeID(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	existingID := pulid.MustNew("ac_")
	originalAccessorialChargeID := pulid.MustNew("acc_")
	replacementAccessorialChargeID := pulid.MustNew("acc_")

	entity.AdditionalCharges[0].ID = existingID
	entity.AdditionalCharges[0].AccessorialChargeID = replacementAccessorialChargeID
	entity.AdditionalCharges[0].Version = 1

	mock.ExpectQuery(`SELECT .*FROM "additional_charges" AS "ac".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit", "version",
		}).
			AddRow(existingID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, originalAccessorialChargeID, entity.AdditionalCharges[0].Method, entity.AdditionalCharges[0].Amount.String(), entity.AdditionalCharges[0].Unit, 1))
	mock.ExpectExec(`UPDATE "additional_charges" AS "ac" SET accessorial_charge_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.Equal(t, replacementAccessorialChargeID, entity.AdditionalCharges[0].AccessorialChargeID)
	assert.Equal(t, int64(2), entity.AdditionalCharges[0].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_RejectsUnknownAdditionalChargeID(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	entity.AdditionalCharges[0].ID = pulid.MustNew("ac_")

	mock.ExpectQuery(`SELECT .*FROM "additional_charges" AS "ac".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit", "version",
		}))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func newShipmentEntity() *shipment.Shipment {
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				AccessorialChargeID: pulid.MustNew("acc_"),
				Method:              accessorialcharge.MethodFlat,
				Amount:              decimal.NewFromInt(10),
				Unit:                1,
			},
		},
	}
}
