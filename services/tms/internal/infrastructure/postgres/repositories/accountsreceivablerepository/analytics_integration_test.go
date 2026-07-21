//go:build integration

package accountsreceivablerepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestARAnalyticsRepositoryQueriesExecute(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	repo := NewAnalytics(Params{DB: conn, Logger: zap.NewNop()})

	var org seededAROrg
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)

	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	asOf := time.Now().UTC().Unix()

	overview, err := repo.GetBalanceOverview(
		ctx,
		repositories.GetARAnalyticsRequest{TenantInfo: tenantInfo, AsOfDate: asOf},
	)
	require.NoError(t, err)
	require.NotNil(t, overview)
	assert.GreaterOrEqual(t, overview.TotalOpenMinor, int64(0))

	stats, err := repo.GetPaymentStats(
		ctx,
		repositories.GetARAnalyticsRequest{TenantInfo: tenantInfo, AsOfDate: asOf},
	)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.UnappliedCashMinor, int64(0))

	series, err := repo.ListBalanceSeries(
		ctx,
		repositories.ListARSeriesRequest{TenantInfo: tenantInfo, AsOfDate: asOf, Weeks: 13},
	)
	require.NoError(t, err)
	assert.Len(t, series, 13)
	for i := 1; i < len(series); i++ {
		assert.Greater(t, series[i].PeriodEnd, series[i-1].PeriodEnd)
	}

	agingTrend, err := repo.ListAgingTrend(
		ctx,
		repositories.ListARSeriesRequest{TenantInfo: tenantInfo, AsOfDate: asOf, Weeks: 13},
	)
	require.NoError(t, err)
	assert.Len(t, agingTrend, 13)

	cashFlow, err := repo.ListCashFlow(
		ctx,
		repositories.ListARCashFlowRequest{
			TenantInfo:  tenantInfo,
			AsOfDate:    asOf,
			PastWeeks:   6,
			FutureWeeks: 7,
		},
	)
	require.NoError(t, err)
	assert.Len(t, cashFlow, 13)
	forecastSeen := false
	for _, point := range cashFlow {
		if point.IsForecast {
			forecastSeen = true
			assert.Zero(t, point.ActualMinor)
		}
	}
	assert.True(t, forecastSeen)

	totals, err := repo.GetCollectionTotals(
		ctx,
		repositories.GetARCollectionMetricsRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			PeriodDays: 91,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, totals)
	assert.Equal(t, asOf, totals.PeriodEnd)

	topOverdue, err := repo.ListTopOverdueCustomers(
		ctx,
		repositories.ListARTopOverdueCustomersRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			Limit:      10,
		},
	)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(topOverdue), 10)

	worklist, err := repo.ListCollectionsWorklist(
		ctx,
		repositories.ListARCollectionsWorklistRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			Limit:      25,
		},
	)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(worklist), 25)

	var customer struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(
		t,
		db.NewSelect().
			Table("customers").
			Column("id").
			Where("organization_id = ?", org.ID).
			Where("business_unit_id = ?", org.BusinessUnitID).
			Limit(1).
			Scan(ctx, &customer),
	)

	snapshot, err := repo.GetCustomerSnapshot(
		ctx,
		repositories.GetARCustomerSnapshotRequest{
			TenantInfo: tenantInfo,
			CustomerID: customer.ID,
			AsOfDate:   asOf,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.NotEmpty(t, snapshot.CustomerName)
}
