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

func memoryIDs(memories []*agent.Memory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}

	return ids
}

func TestMemoryReaders_ScopeAMemoryKeptForOneAgentToThatAgent(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	rateDesk, dispatch := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	now := timeutils.NowUnix()

	organization, err := repo.Create(ctx, &agent.Memory{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        "Quote in the customer's currency",
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, organization.Scope)

	approved := suggestedMemory(tenant, rateDesk, agent.MemoryStatusActive,
		"State only rates a tool returned")
	approved.Scope = agent.MemoryScopeAgent
	forRateDesk, err := repo.Create(ctx, approved)
	require.NoError(t, err)

	written, err := repo.Create(ctx, &agent.Memory{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Kind:              agent.MemoryKindFact,
		Source:            agent.MemorySourceAgent,
		Status:            agent.MemoryStatusActive,
		Content:           "Acme closes at four on Fridays",
		AgentDefinitionID: &dispatch,
		Tainted:           true,
	})
	require.NoError(t, err)

	read := func(agentID pulid.ID) []pulid.ID {
		listed, listErr := repo.ListActive(ctx, repositories.ListActiveAgentMemoriesRequest{
			TenantInfo:        tenant,
			AgentDefinitionID: agentID,
			Now:               now,
			OrganizationWide:  true,
		})
		require.NoError(t, listErr)

		return memoryIDs(listed)
	}

	assert.ElementsMatch(t, []pulid.ID{organization.ID, forRateDesk.ID, written.ID},
		read(rateDesk), "the rate desk reads its own and the organization's")
	assert.ElementsMatch(t, []pulid.ID{organization.ID, written.ID}, read(dispatch),
		"another agent never reads what was kept for the rate desk; an agent's own "+
			"memory reaches every agent as it always did")
	assert.ElementsMatch(t, []pulid.ID{organization.ID, written.ID}, read(pulid.Nil),
		"a prompt for no agent in particular reads only the organization's")

	recalled, err := repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo:        tenant,
		AgentDefinitionID: dispatch,
		Now:               now,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{organization.ID, written.ID}, memoryIDs(recalled))

	stored, err := repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         written.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.True(t, stored.Tainted, "a memory keeps the taint of the run that wrote it")
}

func TestResolveSuggestion_ApprovesForTheScopeAsked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	agentID := pulid.MustNew("agdef_")

	for _, scope := range []agent.MemoryScope{"", agent.MemoryScopeOrganization} {
		created, err := repo.Create(ctx, suggestedMemory(tenant, agentID,
			agent.MemoryStatusSuggested, "Say which rate table a quote came from "+string(scope)))
		require.NoError(t, err)

		resolved, err := repo.ResolveSuggestion(ctx,
			repositories.ResolveAgentMemorySuggestionRequest{
				ID:         created.ID,
				TenantInfo: tenant,
				Status:     agent.MemoryStatusActive,
				Kind:       created.Kind,
				Content:    created.Content,
				Scope:      scope,
				At:         timeutils.NowUnix(),
				Version:    created.Version,
			})
		require.NoError(t, err)

		want := scope
		if want == "" {
			want = agent.MemoryScopeAgent
		}
		assert.Equal(t, want, resolved.Scope)
	}
}
