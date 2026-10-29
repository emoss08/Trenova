//go:build integration

package shipmentprovider

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchconsolerepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/costingcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type providerFixture struct {
	provider *Provider
	ctx      context.Context
	info     pagination.TenantInfo
	deps     providerMocks
}

type providerMocks struct {
	dispatch *dispatchcontrolportsmock.MockDispatchControlRepository
	costing  *costingcontrolportsmock.MockCostingControlRepository
	actuals  *costingcontrolportsmock.MockCostingActualsRepository
}

func newProviderFixture(t *testing.T) *providerFixture {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	info := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, admin, data, info)
	testutil.CreateShipmentGraph(t, ctx, admin, fixture, info, testutil.ShipmentGraphParams{
		BOL:          "BOL-ANALYTICS",
		ProNumber:    "PRO-ANALYTICS",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipmentvocab.MoveStatus{shipmentvocab.MoveStatusNew},
	})

	deps := providerMocks{
		dispatch: dispatchcontrolportsmock.NewMockDispatchControlRepository(t),
		costing:  costingcontrolportsmock.NewMockCostingControlRepository(t),
		actuals:  costingcontrolportsmock.NewMockCostingActualsRepository(t),
	}

	conn := dbtest.NewRLSConnection(t, admin, dsn)

	return &providerFixture{
		provider: &Provider{
			l:            zap.NewNop(),
			db:           conn,
			dispatchRepo: deps.dispatch,
			dispatchConsole: dispatchconsolerepository.New(
				dispatchconsolerepository.Params{DB: conn, Logger: zap.NewNop()},
			),
			costingService: costingservice.NewTestService(costingservice.TestServiceParams{
				Repo:        deps.costing,
				ActualsRepo: deps.actuals,
			}),
		},
		ctx:  data.Ctx(t.Context()),
		info: info,
		deps: deps,
	}
}

func (f *providerFixture) options(include string) *kernelports.AnalyticsRequestOptions {
	return &kernelports.AnalyticsRequestOptions{
		OrgID:    f.info.OrgID,
		BuID:     f.info.BuID,
		UserID:   f.info.UserID,
		Timezone: "UTC",
		Include:  include,
	}
}

func TestGetAnalyticsDataRunsEveryCardQuery(t *testing.T) {
	f := newProviderFixture(t)
	target := 96.5
	control := &costingcontrol.CostingControl{
		ID:                   pulid.MustNew("cstc_"),
		BusinessUnitID:       f.info.BuID,
		OrganizationID:       f.info.OrgID,
		MilesPerGallon:       costingcontrol.DefaultMilesPerGallon(),
		IncludeDeadheadMiles: true,
		GLRollingMonths:      3,
	}
	control.Categories = costingcontrol.DefaultCategories()

	f.deps.costing.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)
	f.deps.actuals.EXPECT().
		FleetCostAggregates(mock.Anything, mock.Anything).
		Return(&costingcontrolports.FleetCostAggregatesResult{}, nil)
	f.deps.dispatch.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&dispatchcontrol.DispatchControl{ServiceFailureTarget: &target}, nil)

	data, err := f.provider.GetAnalyticsData(f.ctx, f.options(""))
	require.NoError(t, err)

	for _, key := range []string{
		"activeShipments", "onTimePercent", "revenueToday", "emptyMilePercent", "atRisk",
		"unassigned", "readyToDispatch", "detentionWatchlist", "customerMix",
		"tomorrowsPickups", "laneHeatmap", "profitability",
	} {
		assert.Contains(t, data, key)
	}
	assert.NotContains(t, data, "tenderAccept")
	assert.NotContains(t, data, "hosNearLimit")
	assert.NotContains(t, data, "detentionAlerts")

	pickups, ok := data["tomorrowsPickups"].(*TomorrowsPickupsCard)
	require.True(t, ok)
	require.Len(t, pickups.Pickups, 1)
	assert.Equal(t, "PRO-ANALYTICS", pickups.Pickups[0].ProNumber)
	assert.Equal(t, TomorrowPickupStatusUnassigned, pickups.Pickups[0].Status)
}

func TestGetAnalyticsDataGatesWhatItReturns(t *testing.T) {
	f := newProviderFixture(t)

	visible := reader{}.provider()
	f.provider.views, f.provider.fieldAccess = visible.views, visible.fieldAccess
	data, err := f.provider.GetAnalyticsData(f.ctx, f.options(tomorrowsPickupsInclude))
	require.NoError(t, err)

	card, ok := data["tomorrowsPickups"].(*TomorrowsPickupsCard)
	require.True(t, ok)
	require.Len(t, card.Pickups, 1)
	require.NotNil(t, card.Pickups[0].Customer)
	assert.Equal(t, "Integration Customer", *card.Pickups[0].Customer)

	gated := reader{denied: withoutLinkedReads, ceiling: belowInternal}.provider()
	f.provider.views, f.provider.fieldAccess = gated.views, gated.fieldAccess
	data, err = f.provider.GetAnalyticsData(f.ctx, f.options(tomorrowsPickupsInclude))
	require.NoError(t, err)

	card, ok = data["tomorrowsPickups"].(*TomorrowsPickupsCard)
	require.True(t, ok)
	require.Len(t, card.Pickups, 1)
	assert.Nil(t, card.Pickups[0].Customer)
	assert.Nil(t, card.Pickups[0].Driver)
}
