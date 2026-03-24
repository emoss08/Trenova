package commodityrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestGetByIDs_PreloadsHazardousMaterial(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	repo := &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	commodityID := pulid.MustNew("com_")
	hazmatID := pulid.MustNew("hm_")

	mock.ExpectQuery(`SELECT .* FROM "commodities" AS "com" .*WHERE .*organization_id.*business_unit_id.*IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "hazardous_material_id", "name", "description",
			"hazardous_material__id", "hazardous_material__organization_id", "hazardous_material__business_unit_id",
			"hazardous_material__name", "hazardous_material__description", "hazardous_material__class", "hazardous_material__packing_group",
		}).AddRow(
			commodityID, orgID, buID, hazmatID, "Paint", "Flammable paint",
			hazmatID, orgID, buID, "UN1263", "Flammable liquid", "Class3", "II",
		))

	entities, err := repo.GetByIDs(t.Context(), repositories.GetCommoditiesByIDsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		CommodityIDs: []pulid.ID{commodityID},
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
	require.NotNil(t, entities[0].HazardousMaterial)
	assert.Equal(t, hazmatID, entities[0].HazardousMaterial.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
