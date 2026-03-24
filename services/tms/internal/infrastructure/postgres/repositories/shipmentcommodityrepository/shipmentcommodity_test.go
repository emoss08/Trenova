package shipmentcommodityrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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

func TestSyncForShipment_InsertsNewCommodities(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()

	mock.ExpectQuery(`SELECT .*FROM "shipment_commodities" AS "sc".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "commodity_id", "weight", "pieces", "version",
		}))
	mock.ExpectQuery(`INSERT INTO "shipment_commodities".*RETURNING .*`).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(0))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.False(t, entity.Commodities[0].ID.IsNil())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_UpdatesExistingCommodityAndDeletesRemovedCommodity(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	keepID := pulid.MustNew("sc_")
	deleteID := pulid.MustNew("sc_")

	entity.Commodities[0].ID = keepID
	entity.Commodities[0].Version = 1

	mock.ExpectQuery(`SELECT .*FROM "shipment_commodities" AS "sc".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "commodity_id", "weight", "pieces", "version",
		}).
			AddRow(keepID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, entity.Commodities[0].CommodityID, entity.Commodities[0].Weight, entity.Commodities[0].Pieces, 1).
			AddRow(deleteID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, pulid.MustNew("com_"), 50, 5, 1))
	mock.ExpectExec(`UPDATE "shipment_commodities" AS "sc".*`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "shipment_commodities" AS "sc".*id IN .*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.Equal(t, int64(2), entity.Commodities[0].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_UpdatesExistingCommodityID(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	existingID := pulid.MustNew("sc_")
	originalCommodityID := pulid.MustNew("com_")
	replacementCommodityID := pulid.MustNew("com_")

	entity.Commodities[0].ID = existingID
	entity.Commodities[0].CommodityID = replacementCommodityID
	entity.Commodities[0].Version = 1

	mock.ExpectQuery(`SELECT .*FROM "shipment_commodities" AS "sc".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "commodity_id", "weight", "pieces", "version",
		}).
			AddRow(existingID, entity.OrganizationID, entity.BusinessUnitID, entity.ID, originalCommodityID, entity.Commodities[0].Weight, entity.Commodities[0].Pieces, 1))
	mock.ExpectExec(`UPDATE "shipment_commodities" AS "sc" SET commodity_id = .*`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SyncForShipment(t.Context(), db, entity)

	require.NoError(t, err)
	assert.Equal(t, replacementCommodityID, entity.Commodities[0].CommodityID)
	assert.Equal(t, int64(2), entity.Commodities[0].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncForShipment_RejectsUnknownCommodityID(t *testing.T) {
	t.Parallel()

	repo, db, mock := newTestRepository(t)
	entity := newShipmentEntity()
	entity.Commodities[0].ID = pulid.MustNew("sc_")

	mock.ExpectQuery(`SELECT .*FROM "shipment_commodities" AS "sc".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "shipment_id", "commodity_id", "weight", "pieces", "version",
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
		Commodities: []*shipment.ShipmentCommodity{
			{
				CommodityID: pulid.MustNew("com_"),
				Weight:      100,
				Pieces:      10,
			},
		},
	}
}
