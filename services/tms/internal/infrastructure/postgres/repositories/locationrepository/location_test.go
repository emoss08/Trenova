package locationrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
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
	}, mock
}

func TestGetByIDHydratesDrawGeofenceVertices(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)

	locationID := pulid.MustNew("loc_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	stateID := pulid.MustNew("us_")
	categoryID := pulid.MustNew("lc_")

	geometry, err := postgis.PolygonGeometry([]postgis.Vertex{
		{Latitude: 26.766562573433664, Longitude: -80.0526537300732},
		{Latitude: 26.766792474681477, Longitude: -80.05071302469669},
		{Latitude: 26.767414037693534, Longitude: -80.05068083818851},
		{Latitude: 26.767481091769568, Longitude: -80.05278247610592},
	})
	require.NoError(t, err)

	rawGeometry, err := geometry.Value()
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT .*loc\.geofence_geometry AS geofence_geometry.* FROM "locations" AS "loc"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"location_category_id",
			"state_id",
			"status",
			"code",
			"name",
			"description",
			"address_line_1",
			"address_line_2",
			"city",
			"postal_code",
			"place_id",
			"is_geocoded",
			"longitude",
			"latitude",
			"geofence_type",
			"geofence_radius_meters",
			"version",
			"created_at",
			"updated_at",
			"geofence_geometry",
		}).AddRow(
			locationID.String(),
			buID.String(),
			orgID.String(),
			categoryID.String(),
			stateID.String(),
			"Active",
			"TERM-LA",
			"Port of Palm Beach",
			"Primary terminal in Los Angeles metro area",
			"Port of Palm Beach, Riviera Beach, FL 33404, USA",
			"",
			"Riviera Beach",
			"33404",
			"ChIJCep1AlPU2IgRmVuPhut1Ri8",
			false,
			-80.05173165714722,
			26.767021832601614,
			"draw",
			nil,
			4,
			1776731491,
			1776970506,
			rawGeometry,
		))
	entity, err := repo.GetByID(t.Context(), paginationRequest(locationID, orgID, buID))
	require.NoError(t, err)
	require.NotNil(t, entity)

	assert.Equal(t, "draw", string(entity.GeofenceType))
	assert.Len(t, entity.GeofenceVertices, 4)
	assert.InDelta(t, 26.766562573433664, entity.GeofenceVertices[0].Latitude, 1e-12)
	assert.InDelta(t, -80.0526537300732, entity.GeofenceVertices[0].Longitude, 1e-12)

	require.NoError(t, mock.ExpectationsWereMet())
}

func paginationRequest(
	locationID pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
) repositories.GetLocationByIDRequest {
	return repositories.GetLocationByIDRequest{
		ID: locationID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}
}
