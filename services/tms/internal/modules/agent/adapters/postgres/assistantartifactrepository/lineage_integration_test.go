//go:build integration

package assistantartifactrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/assistantartifactrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListPageReadsEveryVersionOfThePagesLineages(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	tenant := data.Info()
	repo := assistantartifactrepository.New(assistantartifactrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	thread := &conversation.Thread{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            data.User.ID,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Title:             "Lane review",
	}
	require.NoError(t, conversationdb.Threads.Create(ctx, db, thread))

	insert := func(artifact *assistantartifact.Artifact) *assistantartifact.Artifact {
		t.Helper()
		artifact.OrganizationID = tenant.OrgID
		artifact.BusinessUnitID = tenant.BuID
		artifact.ThreadID = thread.ID
		stored, err := repo.InsertVersion(ctx, artifact)
		require.NoError(t, err)

		return stored
	}
	report := insert(&assistantartifact.Artifact{
		Kind:             assistantartifact.KindReportPreview,
		Title:            "Late loads",
		Slug:             "late-loads",
		SourceToolCallID: "call_1",
		LineageSeq:       1,
		Payload:          map[string]any{"tool": "run_report", "rows": []any{"SHP-1"}},
	})
	revised := insert(&assistantartifact.Artifact{
		Kind:             assistantartifact.KindReportPreview,
		Title:            "Late loads",
		Slug:             "late-loads",
		SourceToolCallID: "call_2",
		LineageID:        report.ID,
		LineageSeq:       2,
		Payload:          map[string]any{"tool": "run_report", "rows": []any{"SHP-1", "SHP-2"}},
	})
	card := insert(&assistantartifact.Artifact{
		Kind:             assistantartifact.KindEntityCard,
		Title:            "SHP-1",
		Slug:             "shp-1",
		SourceToolCallID: "call_3",
		LineageSeq:       1,
		Pinned:           true,
		Payload:          map[string]any{"entity": "shipment", "fields": map[string]any{"pro": "1"}},
	})

	summary, err := repo.ListPage(ctx, assistantartifactports.ListArtifactsRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Summary:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.Counts.All)
	assert.Equal(t, 1, summary.Counts.Pinned)
	assert.Empty(t, summary.NextCursor)
	require.Len(t, summary.Artifacts, 3)
	byID := make(map[pulid.ID]*assistantartifact.Artifact, len(summary.Artifacts))
	for _, artifact := range summary.Artifacts {
		byID[artifact.ID] = artifact
	}
	assert.Equal(t, map[string]any{"tool": "run_report"}, byID[revised.ID].Payload)
	assert.Equal(t, map[string]any{"entity": "shipment"}, byID[card.ID].Payload)

	first, err := repo.ListPage(ctx, assistantartifactports.ListArtifactsRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Limit:      1,
	})
	require.NoError(t, err)
	require.Len(t, first.Artifacts, 1, "the pinned lineage comes first")
	assert.Equal(t, card.ID, first.Artifacts[0].ID)
	assert.Contains(t, first.Artifacts[0].Payload, "fields")
	require.NotEmpty(t, first.NextCursor)

	second, err := repo.ListPage(ctx, assistantartifactports.ListArtifactsRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Limit:      1,
		Cursor:     first.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, second.Artifacts, 2)
	assert.Equal(t, report.ID, second.Artifacts[0].ID)
	assert.Equal(t, revised.ID, second.Artifacts[1].ID)
	assert.Contains(t, second.Artifacts[1].Payload, "rows")
	assert.Empty(t, second.NextCursor)

	versions, err := repo.ListLineage(ctx, assistantartifactports.LineageRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		ID:         revised.ID,
	})
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, report.ID, versions[0].ID)
	assert.Equal(t, revised.ID, versions[1].ID)
}
