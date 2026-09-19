package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWorkerRepo records the request a tool built and returns what it is told.
// It embeds the interface so the methods no tool calls stay unimplemented and
// a future addition to WorkerRepository does not break this file.
type fakeWorkerRepo struct {
	repositories.WorkerRepository

	captured *repositories.ListWorkersRequest
	items    []*worker.Worker
	err      error
}

func (f *fakeWorkerRepo) List(
	_ context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	f.captured = req
	if f.err != nil {
		return nil, f.err
	}

	return &pagination.CursorListResult[*worker.Worker]{Items: f.items}, nil
}

func testParams(params map[string]any) serviceports.QueryToolParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return serviceports.QueryToolParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Params: params,
	}
}

/*
A model that omits an optional argument is not making a mistake, and a tool that
treats the omission as one turns a question into a dead end. This happened in
production: asked which drivers had a medical card expiring soon, the model
called this tool with no query, was told the parameter "must be a non-empty
string", searched for the literal word "driver" instead, matched nobody, and
told the customer their system had no drivers in it.

The repository never needed the term. ApplyCursorFilters only applies a text
search when Query is non-empty (pkg/querybuilder/cursor_sort.go), so an empty
one already means "everything, newest first" — the tool was rejecting a call the
layer beneath it would have served.
*/
func TestSearchWorker_ListsWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{{ID: pulid.MustNew("wrk_")}}}
	tool := newSearchWorkerTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured, "the repository should have been asked")

	assert.Empty(t, repo.captured.Filter.Query, "an absent query filters on nothing")
	assert.NotNil(t, result)
}

func TestSearchWorker_PassesAQueryThrough(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	tool := newSearchWorkerTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"query": "  Ortiz  "}))
	require.NoError(t, err)

	assert.Equal(t, "Ortiz", repo.captured.Filter.Query, "a query is trimmed, not dropped")
}

/*
An empty list and "there is no such data" are different answers, and the model
cannot tell them apart from a bare []. It said "there are no driver records
currently in the Trenova system" — a confident claim about the customer's
database, from a search that matched nothing.
*/
func TestSearchWorker_EmptyResultSaysWhatItSearched(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: nil}
	tool := newSearchWorkerTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "Ortiz"}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok, "an empty result must carry its terms, not be a bare slice")
	assert.Zero(t, outcome.Count)
	assert.Contains(t, strings.Join(outcome.SearchedFor, " "), "Ortiz")
	assert.NotEmpty(t, outcome.Note, "an empty result explains itself")
}

func TestSearchWorker_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	tool := newSearchWorkerTool(repo)

	params := testParams(map[string]any{})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, repo.captured, "a mismatched actor never reaches the repository")
}

func TestSearchWorker_IsNamedForOneWorker(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "search_worker", newSearchWorkerTool(&fakeWorkerRepo{}).Name())
}

/*
The tool name is persisted in agent_definitions.tool_names and as a key in
tool_tiers, so a rename is a data change, not a symbol change. The migration
rewrites both columns, but a definition written by an API client that hard-coded
the old name would otherwise lose the tool silently — which is worse than an
error, because the agent simply stops being able to look drivers up.
*/
func TestRegistry_ResolvesTheLegacyWorkerSearchName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []serviceports.AgentQueryTool{newSearchWorkerTool(&fakeWorkerRepo{})},
	})

	tool, ok := reg.Get("search_workers")
	require.True(t, ok, "an agent saved before the rename still has its tool")
	assert.Equal(t, "search_worker", tool.Name())
}

func TestRegistry_DoesNotAdvertiseTheLegacyName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []serviceports.AgentQueryTool{newSearchWorkerTool(&fakeWorkerRepo{})},
	})

	for _, descriptor := range reg.Descriptors() {
		assert.NotEqual(t, "search_workers", descriptor.Name,
			"the alias is for reading old definitions, not for the model to call")
	}
}
