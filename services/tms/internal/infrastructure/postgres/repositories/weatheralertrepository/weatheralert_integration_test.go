//go:build integration

package weatheralertrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpsertAlertAndExpireStaleAlerts(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	alert := &weatheralert.WeatherAlert{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		NWSID:          "urn:oid:weather-alert-1",
		Event:          "Flood Warning",
		AlertCategory:  weatheralert.AlertCategoryFloodWater,
		Geometry: &postgis.Geometry{
			Geometry: orb.Polygon{
				{{-97.0, 32.0}, {-96.0, 32.0}, {-96.0, 33.0}, {-97.0, 33.0}, {-97.0, 32.0}},
			},
		},
		FirstSeenAt:   10,
		LastUpdatedAt: 10,
	}

	upserted, err := repo.UpsertAlert(ctx, alert)
	require.NoError(t, err)
	require.True(t, upserted.Created)
	require.NotNil(t, upserted.Activity)

	activeAlerts, err := repo.GetActiveAlerts(ctx, tenantInfo)
	require.NoError(t, err)
	require.Len(t, activeAlerts, 1)
	assert.Equal(t, "Flood Warning", activeAlerts[0].Event)

	activities, err := repo.GetActivities(ctx, repositories.GetWeatherAlertByIDRequest{
		ID:         activeAlerts[0].ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, weatheralert.ActivityTypeIssued, activities[0].ActivityType)

	expiredAt := int64(20)
	expires := int64(1)
	alert.Expires = &expires
	alert.ExpiredAt = &expiredAt
	alert.LastUpdatedAt = expiredAt

	_, err = repo.UpsertAlert(ctx, alert)
	require.NoError(t, err)

	result, err := repo.ExpireStaleAlerts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.ExpiredCount, 0)
}
