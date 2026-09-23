//go:build integration

package agentmemoryrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func suggestedMemory(
	tenant pagination.TenantInfo,
	agentID pulid.ID,
	status agent.MemoryStatus,
	content string,
) *agent.Memory {
	return &agent.Memory{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Kind:              agent.MemoryKindCorrection,
		Source:            agent.MemorySourceFeedback,
		Status:            status,
		Content:           content,
		AgentDefinitionID: &agentID,
		Evidence: &agent.MemoryEvidence{
			FeedbackIDs: []pulid.ID{pulid.MustNew("aifb_")},
			PatternKey:  "pattern-" + string(status),
			RatingCount: 3,
		},
	}
}

func TestMemoryReaders_NeverReturnASuggestion(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	agentID := pulid.MustNew("agdef_")
	now := timeutils.NowUnix()

	active, err := repo.Create(ctx, &agent.Memory{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        "Check the lane before quoting",
	})
	require.NoError(t, err)

	suggested, err := repo.Create(ctx, suggestedMemory(tenant, agentID,
		agent.MemoryStatusSuggested, "Check the lane before quoting a rate"))
	require.NoError(t, err)

	_, err = repo.Create(ctx, suggestedMemory(tenant, agentID,
		agent.MemoryStatusDismissed, "Check the lane before quoting anything"))
	require.NoError(t, err)

	listed, err := repo.ListActive(ctx, repositories.ListActiveAgentMemoriesRequest{
		TenantInfo:       tenant,
		Now:              now,
		OrganizationWide: true,
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, active.ID, listed[0].ID)

	found, err := repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: tenant,
		Now:        now,
		Query:      "lane",
	})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, active.ID, found[0].ID)

	same, err := repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Content:    suggested.Content,
	})
	require.NoError(t, err)
	assert.Nil(t, same, "a pending suggestion is not an active memory")

	_, err = repo.SetStatus(ctx, repositories.SetAgentMemoryStatusRequest{
		ID:         suggested.ID,
		TenantInfo: tenant,
		Status:     agent.MemoryStatusActive,
		At:         now,
	})
	require.Error(t, err, "a suggestion is never activated by a status change")

	surrounding, err := repo.ListSuggestionContext(
		ctx,
		repositories.ListAgentMemorySuggestionContextRequest{
			TenantInfo:        tenant,
			AgentDefinitionID: agentID,
			Now:               now,
			DismissedSince:    now - 30*86400,
		},
	)
	require.NoError(t, err)
	assert.Len(t, surrounding, 3, "active, pending and recently dismissed memories all count")

	approved, err := repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:         suggested.ID,
		TenantInfo: tenant,
		Status:     agent.MemoryStatusActive,
		Kind:       agent.MemoryKindCorrection,
		Content:    "Confirm the lane before quoting a rate.",
		ByUserID:   data.User.ID,
		At:         now,
		Version:    suggested.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, approved.Status)
	assert.Equal(t, "Confirm the lane before quoting a rate.", approved.Content)

	_, err = repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:         suggested.ID,
		TenantInfo: tenant,
		Status:     agent.MemoryStatusDismissed,
		At:         now,
		Version:    approved.Version,
	})
	require.Error(t, err, "an approved memory is no longer a suggestion")
}
