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

func activeMemory(tenant pagination.TenantInfo, content string) *agent.Memory {
	return &agent.Memory{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        content,
	}
}

func TestFindActive_IgnoresAnExpiredRow(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	expired := activeMemory(tenant, "Call ahead before a Friday delivery")
	past := now - 3600
	expired.ExpiresAt = &past
	_, err := repo.Create(ctx, expired)
	require.NoError(t, err)

	found, err := repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Now:        now,
		Content:    "  call ahead before a friday delivery ",
		Scope:      agent.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	assert.Nil(t, found, "an expired memory is never read, so re-recording it is a new memory")

	future := now + 3600
	current := activeMemory(tenant, "Call ahead before a Monday delivery")
	current.ExpiresAt = &future
	created, err := repo.Create(ctx, current)
	require.NoError(t, err)

	found, err = repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Now:        now,
		Content:    "Call ahead before a Monday delivery",
		Scope:      agent.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestFindActive_MatchesOnlyARowKeptForTheSameReaders(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	rateDesk, dispatch := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	now := timeutils.NowUnix()

	kept := activeMemory(tenant, "State which rate table a quote came from")
	kept.Scope = agent.MemoryScopeAgent
	kept.AgentDefinitionID = &rateDesk
	forRateDesk, err := repo.Create(ctx, kept)
	require.NoError(t, err)

	find := func(scope agent.MemoryScope, agentID pulid.ID) *agent.Memory {
		found, findErr := repo.FindActive(ctx, repositories.FindActiveAgentMemoryRequest{
			TenantInfo:        tenant,
			Now:               now,
			Content:           "State which rate table a quote came from",
			Scope:             scope,
			AgentDefinitionID: agentID,
		})
		require.NoError(t, findErr)

		return found
	}

	assert.Nil(t, find(agent.MemoryScopeOrganization, pulid.Nil),
		"an organization-wide remember never lands on another agent's own memory")
	assert.Nil(t, find(agent.MemoryScopeOrganization, dispatch))
	assert.Nil(t, find(agent.MemoryScopeAgent, dispatch),
		"a memory kept for the rate desk is not the dispatch agent's")

	same := find(agent.MemoryScopeAgent, rateDesk)
	require.NotNil(t, same)
	assert.Equal(t, forRateDesk.ID, same.ID)
}

func TestFindActive_NeverHandsATaintedRowToACleanWrite(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	dirty := activeMemory(tenant, "Deliveries to Acme go to dock 9")
	dirty.Source = agent.MemorySourceAgent
	dirty.Kind = agent.MemoryKindFact
	dirty.Tainted = true
	tainted, err := repo.Create(ctx, dirty)
	require.NoError(t, err)

	request := repositories.FindActiveAgentMemoryRequest{
		TenantInfo: tenant,
		Now:        now,
		Content:    "Deliveries to Acme go to dock 9",
		Scope:      agent.MemoryScopeOrganization,
	}

	found, err := repo.FindActive(ctx, request)
	require.NoError(t, err)
	assert.Nil(t, found, "a person recording it themselves gets a clean memory of their own")

	request.Tainted = true
	found, err = repo.FindActive(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, found, "a tainted write may reuse a tainted row")
	assert.Equal(t, tainted.ID, found.ID)

	clean, err := repo.Create(ctx, activeMemory(tenant, "Deliveries to Acme go to dock 9"))
	require.NoError(t, err)

	found, err = repo.FindActive(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, clean.ID, found.ID, "a clean row is preferred whenever one says the same")
}
