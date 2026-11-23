//go:build integration

package aianalyticsrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDailyUsageFoldsEachLocalDayOnceAndCountsUnpricedCalls(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx, tenant := seededTenant(ctx, t, db)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	zone, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	localDay := time.Date(2026, time.March, 10, 0, 0, 0, 0, zone)
	evening := localDay.Add(22 * time.Hour).Unix()
	nextLocalDay := localDay.AddDate(0, 0, 1)

	cost := decimal.RequireFromString("0.0200")
	call := func(at int64, priced bool) *aiusage.AIUsageRecord {
		row := &aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			ProviderKind: aiprovider.KindOpenAIChat, Model: "gpt-5.1-mini",
			Task: aitask.TaskAssistantChat, Surface: aiusage.SurfaceChat,
			Feature: aiusage.FeatureAgentTurn, UserID: tenant.UserID,
			Succeeded: true, LatencyMs: 900, InputTokens: 1000, CacheReadTokens: 400,
			OutputTokens: 200, ReasoningTokens: 50, CreatedAt: at,
		}
		if priced {
			row.CostUSD = &cost
		}
		return row
	}
	usage := []*aiusage.AIUsageRecord{
		call(localDay.Add(9*time.Hour).Unix(), true),
		call(evening, false),
		call(nextLocalDay.Add(time.Hour).Unix(), true),
	}
	insert(ctx, t, db, aiusagedb.AIUsageRecords, usage...)
	backdateUsage(ctx, t, db, usage...)

	req := aianalyticsports.RefreshRequest{
		TenantInfo: tenant,
		From:       evening - 600,
		To:         evening + 600,
		Stamp:      1,
	}
	written, err := repo.RefreshDailyUsage(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, written, "a window inside one local day folds that day alone")

	d := aiusagedb.DailyUsages
	rows, err := d.Query().Order(d.UsageDate.Asc()).Limit(10).List(ctx, db)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	day := rows[0]
	assert.Equal(t, localDay.Unix(), day.UsageDate,
		"a call at 10 pm Pacific, after midnight UTC, belongs to the organization's day")
	assert.Equal(t, tenant.UserID, day.UserID)
	assert.Equal(t, int64(2), day.Calls, "the whole local day is recomputed, not only the window")
	assert.Equal(t, int64(1), day.UnpricedCalls)
	assert.Equal(t, int64(2*600), day.InputTokens, "OpenAI's cached input is taken out")
	assert.Equal(t, int64(2*400), day.CacheReadTokens)
	assert.Equal(t, int64(2*200), day.OutputTokens)
	assert.Equal(t, int64(2*(600+400+200)), day.TotalTokens,
		"reasoning is part of the output and is not added twice")
	assert.True(t, cost.Equal(day.CostUSD), "only the priced call is in the estimate")

	again, err := repo.RefreshDailyUsage(ctx, req)
	require.NoError(t, err)
	assert.Zero(t, again, "folding the same window again rewrites nothing")

	written, err = repo.RefreshDailyUsage(ctx, aianalyticsports.RefreshRequest{
		TenantInfo: tenant,
		From:       localDay.Unix(),
		To:         nextLocalDay.Add(2 * time.Hour).Unix(),
		Stamp:      2,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, written, "only the next day is new; the folded day is unchanged")

	rows, err = d.Query().Order(d.UsageDate.Asc()).Limit(10).List(ctx, db)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, nextLocalDay.Unix(), rows[1].UsageDate)
	assert.Equal(t, int64(1), rows[1].Calls)
}
