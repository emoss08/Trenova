//go:build integration

package recurringshipmentrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type quotaCase struct {
	ctx    context.Context
	repo   *repository
	series *recurringshipment.RecurringShipment
	count  func() int64
}

func seriesFromShipment(t *testing.T) quotaCase {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-RSH",
		ProNumber:    "PRO-RSH",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	scoped := data.Ctx(ctx)
	series := &recurringshipment.RecurringShipment{
		OrganizationID:   data.Organization.ID,
		BusinessUnitID:   data.BusinessUnit.ID,
		SourceShipmentID: graph.Shipment.ID,
		CustomerID:       graph.Shipment.CustomerID,
		Name:             "Weekday lane",
		CronExpression:   "0 8 * * 1-5",
		Timezone:         "America/New_York",
		AutoGenerate:     true,
	}
	repo := &repository{db: postgres.NewTestConnection(db), l: zap.NewNop()}
	count := func() int64 {
		n, err := recurringshipmentdb.RecurringShipments.Query().Count(scoped, db)
		require.NoError(t, err)
		return n
	}

	return quotaCase{ctx: scoped, repo: repo, series: series, count: count}
}

func TestInsertSeriesRefusesASeriesPastTheQuota(t *testing.T) {
	qc := seriesFromShipment(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	qc.repo.quota = guard

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: seriesTenant(qc.series),
		Meter:      platformcatalog.MeterRecurringShipmentSeries,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterRecurringShipmentSeries), 1, 1, "free_demo",
	))

	err := qc.repo.insertSeries(qc.ctx, qc.series)

	require.True(t, errortypes.IsQuotaExceededError(err))
	require.Zero(t, qc.count(), "a refused series writes nothing")
}

func TestInsertSeriesWritesUnderTheQuota(t *testing.T) {
	qc := seriesFromShipment(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	qc.repo.quota = guard

	guard.EXPECT().Enforce(mock.Anything, mock.Anything).Return(nil)

	require.NoError(t, qc.repo.insertSeries(qc.ctx, qc.series))
	require.Equal(t, int64(1), qc.count())
	require.Equal(t, recurringshipment.StatusActive, qc.series.Status)
	require.Equal(t, recurringshipment.ExceptionPolicySkip, qc.series.ExceptionPolicy)
}
