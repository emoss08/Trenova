//go:build integration

package locationrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateRefusesALocationPastTheQuota(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	conn := dbtest.NewRLSConnection(t, admin, dsn)
	info := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	tenantCtx := dbscope.WithTenant(t.Context(), info.DBTenant())

	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := locationrepository.New(locationrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
		Quota:  guard,
	})
	entity := &location.Location{
		OrganizationID: info.OrgID,
		BusinessUnitID: info.BuID,
		StateID:        data.State.ID,
		Code:           "QUOTA-1",
		Name:           "Dock 4",
		GeofenceType:   geofence.TypeAuto,
	}

	guard.EXPECT().
		Enforce(mock.MatchedBy(func(ctx context.Context) bool {
			return keel.InTx(ctx, conn.Keel(ctx))
		}), &platformplanports.QuotaRequest{
			TenantInfo: info,
			Meter:      platformcatalog.MeterLocationsTotal,
			Quantity:   1,
		}).
		Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterLocationsTotal), 25, 25, "free_demo")).
		Once()

	created, err := repo.Create(tenantCtx, entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))

	var written int64
	require.NoError(t, admin.Pool().QueryRow(ctx,
		"SELECT count(*) FROM locations WHERE organization_id = $1 AND business_unit_id = $2 AND code = $3",
		info.OrgID.String(), info.BuID.String(), entity.Code,
	).Scan(&written))
	require.Zero(t, written)
}
