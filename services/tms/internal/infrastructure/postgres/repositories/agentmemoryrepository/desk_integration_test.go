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

/*
A memory kept for one person reaches only that person's prompts, one kept for
a role only its holders', and a paused one nobody's. The Desk lists the same
memories the prompt would read for the person, paused ones too, and counts
them per scope in one read.
*/
func TestDeskMemories_ReachOnlyTheirReadersAndPauseReachesNobody(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	avery, jordan := data.User.ID, pulid.MustNew("usr_")
	billing, dispatch := pulid.MustNew("rol_"), pulid.MustNew("rol_")
	now := timeutils.NowUnix()

	keep := func(content string, scope agent.MemoryScope, owner, role pulid.ID) *agent.Memory {
		memory := &agent.Memory{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Kind:           agent.MemoryKindInstruction,
			Source:         agent.MemorySourceUser,
			Status:         agent.MemoryStatusActive,
			Content:        content,
		}
		memory.SetAudience(scope, owner, role)
		created, err := repo.Create(ctx, memory)
		require.NoError(t, err)

		return created
	}

	organization := keep("Use the DOE weekly average.", agent.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	averys := keep("Group AR by facility.", agent.MemoryScopeUser, avery, pulid.Nil)
	jordans := keep("Jordan's own note.", agent.MemoryScopeUser, jordan, pulid.Nil)
	billings := keep("Acme is billed net-45.", agent.MemoryScopeRole, pulid.Nil, billing)
	dispatchs := keep("Dispatch note.", agent.MemoryScopeRole, pulid.Nil, dispatch)
	paused := keep("Harbor wants PODs on every invoice.", agent.MemoryScopeUser, avery, pulid.Nil)
	_, err := repo.SetStatus(ctx, repositories.SetAgentMemoryStatusRequest{
		ID: paused.ID, TenantInfo: tenant, Status: agent.MemoryStatusPaused, At: now,
	})
	require.NoError(t, err)

	reader := agent.MemoryReader{UserID: avery, RoleIDs: []pulid.ID{billing}}
	listed, err := repo.ListActive(ctx, repositories.ListActiveAgentMemoriesRequest{
		TenantInfo:       tenant,
		Reader:           reader,
		Now:              now,
		OrganizationWide: true,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{organization.ID, averys.ID, billings.ID}, memoryIDs(listed),
		"Avery's prompt reads the organization's, her own and Billing's; not Jordan's, not Dispatch's, not the paused one")

	nobody, err := repo.ListActive(ctx, repositories.ListActiveAgentMemoriesRequest{
		TenantInfo:       tenant,
		Now:              now,
		OrganizationWide: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{organization.ID}, memoryIDs(nobody),
		"a run nobody is in reads only the organization's")

	recalled, err := repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: tenant, Reader: reader, Now: now, Query: "note",
	})
	require.NoError(t, err)
	assert.Empty(t, recalled, "recall finds neither Jordan's note nor Dispatch's")

	rows, err := repo.ListDesk(ctx, repositories.ListDeskMemoriesRequest{
		Filter: repositories.DeskMemoryFilter{TenantInfo: tenant, Reader: reader},
		Limit:  10,
	})
	require.NoError(t, err)
	listedIDs := make([]pulid.ID, 0, len(rows))
	for _, row := range rows {
		listedIDs = append(listedIDs, row.Memory.ID)
	}
	assert.ElementsMatch(t, []pulid.ID{organization.ID, averys.ID, billings.ID, paused.ID}, listedIDs,
		"the Desk lists the paused one too")
	assert.NotContains(t, listedIDs, jordans.ID)
	assert.NotContains(t, listedIDs, dispatchs.ID)

	counts, err := repo.CountDesk(ctx, repositories.DeskMemoryFilter{TenantInfo: tenant, Reader: reader})
	require.NoError(t, err)
	byScope := map[agent.MemoryScope]int{}
	for _, count := range counts {
		byScope[count.Scope] += count.Count
		if count.Scope == agent.MemoryScopeRole {
			assert.Equal(t, billing, count.RoleID)
		}
	}
	assert.Equal(t, map[agent.MemoryScope]int{
		agent.MemoryScopeOrganization: 1,
		agent.MemoryScopeUser:         2,
		agent.MemoryScopeRole:         1,
	}, byScope)

	searched, err := repo.ListDesk(ctx, repositories.ListDeskMemoriesRequest{
		Filter: repositories.DeskMemoryFilter{TenantInfo: tenant, Reader: reader, Query: "facil"},
	})
	require.NoError(t, err)
	require.Len(t, searched, 1, "a word's start finds it")
	assert.Equal(t, averys.ID, searched[0].Memory.ID)
}

func TestDeskMemories_PageNewestFirstAndRevise(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	reader := agent.MemoryReader{UserID: data.User.ID}

	for _, content := range []string{"One.", "Two.", "Three."} {
		memory := &agent.Memory{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Kind:           agent.MemoryKindFact,
			Source:         agent.MemorySourceUser,
			Status:         agent.MemoryStatusActive,
			Content:        content,
		}
		memory.SetAudience(agent.MemoryScopeUser, data.User.ID, pulid.Nil)
		_, err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	filter := repositories.DeskMemoryFilter{TenantInfo: tenant, Reader: reader}
	first, err := repo.ListDesk(ctx, repositories.ListDeskMemoriesRequest{Filter: filter, Limit: 2})
	require.NoError(t, err)
	require.Len(t, first, 2)
	last := first[1].Memory
	rest, err := repo.ListDesk(ctx, repositories.ListDeskMemoriesRequest{
		Filter: filter,
		After:  &repositories.DeskMemoryCursor{CreatedAt: last.CreatedAt, ID: last.ID},
		Limit:  2,
	})
	require.NoError(t, err)
	require.Len(t, rest, 1, "the cursor picks up after the page")
	assert.NotEqual(t, first[0].Memory.ID, rest[0].Memory.ID)

	revised, err := repo.Revise(ctx, repositories.ReviseAgentMemoryRequest{
		ID:         rest[0].Memory.ID,
		TenantInfo: tenant,
		Content:    "Now for everyone.",
		Scope:      agent.MemoryScopeOrganization,
		Version:    rest[0].Memory.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, revised.Scope)
	assert.Nil(t, revised.OwnerUserID, "moving it clears whose it was")

	_, err = repo.Revise(ctx, repositories.ReviseAgentMemoryRequest{
		ID: revised.ID, TenantInfo: tenant, Content: "Stale.",
		Scope: agent.MemoryScopeOrganization, Version: rest[0].Memory.Version,
	})
	require.Error(t, err, "an edit from an older version is refused")
}

func TestMemoryPreference_SavesOneRowPerPerson(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	req := repositories.GetAgentMemoryPreferenceRequest{TenantInfo: tenant, UserID: data.User.ID}

	none, err := repo.GetPreference(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, none)

	for _, mode := range []agent.MemorySavingMode{agent.MemorySavingAskFirst, agent.MemorySavingAutomatic} {
		_, err = repo.SavePreference(ctx, &agent.MemoryPreference{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			UserID:         data.User.ID,
			SavingMode:     mode,
		})
		require.NoError(t, err)
	}

	saved, err := repo.GetPreference(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, agent.MemorySavingAutomatic, saved.SavingMode, "the latest choice stands")
}
