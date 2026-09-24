//go:build integration

package agentmemoryrepository

import (
	"context"
	"fmt"
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

type searchFixture struct {
	repo   repositories.AgentMemoryRepository
	tenant pagination.TenantInfo
	now    int64
}

func newSearchFixture(t *testing.T) (context.Context, searchFixture) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)

	return ctx, searchFixture{
		repo:   New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		tenant: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
		now:    timeutils.NowUnix(),
	}
}

func (f searchFixture) record(
	ctx context.Context,
	t *testing.T,
	memory *agent.Memory,
) *agent.Memory {
	t.Helper()

	memory.OrganizationID = f.tenant.OrgID
	memory.BusinessUnitID = f.tenant.BuID
	if memory.Kind == "" {
		memory.Kind = agent.MemoryKindFact
	}
	memory.Source = agent.MemorySourceUser
	memory.Status = agent.MemoryStatusActive
	created, err := f.repo.Create(ctx, memory)
	require.NoError(t, err)

	return created
}

func (f searchFixture) search(
	ctx context.Context,
	t *testing.T,
	query string,
) []pulid.ID {
	t.Helper()

	found, err := f.repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      query,
	})
	require.NoError(t, err)

	return memoryIDs(found)
}

func TestSearch_FindsAMemoryByItsWordsNotTheWholePhrase(t *testing.T) {
	ctx, f := newSearchFixture(t)
	customerID := pulid.MustNew("cus_")

	dock := f.record(ctx, t, &agent.Memory{
		Content: "The dock closes at four on Fridays; book the appointment before noon.",
		SubjectType: agent.MemorySubjectCustomer, SubjectID: &customerID,
		SubjectLabel: "Acme Foods",
	})
	hours := f.record(ctx, t, &agent.Memory{Content: "Office hours are nine to five."})
	f.record(ctx, t, &agent.Memory{Content: "Quote every lane in dollars."})

	assert.Equal(t, []pulid.ID{dock.ID}, f.search(ctx, t, "acme dock"),
		"the subject's name and the content are one set of words")
	assert.Equal(t, []pulid.ID{dock.ID, hours.ID}, f.search(ctx, t, "acme dock hours"),
		"no memory has every word, so the one sharing most of them ranks first")
	assert.Equal(t, []pulid.ID{dock.ID}, f.search(ctx, t, "appoint"),
		"a word finds the longer words it begins")
	assert.Empty(t, f.search(ctx, t, "detention"))
	assert.Empty(t, f.search(ctx, t, "?!"))
}

func TestSearch_HonoursWebsearchOperators(t *testing.T) {
	ctx, f := newSearchFixture(t)

	friday := f.record(ctx, t, &agent.Memory{Content: "Acme receives on Fridays only."})
	monday := f.record(ctx, t, &agent.Memory{Content: "Acme receives on Mondays as well."})

	assert.Equal(t, []pulid.ID{monday.ID}, f.search(ctx, t, "acme -fridays"))
	assert.Equal(t, []pulid.ID{friday.ID}, f.search(ctx, t, `"on fridays"`))
}

func TestSearch_ReadsToolNamesAndRanksBeforeKind(t *testing.T) {
	ctx, f := newSearchFixture(t)

	correction := f.record(ctx, t, &agent.Memory{
		Kind:     agent.MemoryKindCorrection,
		ToolName: "assign_move",
		Content:  "A person changed the tractor to the one already at the yard.",
	})
	instruction := f.record(ctx, t, &agent.Memory{
		Kind:    agent.MemoryKindInstruction,
		Content: "Mention the tractor number when you assign.",
	})

	found := f.search(ctx, t, "assign tractor")
	assert.ElementsMatch(t, []pulid.ID{correction.ID, instruction.ID}, found)
}

func TestSearch_ReadsOneMemoryByIDAndBoundsTheRows(t *testing.T) {
	ctx, f := newSearchFixture(t)

	var last *agent.Memory
	for idx := range agent.MaxMemoryRecallLimit + 5 {
		last = f.record(ctx, t, &agent.Memory{Content: fmt.Sprintf("Lane rule number %d.", idx)})
	}

	byID, err := f.repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		IDs:        []pulid.ID{last.ID},
	})
	require.NoError(t, err)
	require.Len(t, byID, 1)
	assert.Equal(t, last.ID, byID[0].ID)
	assert.Empty(t, byID[0].SearchVector, "the search column is never read back")

	defaulted, err := f.repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      "lane",
	})
	require.NoError(t, err)
	assert.Len(t, defaulted, agent.DefaultMemoryRecallLimit)

	capped, err := f.repo.Search(ctx, repositories.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      "lane",
		Limit:      1000,
	})
	require.NoError(t, err)
	assert.Len(t, capped, agent.MaxMemoryRecallLimit)
}

func TestCountActive_CountsOnlyWhatAgentsMayRead(t *testing.T) {
	ctx, f := newSearchFixture(t)

	f.record(ctx, t, &agent.Memory{Content: "Read one."})
	past := f.now - 60
	f.record(ctx, t, &agent.Memory{Content: "Expired.", ExpiresAt: &past})
	retired := f.record(ctx, t, &agent.Memory{Content: "Retired."})
	_, err := f.repo.SetStatus(ctx, repositories.SetAgentMemoryStatusRequest{
		ID:         retired.ID,
		TenantInfo: f.tenant,
		Status:     agent.MemoryStatusRetired,
		At:         f.now,
	})
	require.NoError(t, err)

	count, err := f.repo.CountActive(ctx, repositories.CountActiveAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
