package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWorkerRepo records the request a tool built and returns what it is told.
// It embeds the interface so the methods no tool calls stay unimplemented and
// a future addition to WorkerRepository does not break this file.
type fakeWorkerRepo struct {
	workerports.WorkerRepository

	captured *workerports.ListWorkersRequest
	items    []*worker.Worker
	err      error
}

func (f *fakeWorkerRepo) List(
	_ context.Context,
	req *workerports.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	f.captured = req
	if f.err != nil {
		return nil, f.err
	}

	return &pagination.CursorListResult[*worker.Worker]{Items: f.items}, nil
}

func testParams(params map[string]any) *agenttoolports.QueryToolParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return &agenttoolports.QueryToolParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
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
called the name search with no query, was told the parameter "must be a
non-empty string", searched for the literal word "driver" instead, matched
nobody, and told the customer their system had no drivers in it.

The repository never needed the term. listquery only applies a text
search when Query is non-empty (pkg/listquery/search.go), so an empty
one already means "everything, newest first".
*/
func TestListWorkers_ListsTheRosterWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{{ID: pulid.MustNew("wrk_")}}}
	tool := newListWorkersTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured, "the repository should have been asked")

	assert.Empty(t, repo.captured.Filter.Query, "an absent query filters on nothing")
	rows, _ := listedRows[workerRow](t, result)
	assert.Len(t, rows, 1)
}

func TestListWorkers_PassesANameQueryThrough(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{namedWorker("Luis", "Ortiz")}}
	tool := newListWorkersTool(repo, &fakePermissions{})

	_, err := tool.Query(t.Context(), testParams(map[string]any{"query": "  Ortiz  "}))
	require.NoError(t, err)

	assert.Equal(t, "Ortiz", repo.captured.Filter.Query, "a query is trimmed, not dropped")
}

func TestListWorkers_QueryMatchesFirstAndLastName(t *testing.T) {
	t.Parallel()

	config := (&worker.Worker{}).GetPostgresSearchConfig()
	names := make([]string, 0, len(config.SearchableFields))
	for _, field := range config.SearchableFields {
		names = append(names, field.Name)
	}

	assert.True(t, config.UseSearchVector)
	assert.Contains(t, names, "first_name")
	assert.Contains(t, names, "last_name")
}

func namedWorker(first, last string) *worker.Worker {
	return &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: first, LastName: last}
}

func TestListWorkers_PutsTheClosestNameFirst(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		query string
		items []*worker.Worker
		first string
	}{
		{
			query: "Maria Ortiz",
			items: []*worker.Worker{
				namedWorker("Mark", "Ortega"),
				namedWorker("Luis", "Ortiz"),
				namedWorker("Maria", "Ortiz"),
			},
			first: "Maria Ortiz",
		},
		{
			query: "Ortiz",
			items: []*worker.Worker{
				namedWorker("Sam", "Ortega"),
				namedWorker("Luis", "Ortiz"),
			},
			first: "Luis Ortiz",
		},
		{
			query: "ortiz maria",
			items: []*worker.Worker{
				namedWorker("Mario", "Ortino"),
				namedWorker("Maria", "Ortiz"),
			},
			first: "Maria Ortiz",
		},
	} {
		tool := newListWorkersTool(&fakeWorkerRepo{items: tc.items}, &fakePermissions{})

		result, err := tool.Query(t.Context(), testParams(map[string]any{"query": tc.query}))
		require.NoError(t, err)

		rows, _ := listedRows[workerRow](t, result)
		require.Len(t, rows, len(tc.items), tc.query)
		assert.Equal(t, tc.first, rows[0].Name, tc.query)
	}
}

func TestListWorkers_KeepsTheRequestedSortOverTheNameRank(t *testing.T) {
	t.Parallel()

	items := []*worker.Worker{namedWorker("Mark", "Ortega"), namedWorker("Maria", "Ortiz")}
	tool := newListWorkersTool(&fakeWorkerRepo{items: items}, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query":  "Maria Ortiz",
		"sortBy": "lastName",
	}))
	require.NoError(t, err)

	rows, _ := listedRows[workerRow](t, result)
	require.Len(t, rows, 2)
	assert.Equal(t, "Mark Ortega", rows[0].Name, "an asked-for order is the repository's")
}

func TestListWorkers_DescribesTheNameSearchAndTheCredentialList(t *testing.T) {
	t.Parallel()

	tool := newListWorkersTool(&fakeWorkerRepo{}, &fakePermissions{})

	assert.Contains(t, tool.Description(), "by name")
	assert.Contains(t, tool.Description(), "worker_list_credentials")
	assert.Contains(t, tool.Description(), "none on file")
	assert.Contains(t, tool.(*listTool).SearchTerms(), "look up a driver by name")
}

/*
An empty list and "there is no such data" are different answers, and the model
cannot tell them apart from a bare []. It said "there are no driver records
currently in the Trenova system" — a confident claim about the customer's
database, from a search that matched nothing.
*/
func TestListWorkers_EmptyNameSearchSaysWhatItSearched(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: nil}
	tool := newListWorkersTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "Ortiz"}))
	require.NoError(t, err)

	gated, ok := result.(*gatedOutcome)
	require.True(t, ok, "an empty result must carry its terms, not be a bare slice")
	outcome := gated.searchOutcome
	assert.Zero(t, outcome.Count)
	assert.Contains(t, strings.Join(outcome.SearchedFor, " "), "Ortiz")
	assert.NotEmpty(t, outcome.Note, "an empty result explains itself")
}

func TestListWorkers_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	tool := newListWorkersTool(repo, &fakePermissions{})

	params := testParams(map[string]any{})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, repo.captured, "a mismatched actor never reaches the repository")
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
		Tools: []agenttoolports.AgentQueryTool{
			newListWorkersTool(&fakeWorkerRepo{}, &fakePermissions{}),
		},
	})

	tool, ok := reg.Get("search_workers")
	require.True(t, ok, "an agent saved before the rename still has its tool")
	assert.Equal(t, "worker_list", tool.Name())
}

func TestRegistry_DoesNotAdvertiseTheLegacyName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []agenttoolports.AgentQueryTool{
			newListWorkersTool(&fakeWorkerRepo{}, &fakePermissions{}),
		},
	})

	for _, descriptor := range reg.Descriptors() {
		assert.NotEqual(t, "search_workers", descriptor.Name,
			"the alias is for reading old definitions, not for the model to call")
	}
}

/*
A name search returns the same curated row the roster does, not the stored
entity.

It used to return the entity. For a nearly empty worker that was 1,398 bytes
against 231 for the row — six times the context for the same answer, most of
it nulls nobody asked about: businessUnitId, stateId, profilePicUrl,
addressLine2, externalId, gender. Eight drivers overran the 12,000-character
tool-result cap, the result was cut mid-record, and the model announced it
would "see the remaining two workers from the truncated data" and invented
them.

The entity also reported an unrecorded credential as a bare null, which is the
reading that produced a wrong compliance answer.
*/
func TestListWorkers_ReturnsTheCuratedRowNotTheStoredEntity(t *testing.T) {
	t.Parallel()

	expiry := int64(1791591001)
	repo := &fakeWorkerRepo{items: []*worker.Worker{{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Mike",
		LastName:  "Johnson",
		Profile:   &worker.WorkerProfile{MedicalCardExpiry: &expiry},
	}}}

	result, err := newListWorkersTool(
		repo,
		&fakePermissions{},
	).Query(t.Context(), testParams(map[string]any{"query": "Mike"}))
	require.NoError(t, err)

	rows, _ := listedRows[workerRow](t, result)
	require.Len(t, rows, 1)

	assert.Equal(t, "Mike Johnson", rows[0].Name)

	encoded, err := sonic.Marshal(rows)
	require.NoError(t, err)
	for _, noise := range []string{"profilePicUrl", "addressLine2", "externalId", "gender"} {
		assert.NotContains(t, string(encoded), noise,
			"the entity's %s costs context and answers nothing", noise)
	}
}

// The absence semantics the rows carry have to reach a name search too, or
// the path the incident actually used is still the one that reports a missing
// certificate as nothing at all.
func TestListWorkers_SaysWhenACredentialIsNotOnFile(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{{
		ID: pulid.MustNew("wrk_"), FirstName: "David", LastName: "Park",
		Profile: &worker.WorkerProfile{},
	}}}

	result, err := newListWorkersTool(
		repo,
		&fakePermissions{},
	).Query(t.Context(), testParams(map[string]any{"query": "Park"}))
	require.NoError(t, err)

	rows, _ := listedRows[workerRow](t, result)
	encoded, err := sonic.Marshal(rows)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), "none on file")
	assert.NotContains(t, string(encoded), "null")
}

/*
A NonCompliant driver is not dispatchable, and the row has to say so.

workers.can_be_assigned means "no employment-level block" — no termination, no
leave, no manual hold. The domain's own dispatch decision (workercompliance.Standing)
additionally refuses a worker whose credentials are NonCompliant. The row
reported the column verbatim, so a driver whose medical card had lapsed four
days earlier came back as canBeAssigned: true, alongside
complianceStatus: NonCompliant. Both were in the same object, contradicting
each other, and the reader believed the friendlier one.
*/
func TestWorkerRow_DoesNotCallANonCompliantDriverAssignable(t *testing.T) {
	t.Parallel()

	lapsed := int64(1789516800)
	row := toWorkerRow(&worker.Worker{
		ID:            pulid.MustNew("wrk_"),
		FirstName:     "John",
		LastName:      "Smith",
		CanBeAssigned: true,
		Profile: &worker.WorkerProfile{
			ComplianceStatus:  worker.ComplianceStatusNonCompliant,
			MedicalCardExpiry: &lapsed,
		},
	})

	assert.False(t, row.CanBeAssigned,
		"the stored flag does not know about credentials; the domain does")
	assert.NotEmpty(t, row.AssignmentBlocked, "and it has to say why")
}

// A compliant driver keeps whatever the employment flag said, in both
// directions — this narrows the answer, it never widens it.
func TestWorkerRow_LeavesACompliantDriverAlone(t *testing.T) {
	t.Parallel()

	assignable := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: true,
		Profile: &worker.WorkerProfile{ComplianceStatus: worker.ComplianceStatusCompliant},
	})
	assert.True(t, assignable.CanBeAssigned)

	held := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: false,
		Profile: &worker.WorkerProfile{ComplianceStatus: worker.ComplianceStatusCompliant},
	})
	assert.False(t, held.CanBeAssigned)
}

// An existing employment reason is more specific than the compliance one and
// must not be overwritten by it.
func TestWorkerRow_KeepsAMoreSpecificBlockReason(t *testing.T) {
	t.Parallel()

	row := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: false,
		AssignmentBlocked: "On unpaid leave",
		Profile: &worker.WorkerProfile{
			ComplianceStatus: worker.ComplianceStatusNonCompliant,
		},
	})

	assert.Equal(t, "On unpaid leave", row.AssignmentBlocked)
}

func (f *fakeWorkerRepo) GetByID(
	_ context.Context,
	_ workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.items[0], nil
}

type pagedWorkerRepo struct {
	workerports.WorkerRepository

	captured *workerports.ListWorkersRequest
	items    []*worker.Worker
}

func (f *pagedWorkerRepo) List(
	_ context.Context,
	req *workerports.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	f.captured = req

	return pageAfter(f.items, req.Cursor), nil
}

func TestListWorkers_PagesByCursor(t *testing.T) {
	t.Parallel()

	items := make([]*worker.Worker, 0, 7)
	for range 7 {
		items = append(items, namedWorker("Luis", "Ortiz"))
	}
	repo := &pagedWorkerRepo{items: items}
	tool := newListWorkersTool(repo, &fakePermissions{})
	params := testParams(nil)

	assertCursorPaging(t, cursorPagingCase{
		query: func(args map[string]any) (any, error) {
			params.Params = args
			return tool.Query(t.Context(), params)
		},
		received: func() pagination.CursorInfo { return repo.captured.Cursor },
		args:     map[string]any{"query": "Ortiz"},
		changed:  map[string]any{"query": "Ortega"},
		ids:      cursorIDs(items),
		limit:    3,
	})
	assert.Zero(t, repo.captured.Filter.Pagination.Offset, "a cursor list is never offset")
}
