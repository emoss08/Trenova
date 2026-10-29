//go:build integration

package agentmemoryrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
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
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	jordanUser := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("jordan").
		WithEmail("jordan@example.com").
		Build(t, ctx, db)
	avery, jordan := data.User.ID, jordanUser.ID
	billing, dispatch := pulid.MustNew("rol_"), pulid.MustNew("rol_")
	now := timeutils.NowUnix()

	keep := func(content string, scope agentmemory.MemoryScope, owner, role pulid.ID) *agentmemory.Memory {
		memory := &agentmemory.Memory{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Kind:           agentmemory.MemoryKindInstruction,
			Source:         agentmemory.MemorySourceUser,
			Status:         agentmemory.MemoryStatusActive,
			Content:        content,
		}
		memory.SetAudience(scope, owner, role)
		created, err := repo.Create(ctx, memory)
		require.NoError(t, err)

		return created
	}

	organization := keep(
		"Use the DOE weekly average.",
		agentmemory.MemoryScopeOrganization,
		pulid.Nil,
		pulid.Nil,
	)
	averys := keep("Group AR by facility.", agentmemory.MemoryScopeUser, avery, pulid.Nil)
	jordans := keep("Jordan's own note.", agentmemory.MemoryScopeUser, jordan, pulid.Nil)
	billings := keep("Acme is billed net-45.", agentmemory.MemoryScopeRole, pulid.Nil, billing)
	dispatchs := keep("Dispatch note.", agentmemory.MemoryScopeRole, pulid.Nil, dispatch)
	paused := keep(
		"Harbor wants PODs on every invoice.",
		agentmemory.MemoryScopeUser,
		avery,
		pulid.Nil,
	)
	_, err := repo.SetStatus(ctx, agentmemoryports.SetAgentMemoryStatusQuery{
		ID: paused.ID, TenantInfo: tenant, Status: agentmemory.MemoryStatusPaused, At: now,
	})
	require.NoError(t, err)

	// What Avery offered for the organization waits for an administrator: no
	// prompt reads it, but her Desk lists it as hers to withdraw. Jordan's offer
	// and an agent's suggestion are not hers.
	offer := func(content string, by pulid.ID, source agentmemory.MemorySource) *agentmemory.Memory {
		memory := &agentmemory.Memory{
			OrganizationID:  tenant.OrgID,
			BusinessUnitID:  tenant.BuID,
			Kind:            agentmemory.MemoryKindInstruction,
			Source:          source,
			Status:          agentmemory.MemoryStatusSuggested,
			Content:         content,
			CreatedByUserID: &by,
		}
		memory.SetAudience(agentmemory.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
		created, err := repo.Create(ctx, memory)
		require.NoError(t, err)

		return created
	}
	averysOffer := offer(
		"Granite wants the POD on the invoice.",
		avery,
		agentmemory.MemorySourceUser,
	)
	jordansOffer := offer("Jordan's offer.", jordan, agentmemory.MemorySourceUser)
	agentsOffer := offer("An agent's offer.", avery, agentmemory.MemorySourceAgent)

	reader := agentmemory.MemoryReader{UserID: avery, RoleIDs: []pulid.ID{billing}}
	listed, err := repo.ListActive(ctx, agentmemoryports.ListActiveAgentMemoriesRequest{
		TenantInfo:       tenant,
		Reader:           reader,
		Now:              now,
		OrganizationWide: true,
	})
	require.NoError(t, err)
	assert.ElementsMatch(
		t,
		[]pulid.ID{organization.ID, averys.ID, billings.ID},
		memoryIDs(listed),
		"Avery's prompt reads the organization's, her own and Billing's; not Jordan's, not Dispatch's, not the paused one",
	)

	nobody, err := repo.ListActive(ctx, agentmemoryports.ListActiveAgentMemoriesRequest{
		TenantInfo:       tenant,
		Now:              now,
		OrganizationWide: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{organization.ID}, memoryIDs(nobody),
		"a run nobody is in reads only the organization's")

	recalled, err := repo.Search(ctx, agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo: tenant, Reader: reader, Now: now, Query: "note",
	})
	require.NoError(t, err)
	assert.Empty(t, recalled, "recall finds neither Jordan's note nor Dispatch's")

	rows, err := repo.ListDesk(ctx, agentmemoryports.ListDeskMemoriesQuery{
		Filter: agentmemoryports.DeskMemoryFilter{TenantInfo: tenant, Reader: reader},
		Limit:  10,
	})
	require.NoError(t, err)
	listedIDs := make([]pulid.ID, 0, len(rows))
	for _, row := range rows {
		listedIDs = append(listedIDs, row.Memory.ID)
	}
	assert.ElementsMatch(t,
		[]pulid.ID{organization.ID, averys.ID, billings.ID, paused.ID, averysOffer.ID},
		listedIDs,
		"the Desk lists the paused one too, and what she offered for the organization")
	assert.NotContains(t, listedIDs, jordans.ID)
	assert.NotContains(t, listedIDs, dispatchs.ID)
	assert.NotContains(t, listedIDs, jordansOffer.ID, "another person's offer is theirs")
	assert.NotContains(
		t,
		listedIDs,
		agentsOffer.ID,
		"an agent's suggestion waits in its conversation",
	)

	counts, err := repo.CountDesk(
		ctx,
		agentmemoryports.DeskMemoryFilter{TenantInfo: tenant, Reader: reader},
	)
	require.NoError(t, err)
	byScope := map[agentmemory.MemoryScope]int{}
	for _, count := range counts {
		byScope[count.Scope] += count.Count
		if count.Scope == agentmemory.MemoryScopeRole {
			assert.Equal(t, billing, count.RoleID)
		}
	}
	assert.Equal(t, map[agentmemory.MemoryScope]int{
		agentmemory.MemoryScopeOrganization: 2,
		agentmemory.MemoryScopeUser:         2,
		agentmemory.MemoryScopeRole:         1,
	}, byScope, "her offer counts with the organization's")

	searched, err := repo.ListDesk(ctx, agentmemoryports.ListDeskMemoriesQuery{
		Filter: agentmemoryports.DeskMemoryFilter{
			TenantInfo: tenant,
			Reader:     reader,
			Query:      "facil",
		},
	})
	require.NoError(t, err)
	require.Len(t, searched, 1, "a word's start finds it")
	assert.Equal(t, averys.ID, searched[0].Memory.ID)
}

func TestDeskMemories_PageNewestFirstAndRevise(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	reader := agentmemory.MemoryReader{UserID: data.User.ID}

	for _, content := range []string{"One.", "Two.", "Three."} {
		memory := &agentmemory.Memory{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Kind:           agentmemory.MemoryKindFact,
			Source:         agentmemory.MemorySourceUser,
			Status:         agentmemory.MemoryStatusActive,
			Content:        content,
		}
		memory.SetAudience(agentmemory.MemoryScopeUser, data.User.ID, pulid.Nil)
		_, err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	filter := agentmemoryports.DeskMemoryFilter{TenantInfo: tenant, Reader: reader}
	first, err := repo.ListDesk(
		ctx,
		agentmemoryports.ListDeskMemoriesQuery{Filter: filter, Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, first, 2)
	last := first[1].Memory
	rest, err := repo.ListDesk(ctx, agentmemoryports.ListDeskMemoriesQuery{
		Filter: filter,
		After:  &agentmemoryports.DeskMemoryCursor{CreatedAt: last.CreatedAt, ID: last.ID},
		Limit:  2,
	})
	require.NoError(t, err)
	require.Len(t, rest, 1, "the cursor picks up after the page")
	assert.NotEqual(t, first[0].Memory.ID, rest[0].Memory.ID)

	revised, err := repo.Revise(ctx, agentmemoryports.ReviseAgentMemoryRequest{
		ID:         rest[0].Memory.ID,
		TenantInfo: tenant,
		Content:    "Now for everyone.",
		Scope:      agentmemory.MemoryScopeOrganization,
		Version:    rest[0].Memory.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, revised.Scope)
	assert.Nil(t, revised.OwnerUserID, "moving it clears whose it was")

	_, err = repo.Revise(ctx, agentmemoryports.ReviseAgentMemoryRequest{
		ID: revised.ID, TenantInfo: tenant, Content: "Stale.",
		Scope: agentmemory.MemoryScopeOrganization, Version: rest[0].Memory.Version,
	})
	require.Error(t, err, "an edit from an older version is refused")
}

func TestMemoryPreference_SavesOneRowPerPerson(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	req := agentmemoryports.GetAgentMemoryPreferenceRequest{
		TenantInfo: tenant,
		UserID:     data.User.ID,
	}

	none, err := repo.GetPreference(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, none)

	for _, mode := range []agentmemory.MemorySavingMode{agentmemory.MemorySavingAskFirst, agentmemory.MemorySavingAutomatic} {
		_, err = repo.SavePreference(ctx, &agentmemory.MemoryPreference{
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
	assert.Equal(t, agentmemory.MemorySavingAutomatic, saved.SavingMode, "the latest choice stands")
}
