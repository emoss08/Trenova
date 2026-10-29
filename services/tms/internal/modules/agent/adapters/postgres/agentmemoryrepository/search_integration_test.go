//go:build integration

package agentmemoryrepository

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type searchFixture struct {
	repo   agentmemoryports.AgentMemoryRepository
	tenant pagination.TenantInfo
	now    int64
}

func newSearchFixture(t *testing.T) (context.Context, searchFixture) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)

	return ctx, searchFixture{
		repo:   New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		tenant: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
		now:    timeutils.NowUnix(),
	}
}

func (f searchFixture) record(
	ctx context.Context,
	t *testing.T,
	memory *agentmemory.Memory,
) *agentmemory.Memory {
	t.Helper()

	memory.OrganizationID = f.tenant.OrgID
	memory.BusinessUnitID = f.tenant.BuID
	if memory.Kind == "" {
		memory.Kind = agentmemory.MemoryKindFact
	}
	memory.Source = agentmemory.MemorySourceUser
	memory.Status = agentmemory.MemoryStatusActive
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

	terms := make([]string, 0, 4)
	for _, word := range stringutils.SearchWords(query) {
		if len(word) >= minTermRunes {
			terms = append(terms, word)
		}
	}
	found, err := f.repo.Search(ctx, agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      query,
		Terms:      terms,
	})
	require.NoError(t, err)

	return memoryIDs(found)
}

func TestSearch_FindsAMemoryByItsWordsNotTheWholePhrase(t *testing.T) {
	ctx, f := newSearchFixture(t)
	customerID := pulid.MustNew("cus_")

	dock := f.record(ctx, t, &agentmemory.Memory{
		Content:     "The dock closes at four on Fridays; book the appointment before noon.",
		SubjectType: agentmemory.MemorySubjectCustomer, SubjectID: &customerID,
		SubjectLabel: "Acme Foods",
	})
	hours := f.record(ctx, t, &agentmemory.Memory{Content: "Office hours are nine to five."})
	f.record(ctx, t, &agentmemory.Memory{Content: "Quote every lane in dollars."})

	assert.Equal(t, []pulid.ID{dock.ID}, f.search(ctx, t, "acme dock"),
		"the subject's name and the content are one set of words")
	assert.Equal(t, []pulid.ID{dock.ID}, f.search(ctx, t, "acme dock hours"),
		"no memory has every word, so the one sharing two of them is found")
	assert.Equal(t, []pulid.ID{hours.ID}, f.search(ctx, t, "office hours"))
	assert.Empty(t, f.search(ctx, t, "warehouse hours"),
		"a memory sharing a single word of a longer question is not found")
	assert.Equal(t, []pulid.ID{dock.ID}, f.search(ctx, t, "appoint"),
		"a word finds the longer words it begins")
	assert.Empty(t, f.search(ctx, t, "detention"))
	assert.Empty(t, f.search(ctx, t, "?!"))
}

func TestSearch_HonoursWebsearchOperators(t *testing.T) {
	ctx, f := newSearchFixture(t)

	friday := f.record(ctx, t, &agentmemory.Memory{Content: "Acme receives on Fridays only."})
	monday := f.record(ctx, t, &agentmemory.Memory{Content: "Acme receives on Mondays as well."})

	assert.Equal(t, []pulid.ID{monday.ID}, f.search(ctx, t, "acme -fridays"))
	assert.Equal(t, []pulid.ID{friday.ID}, f.search(ctx, t, `"on fridays"`))
}

func TestSearch_ReadsToolNamesAndRanksBeforeKind(t *testing.T) {
	ctx, f := newSearchFixture(t)

	correction := f.record(ctx, t, &agentmemory.Memory{
		Kind:     agentmemory.MemoryKindCorrection,
		ToolName: "dispatch_assign_move",
		Content:  "A person changed the tractor to the one already at the yard.",
	})
	instruction := f.record(ctx, t, &agentmemory.Memory{
		Kind:    agentmemory.MemoryKindInstruction,
		Content: "Mention the tractor number when you assign.",
	})

	found := f.search(ctx, t, "assign tractor")
	assert.ElementsMatch(t, []pulid.ID{correction.ID, instruction.ID}, found)
}

func TestSearch_ReadsOneMemoryByIDAndBoundsTheRows(t *testing.T) {
	ctx, f := newSearchFixture(t)

	var last *agentmemory.Memory
	for idx := range agentmemory.MaxMemoryRecallLimit + 5 {
		last = f.record(
			ctx,
			t,
			&agentmemory.Memory{Content: fmt.Sprintf("Lane rule number %d.", idx)},
		)
	}

	byID, err := f.repo.Search(ctx, agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		IDs:        []pulid.ID{last.ID},
	})
	require.NoError(t, err)
	require.Len(t, byID, 1)
	assert.Equal(t, last.ID, byID[0].ID)
	assert.Empty(t, byID[0].SearchVector, "the search column is never read back")

	defaulted, err := f.repo.Search(ctx, agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      "lane",
	})
	require.NoError(t, err)
	assert.Len(t, defaulted, agentmemory.DefaultMemoryRecallLimit)

	capped, err := f.repo.Search(ctx, agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
		Query:      "lane",
		Limit:      1000,
	})
	require.NoError(t, err)
	assert.Len(t, capped, agentmemory.MaxMemoryRecallLimit)
}

func TestCountActive_CountsOnlyWhatAgentsMayRead(t *testing.T) {
	ctx, f := newSearchFixture(t)

	f.record(ctx, t, &agentmemory.Memory{Content: "Read one."})
	past := f.now - 60
	f.record(ctx, t, &agentmemory.Memory{Content: "Expired.", ExpiresAt: &past})
	retired := f.record(ctx, t, &agentmemory.Memory{Content: "Retired."})
	_, err := f.repo.SetStatus(ctx, agentmemoryports.SetAgentMemoryStatusQuery{
		ID:         retired.ID,
		TenantInfo: f.tenant,
		Status:     agentmemory.MemoryStatusRetired,
		At:         f.now,
	})
	require.NoError(t, err)

	count, err := f.repo.CountActive(ctx, agentmemoryports.CountActiveAgentMemoriesRequest{
		TenantInfo: f.tenant,
		Now:        f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
