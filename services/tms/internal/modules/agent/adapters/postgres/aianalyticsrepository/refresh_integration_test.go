//go:build integration

package aianalyticsrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func seededTenant(ctx context.Context, t *testing.T, db *bun.DB) pagination.TenantInfo {
	t.Helper()

	data := seedtest.SeedFullTestData(t, ctx, db)

	return pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
}

func insert(ctx context.Context, t *testing.T, db *bun.DB, models ...any) {
	t.Helper()
	for _, model := range models {
		_, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)
	}
}

func TestRefreshWorksOutUnitsToolsAndProviderHoursFromTheSourceRecords(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seededTenant(ctx, t, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	now := timeutils.NowUnix()
	vllm := pulid.MustNew("aiprv_")
	groq := pulid.MustNew("aiprv_")

	run := &agent.AgentRun{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		AgentType: agent.TypeBillingException, SubjectType: agent.SubjectShipment,
		SubjectID: pulid.MustNew("shp_"), Status: agent.RunStatusCompleted,
		PromptVersion: "v1", InputContextHash: "hash", StartedAt: now - 30, CompletedAt: new(int64),
	}
	*run.CompletedAt = now - 10
	stopped := &agent.AgentRun{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		AgentType: agent.TypeGeneral, SubjectType: agent.SubjectShipment,
		SubjectID: pulid.MustNew("shp_"), Status: agent.RunStatusCompleted,
		PromptVersion: "v1", InputContextHash: "hash", StartedAt: now - 20,
		Fingerprint: &agent.Fingerprint{Model: "claude-sonnet-5", StoppedAtLimit: true},
	}
	thread := &conversation.Thread{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		UserID: tenant.UserID, AgentDefinitionID: pulid.MustNew("agd_"),
		Title: "Analytics fixture",
	}
	turn := &conversation.AssistantTurn{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		UserID: tenant.UserID,
		Origin: conversation.AssistantTurnOriginPerson, Status: conversation.AssistantTurnStatusStopped,
		StartedAt:   now - 15,
		Fingerprint: &agent.Fingerprint{Model: "gpt-5.1-mini"},
	}
	compaction := &conversation.AssistantTurn{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		UserID: tenant.UserID,
		Origin: conversation.AssistantTurnOriginCompaction, Status: conversation.AssistantTurnStatusCompleted,
		Fingerprint: &agent.Fingerprint{Model: "gpt-5.1-mini"},
	}
	insert(ctx, t, db, thread)
	turn.ThreadID, compaction.ThreadID = thread.ID, thread.ID
	insert(ctx, t, db, run, stopped, turn, compaction)
	_, err := db.NewInsert().Model(&agent.AgentException{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, RunID: stopped.ID,
		Category: agent.CategoryUnableToDiagnose, SubjectType: agent.SubjectShipment,
		SubjectID: stopped.SubjectID, AttemptSummary: "could not", Evidence: []agent.EvidenceRef{},
	}).Exec(ctx)
	require.NoError(t, err)

	cost := decimal.RequireFromString("0.0123")
	firstToken := int64(450)
	insert(ctx, t, db,
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, ProviderID: vllm,
			ProviderKind: aiprovider.KindOpenAIChat, Model: "meta-llama/Llama-3.3-70B-Instruct",
			Task: aiprovider.TaskBillingDiagnosis, Surface: aiusage.SurfaceBackground,
			Succeeded: false, ErrorClass: "provider_error", ErrorMessage: "401 Unauthorized",
			LatencyMs: 300, TraceID: "aaaabbbbccccddddeeeeffff00001111",
			OwnerKind: agent.RunOwnerAgentRun, OwnerID: run.ID, CreatedAt: now - 25,
		},
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, ProviderID: groq,
			ProviderKind: aiprovider.KindOpenAIChat, Model: "llama-3.3-70b-versatile",
			Task: aiprovider.TaskBillingDiagnosis, Surface: aiusage.SurfaceBackground,
			Succeeded: true, LatencyMs: 2100, CostUSD: &cost, FirstTokenMs: &firstToken,
			Failover: true, TraceID: "aaaabbbbccccddddeeeeffff00001111",
			OwnerKind: agent.RunOwnerAgentRun, OwnerID: run.ID, CreatedAt: now - 20,
		},
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, ProviderID: groq,
			ProviderKind: aiprovider.KindOpenAIChat, Model: "qwen2.5:14b",
			Task: aiprovider.TaskDocumentClassification, Surface: aiusage.SurfaceStructured,
			Succeeded: true, LatencyMs: 400, CreatedAt: now - 5,
		},
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, ProviderID: groq,
			ProviderKind: aiprovider.KindOpenAIChat, Model: "nomic-embed-text",
			Task: aiprovider.TaskEmbedding, Surface: aiusage.SurfaceIndexing,
			Succeeded: true, LatencyMs: 40, CreatedAt: now - 5,
		},
	)
	insert(ctx, t, db,
		&agent.AgentRunStep{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			OwnerKind: string(
				agent.RunOwnerAgentRun,
			), OwnerID: run.ID, Kind: "Tool", Status: "Completed",
			StepKey: "k1", ToolName: "lookup_rate", Outcome: map[string]any{"verdict": "ran"},
		},
		&agent.AgentRunStep{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			OwnerKind: string(
				agent.RunOwnerAgentRun,
			), OwnerID: run.ID, Kind: "Tool", Status: "Failed",
			StepKey: "k2", ToolName: "assign_driver",
			Outcome: map[string]any{
				"verdict": "invalid",
				"content": `"driver_id" must be a string`,
			},
		},
		&agent.AgentRunStep{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			OwnerKind: string(
				agent.RunOwnerAgentRun,
			), OwnerID: run.ID, Kind: "Tool", Status: "Failed",
			StepKey: "k3", ToolName: "assign_driver",
			Outcome: map[string]any{
				"verdict": "failed",
				"cause":   "ToolTimedOut",
				"reason":  "deadline exceeded",
			},
		},
	)

	stamp := timeutils.NowUnixMilli()
	req := aianalyticsports.RefreshRequest{
		TenantInfo: tenant,
		From:       now - 3600,
		To:         timeutils.NowUnix() + 1,
		Stamp:      stamp,
	}
	units, err := repo.RefreshUnits(ctx, req)
	require.NoError(t, err)
	assert.Equal(
		t,
		4,
		units,
		"two runs, one turn and one standalone call; compaction and indexing are left out",
	)
	_, err = repo.RefreshProviderHours(ctx, req)
	require.NoError(t, err)

	keyReq := aianalyticsports.ModelKeyRequest{
		TenantInfo: tenant,
		Stamp:      stamp,
		FromHour:   floorHour(now - 3600),
	}
	models, err := repo.UnkeyedModels(ctx, keyReq)
	require.NoError(t, err)
	keys := make([]aianalyticsports.ModelKey, 0, len(models))
	for _, model := range models {
		keys = append(
			keys,
			aianalyticsports.ModelKey{Model: model, Key: aianalytics.CanonicalModel(model)},
		)
	}
	require.NoError(t, repo.SetModelKeys(ctx, keyReq, keys))

	filter := aianalyticsports.Filter{
		TenantInfo: tenant,
		Since:      now - 3600,
		CompareBy:  aianalytics.CompareByModel,
	}
	deployments, err := repo.Deployments(ctx, filter)
	require.NoError(t, err)
	byKey := make(map[string]aianalyticsports.Counts, len(deployments))
	for _, row := range deployments {
		counts := byKey[row.ModelKey]
		counts.Runs += row.Runs
		counts.Completed += row.Completed
		counts.HandedOff += row.HandedOff
		counts.Stopped += row.Stopped
		counts.ModelCalls += row.ModelCalls
		counts.FailedCalls += row.FailedCalls
		counts.ToolCalls += row.ToolCalls
		counts.ToolFailuresModel += row.ToolFailuresModel
		counts.ToolFailuresTool += row.ToolFailuresTool
		counts.CostUSD = counts.CostUSD.Add(row.CostUSD)
		byKey[row.ModelKey] = counts
	}
	require.Contains(t, byKey, "llama-3.3-70b")
	llama := byKey["llama-3.3-70b"]
	assert.Equal(t, 1, llama.Completed)
	assert.Equal(t, 2, llama.ModelCalls)
	assert.Equal(t, 1, llama.FailedCalls)
	assert.Equal(t, 3, llama.ToolCalls)
	assert.Equal(t, 1, llama.ToolFailuresModel)
	assert.Equal(t, 1, llama.ToolFailuresTool)
	assert.True(t, cost.Equal(llama.CostUSD))
	assert.Equal(
		t,
		1,
		byKey["claude-sonnet-5"].HandedOff,
		"an exception hands the run off before its limit",
	)
	assert.Equal(t, 1, byKey["gpt-5.1-mini"].Stopped)
	assert.Equal(t, 1, byKey["qwen-2.5-14b"].Completed)

	var served *aianalyticsports.DeploymentRow
	for idx := range deployments {
		if deployments[idx].ModelKey == "llama-3.3-70b" {
			served = &deployments[idx]
		}
	}
	require.NotNil(t, served)
	assert.Equal(t, groq.String(), served.ProviderID, "the run is the provider's that answered")
	assert.Equal(t, now-10, served.LastRanAt)

	tools, err := repo.ToolCells(ctx, filter)
	require.NoError(t, err)
	assert.Contains(t, tools, aianalyticsports.ToolCellRow{
		DeploymentKey: aianalyticsports.DeploymentKey{
			ModelKey:   "llama-3.3-70b",
			ProviderID: groq.String(),
		},
		ToolName: "assign_driver",
		Cause:    aianalytics.CauseWrongArguments,
		Calls:    1,
	}, "a tool row carries its unit's model and provider once the model is keyed")

	failures, err := repo.ToolFailures(ctx, aianalyticsports.ToolFailureRequest{
		Filter: filter, SeriesKey: "llama-3.3-70b", ToolName: "assign_driver", Limit: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, failures.Failed)
	assert.ElementsMatch(t, []aianalyticsports.CauseRow{
		{Cause: aianalytics.CauseWrongArguments, Calls: 1},
		{Cause: aianalytics.CauseToolTimedOut, Calls: 1},
	}, failures.Causes)
	require.Len(t, failures.Examples, 2)

	calls, err := repo.ProviderCalls(ctx, filter)
	require.NoError(t, err)
	var denied *aianalyticsports.ProviderCallRow
	for idx := range calls {
		if calls[idx].ProviderID == vllm.String() &&
			calls[idx].Error == aianalytics.ProviderErrorAccessDenied {
			denied = &calls[idx]
		}
	}
	require.NotNil(t, denied, "a 401 is access denied whatever class it was recorded under")
	assert.Equal(t, 1, denied.Calls)
	assert.Equal(t, 1, denied.FellToNext)

	require.NoError(
		t,
		repo.Prune(ctx, aianalyticsports.PruneRequest{TenantInfo: tenant, Before: now + 3600}),
	)
	pruned, err := repo.Deployments(ctx, filter)
	require.NoError(t, err)
	assert.Empty(t, pruned)
}

func TestRefreshCountsUncachedInputByTheCallsOwnProtocolWhenTheProviderIsGone(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seededTenant(ctx, t, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	now := timeutils.NowUnix()

	insert(ctx, t, db,
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			ProviderID: pulid.MustNew("aiprv_"), ProviderKind: aiprovider.KindAnthropicMessages,
			Model: "claude-haiku-5-5", Task: aiprovider.TaskDocumentClassification,
			Surface: aiusage.SurfaceStructured, Succeeded: true, LatencyMs: 300,
			InputTokens: 120, CacheReadTokens: 900, CacheWriteTokens: 60, OutputTokens: 40,
			CreatedAt: now - 5,
		},
		&aiusage.AIUsageRecord{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			ProviderID: pulid.MustNew("aiprv_"), ProviderKind: aiprovider.KindOpenAIResponses,
			Model: "gpt-6-luna", Task: aiprovider.TaskDocumentClassification,
			Surface: aiusage.SurfaceStructured, Succeeded: true, LatencyMs: 300,
			InputTokens: 1000, CacheReadTokens: 800, OutputTokens: 40,
			CreatedAt: now - 5,
		},
	)

	_, err := repo.RefreshUnits(ctx, aianalyticsports.RefreshRequest{
		TenantInfo: tenant,
		From:       now - 3600,
		To:         timeutils.NowUnix() + 1,
		Stamp:      timeutils.NowUnixMilli(),
	})
	require.NoError(t, err)

	var rows []struct {
		Model      string `bun:"model"`
		Input      int64  `bun:"input_tokens"`
		CacheRead  int64  `bun:"cache_read_tokens"`
		CacheWrite int64  `bun:"cache_write_tokens"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("ai_work_units").
		Column("model", "input_tokens", "cache_read_tokens", "cache_write_tokens").
		Where("organization_id = ? AND business_unit_id = ?", tenant.OrgID, tenant.BuID).
		Order("model").
		Scan(ctx, &rows))
	require.Len(t, rows, 2)

	assert.Equal(t, "claude-haiku-5-5", rows[0].Model)
	assert.Equal(t, int64(120), rows[0].Input,
		"Anthropic reports input without the cache, whether or not its provider row still exists")
	assert.Equal(t, int64(900), rows[0].CacheRead)
	assert.Equal(t, int64(60), rows[0].CacheWrite)

	assert.Equal(t, "gpt-6-luna", rows[1].Model)
	assert.Equal(t, int64(200), rows[1].Input, "OpenAI's input includes the cache read")
	assert.Equal(t, int64(800), rows[1].CacheRead)
}
