//go:build integration

package aiusagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestSummary_BreaksUsageDownByFeature(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	cols := buncolgen.DefinitionColumns
	var tenant tenantRow
	require.NoError(t, db.NewSelect().
		TableExpr(buncolgen.DefinitionTable.As(buncolgen.DefinitionTable.Alias)).
		Column(cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		Limit(1).
		Scan(ctx, &tenant))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	since := timeutils.NowUnix() - 60
	documentID := pulid.MustNew("doc_").String()

	type call struct {
		feature   aiusage.Feature
		surface   aiusage.Surface
		subject   aiusage.Subject
		latencyMs int64
		cost      string
	}
	for _, c := range []call{
		{
			feature:   aiusage.FeatureDocumentIntelligenceExtract,
			surface:   aiusage.SurfaceStructured,
			subject:   aiusage.Subject{Type: aiusage.SubjectTypeDocument, ID: documentID},
			latencyMs: 2000,
			cost:      "0.30",
		},
		{
			feature:   aiusage.FeatureDocumentIntelligenceExtract,
			surface:   aiusage.SurfaceBackground,
			subject:   aiusage.Subject{Type: aiusage.SubjectTypeDocument, ID: documentID},
			latencyMs: 600_000,
			cost:      "0.10",
		},
		{
			feature:   aiusage.FeatureFormulaGenerate,
			surface:   aiusage.SurfaceStructured,
			subject:   aiusage.Subject{Type: aiusage.SubjectTypeFormulaSchema, ID: "shipment"},
			latencyMs: 800,
			cost:      "0.05",
		},
		{surface: aiusage.SurfaceChat, latencyMs: 400, cost: "0.01"},
	} {
		amount := decimal.RequireFromString(c.cost)
		record := &aiusage.AIUsageRecord{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
			ProviderKind:   aiprovider.KindOpenAIChat,
			Model:          "test-model",
			Task:           aiprovider.TaskGeneral,
			Surface:        c.surface,
			Feature:        c.feature,
			Succeeded:      true,
			LatencyMs:      c.latencyMs,
			InputTokens:    100,
			OutputTokens:   10,
			CostUSD:        &amount,
		}
		if c.subject.Recordable() {
			record.SubjectType = c.subject.Type
			record.SubjectID = c.subject.ID
		}
		require.NoError(t, repo.Create(ctx, record))
	}

	summary, err := repo.Summary(ctx, repositories.AIUsageSummaryRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: tenant.OrganizationID,
			BuID:  tenant.BusinessUnitID,
		},
		Since: since,
	})
	require.NoError(t, err)

	byFeature := make(map[aiusage.Feature]repositories.AIUsageFeatureTotals, len(summary.ByFeature))
	for _, slice := range summary.ByFeature {
		byFeature[slice.Feature] = slice
	}

	extract := byFeature[aiusage.FeatureDocumentIntelligenceExtract]
	assert.Equal(t, 2, extract.Calls)
	assert.Equal(t, int64(200), extract.InputTokens)
	assert.True(t,
		decimal.RequireFromString(extract.CostUSD).Equal(decimal.RequireFromString("0.40")),
		extract.CostUSD,
	)
	assert.Equal(t, int64(2000), extract.LatencyP50,
		"a background call's wait is not a person's wait")

	assert.Equal(t, 1, byFeature[aiusage.FeatureFormulaGenerate].Calls)
	assert.Equal(t, 1, byFeature[""].Calls, "a call that named no feature is still counted")
	assert.Equal(t, aiusage.FeatureDocumentIntelligenceExtract, summary.ByFeature[0].Feature,
		"the busiest feature comes first")
	assert.Less(t, summary.Totals.LatencyP95, int64(600_000))

	bad := &aiusage.AIUsageRecord{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ProviderKind:   aiprovider.KindOpenAIChat,
		Model:          "test-model",
		Task:           aiprovider.TaskGeneral,
		Surface:        aiusage.SurfaceChat,
		Feature:        aiusage.Feature("Unknown"),
		Succeeded:      true,
	}
	require.Error(t, repo.Create(ctx, bad),
		"the check constraint refuses a feature nobody declared")

	halfSubject := &aiusage.AIUsageRecord{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ProviderKind:   aiprovider.KindOpenAIChat,
		Model:          "test-model",
		Task:           aiprovider.TaskGeneral,
		Surface:        aiusage.SurfaceChat,
		SubjectType:    aiusage.SubjectTypeDocument,
		Succeeded:      true,
	}
	require.Error(t, repo.Create(ctx, halfSubject), "a subject type without an id names nothing")
}
