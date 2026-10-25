package agentquerytoolservice_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/pagetest"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// manufacturerRows serves both paths a tool has had: the repository list the
// old tool paged by offset, and the connection the GraphQL resolver pages by
// cursor, over the same rows and the same filters.
type manufacturerRows struct {
	equipmentmanufacturerports.EquipmentManufacturerRepository

	rows []*equipmentmanufacturer.EquipmentManufacturer

	mu          sync.Mutex
	connections []*equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest
}

func (m *manufacturerRows) matching(filter *pagination.QueryOptions) []*equipmentmanufacturer.EquipmentManufacturer {
	out := make([]*equipmentmanufacturer.EquipmentManufacturer, 0, len(m.rows))
	for _, row := range m.rows {
		if matches(row, filter) {
			out = append(out, row)
		}
	}

	return out
}

func matches(row *equipmentmanufacturer.EquipmentManufacturer, filter *pagination.QueryOptions) bool {
	if filter == nil {
		return true
	}
	if filter.Query != "" && !strings.Contains(strings.ToLower(row.Name), strings.ToLower(filter.Query)) {
		return false
	}
	for _, f := range filter.FieldFilters {
		switch {
		case f.Field == "status" && f.Operator == dbtype.OpIn:
			if !slices.Contains(stringsOf(f.Value), string(row.Status)) {
				return false
			}
		case f.Field == "name" && f.Operator == dbtype.OpContains:
			if !strings.Contains(strings.ToLower(row.Name), strings.ToLower(f.Value.(string))) {
				return false
			}
		}
	}

	return true
}

func stringsOf(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, item.(string))
		}
		return out
	}

	return nil
}

func (m *manufacturerRows) List(
	_ context.Context,
	req *equipmentmanufacturerports.ListEquipmentManufacturersRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	rows := m.matching(req.Filter)
	start := min(req.Filter.Pagination.Offset, len(rows))
	end := min(start+req.Filter.Pagination.Limit, len(rows))

	return &pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: rows[start:end], Total: len(rows),
	}, nil
}

func (m *manufacturerRows) ListConnection(
	_ context.Context,
	req *equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest,
) (*pagination.CursorListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	m.mu.Lock()
	m.connections = append(m.connections, req)
	m.mu.Unlock()

	rows := m.matching(req.Filter)
	start := 0
	if after := req.Cursor.Cursor.Keyset; after != "" {
		for idx, row := range rows {
			if row.ID.String() == after {
				start = idx + 1
			}
		}
	}
	end := min(start+req.Cursor.Limit+1, len(rows))
	if req.Cursor.IncludeTotalCount {
		total := len(rows)
		return pagetest.OfTotal(rows[start:end], req.Cursor.Limit, &total), nil
	}

	return pagetest.Of(rows[start:end], req.Cursor.Limit), nil
}

func (m *manufacturerRows) lastConnection() *equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.connections[len(m.connections)-1]
}

func seedManufacturers() *manufacturerRows {
	repo := &manufacturerRows{}
	for idx, name := range []string{"Freightliner", "Kenworth", "Peterbilt", "Utility", "Wabash"} {
		repo.rows = append(repo.rows, &equipmentmanufacturer.EquipmentManufacturer{
			ID: pulid.MustNew("em_"), Name: name, Status: domaintypes.StatusActive,
			CreatedAt: int64(1_760_000_000 + idx),
		})
	}
	repo.rows = append(repo.rows, &equipmentmanufacturer.EquipmentManufacturer{
		ID: pulid.MustNew("em_"), Name: "Fruehauf", Status: domaintypes.StatusInactive,
		CreatedAt: 1_750_000_000,
	})

	return repo
}

type harness struct {
	repo   *manufacturerRows
	engine permissionports.PermissionEngine
	tool   agentports.AgentQueryTool
}

func newHarness(t *testing.T, engine permissionports.PermissionEngine) *harness {
	t.Helper()

	repo := seedManufacturers()
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			EquipmentManufacturerService: equipmentmanufacturerservice.New(
				equipmentmanufacturerservice.Params{Logger: zap.NewNop(), Repo: repo},
			),
		},
	})
	tool, err := agentquerytoolservice.ProvideListEquipmentManufacturersTool(reader, engine)
	require.NoError(t, err)

	return &harness{repo: repo, engine: engine, tool: tool}
}

func (h *harness) query(
	t *testing.T,
	actor *permissionports.RequestActor,
	args map[string]any,
) (map[string]any, error) {
	t.Helper()

	return h.read(t, actor, args, false)
}

func (h *harness) read(
	t *testing.T,
	actor *permissionports.RequestActor,
	args map[string]any,
	includeTotal bool,
) (map[string]any, error) {
	t.Helper()

	data, err := h.tool.Query(readscope.With(t.Context(), nil), &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Timezone:       "America/Chicago",
		Params:         args,
		RunID:          pulid.MustNew("ar_"),
		IncludeTotal:   includeTotal,
	})
	if err != nil {
		return nil, err
	}

	return asMap(t, data), nil
}

func asMap(t *testing.T, data any) map[string]any {
	t.Helper()

	encoded, err := sonic.Marshal(data)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &out))

	return out
}

func itemIDs(outcome map[string]any) []string {
	items, _ := outcome["items"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.(map[string]any)["id"].(string))
	}

	return ids
}

func TestGraphQLListTool_TypedFiltersReachTheOperation(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})

	outcome, err := h.query(t, agentreadtest.Person(), map[string]any{
		"status":        []any{"Active"},
		"name":          "worth",
		"createdAtFrom": "2025-01-01",
		"sortBy":        "name",
		"sortDirection": "desc",
		"limit":         10,
	})

	require.NoError(t, err)
	assert.Len(t, itemIDs(outcome), 1, "only Kenworth is active and named like it")
	req := h.repo.lastConnection()
	got := make(map[string]dbtype.Operator, len(req.Filter.FieldFilters))
	for _, f := range req.Filter.FieldFilters {
		got[f.Field] = f.Operator
	}
	assert.Equal(t, map[string]dbtype.Operator{
		"status": dbtype.OpIn, "name": dbtype.OpContains, "createdAt": dbtype.OpGreaterThanOrEqual,
	}, got)
	require.Len(t, req.Filter.Sort, 1)
	assert.Equal(t, "name", req.Filter.Sort[0].Field)
	assert.Equal(t, 10, req.Cursor.Limit)
}

func TestGraphQLListTool_TwoPagesNeverRepeatARow(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})
	person := agentreadtest.Person()
	active := []any{"Active"}

	first, err := h.query(t, person, map[string]any{"status": active, "limit": 3})
	require.NoError(t, err)
	require.Equal(t, true, first["hasMore"])
	cursor, _ := first["nextCursor"].(string)
	require.NotEmpty(t, cursor)

	second, err := h.query(t, person, map[string]any{"status": active, "limit": 3, "after": cursor})
	require.NoError(t, err)

	assert.Len(t, itemIDs(first), 3)
	assert.Len(t, itemIDs(second), 2)
	for _, id := range itemIDs(second) {
		assert.NotContains(t, itemIDs(first), id)
	}
	assert.Equal(t, false, second["hasMore"])
	assert.NotContains(t, second, "nextCursor")
}

func TestGraphQLListTool_CountsTheListOnlyWhenAPersonsTableAsks(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})
	person := agentreadtest.Person()
	active := map[string]any{"status": []any{"Active"}, "limit": 2}

	counted, err := h.read(t, person, active, true)
	require.NoError(t, err)
	assert.True(t, h.repo.lastConnection().Cursor.IncludeTotalCount)
	assert.Equal(t, float64(5), counted["total"])
	assert.NotContains(t, counted, "totalEstimated", "a counted total is not marked as an estimate")
	assert.Len(t, itemIDs(counted), 2)

	uncounted, err := h.query(t, person, active)
	require.NoError(t, err)
	assert.False(t, h.repo.lastConnection().Cursor.IncludeTotalCount)
	assert.NotContains(t, uncounted, "total")
}

// A cursor is bound to the tool and the arguments that produced it: changed
// filters, an altered cursor or an offset are refused with the same errors
// the repository lists return.
func TestGraphQLListTool_KeepsTheCursorContract(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})
	person := agentreadtest.Person()

	first, err := h.query(t, person, map[string]any{"status": []any{"Active"}, "limit": 3})
	require.NoError(t, err)
	cursor, _ := first["nextCursor"].(string)
	require.NotEmpty(t, cursor)

	for name, args := range map[string]map[string]any{
		"changed filters": {"status": []any{"Inactive"}, "limit": 3, "after": cursor},
		"altered cursor":  {"status": []any{"Active"}, "limit": 3, "after": cursor + "x"},
		"a raw cursor":    {"status": []any{"Active"}, "limit": 3, "after": "not-a-cursor"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := h.query(t, person, args)
			require.ErrorIs(t, err, agentquerytoolservice.ErrStaleCursor)
		})
	}

	t.Run("offset", func(t *testing.T) {
		t.Parallel()

		_, err := h.query(t, person, map[string]any{"limit": 3, "offset": 3})
		require.ErrorIs(t, err, agentquerytoolservice.ErrOffsetOnCursorList)
	})

	t.Run("a smaller page", func(t *testing.T) {
		t.Parallel()

		_, err := h.query(t, person, map[string]any{"status": []any{"Active"}, "limit": 2, "after": cursor})
		require.NoError(t, err)
	})
}

// The permission check the tool's read meets is the one dispatch's
// authorize makes and the screen's resolver makes: the same principal, the
// same resource and operation, the same answer.
func TestGraphQLListTool_RefusesExactlyWhatAuthorizeRefuses(t *testing.T) {
	t.Parallel()

	person := agentreadtest.Person()
	engines := map[string]permissionports.PermissionEngine{
		"granted": &mocks.DenyResourcesPermissionEngine{},
		"denied": &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceEquipmentManufacturer.String()},
		},
	}
	actors := map[string]*permissionports.RequestActor{
		"a person":            person,
		"an unattended agent": agentreadtest.Agent(person),
	}
	for engineName, engine := range engines {
		for actorName, actor := range actors {
			t.Run(engineName+"/"+actorName, func(t *testing.T) {
				t.Parallel()

				check, err := engine.Check(t.Context(),
					actor.PermissionCheck(permission.ResourceEquipmentManufacturer, permission.OpRead))
				require.NoError(t, err)

				_, readErr := newHarness(t, engine).query(t, actor, map[string]any{"limit": 5})

				var refused *agentports.ReadRefusedError
				assert.Equal(t, !check.Allowed, errors.As(readErr, &refused),
					"authorize allowed=%v, the read returned %v", check.Allowed, readErr)
				if check.Allowed {
					assert.NoError(t, readErr)
				}
			})
		}
	}
}

// The repository-backed tool and the GraphQL one read the same rows for the
// same call, and the GraphQL one withholds at least what the old one did.
func TestGraphQLListTool_ReadsWhatTheRepositoryToolRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})
	old := agentquerytoolservice.NewListEquipmentManufacturersTool(h.repo)
	person := agentreadtest.Person()

	for name, args := range map[string]map[string]any{
		"every row":     {"limit": 50},
		"active ones":   {"limit": 50, "status": []any{"Active"}},
		"inactive ones": {"limit": 50, "status": []any{"Inactive"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			converted, err := h.query(t, person, args)
			require.NoError(t, err)

			oldArgs := map[string]any{"limit": 50}
			if status, ok := args["status"]; ok {
				oldArgs["filters"] = []any{map[string]any{
					"field": "status", "operator": "in", "values": status,
				}}
			}
			before, err := old.Query(t.Context(), &agentports.QueryToolParams{
				OrganizationID: person.OrganizationID,
				BusinessUnitID: person.BusinessUnitID,
				Actor:          person,
				Params:         oldArgs,
			})
			require.NoError(t, err)

			assert.ElementsMatch(t, itemIDs(asMap(t, before)), itemIDs(converted))
			if withheld, ok := asMap(t, before)["withheldByAccess"].([]any); ok {
				for _, field := range withheld {
					assert.Contains(t, converted["withheldByAccess"], field)
				}
			}
		})
	}
}

// Walking every page of the GraphQL tool reads, without a repeat, exactly the
// rows walking every page of the repository tool read.
func TestGraphQLListTool_PagesThroughWhatTheRepositoryToolPagedThrough(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &mocks.DenyResourcesPermissionEngine{})
	old := agentquerytoolservice.NewListEquipmentManufacturersTool(h.repo)
	person := agentreadtest.Person()

	var converted []string
	args := map[string]any{"limit": 2}
	for pages := 0; ; pages++ {
		require.Less(t, pages, 10, "the GraphQL tool never reached its last page")
		got, err := h.query(t, person, args)
		require.NoError(t, err)
		converted = append(converted, itemIDs(got)...)
		cursor, _ := got["nextCursor"].(string)
		if got["hasMore"] != true {
			break
		}
		require.NotEmpty(t, cursor)
		args = map[string]any{"limit": 2, "after": cursor}
	}

	var before []string
	oldArgs := map[string]any{"limit": 2}
	for pages := 0; ; pages++ {
		require.Less(t, pages, 10, "the repository tool never reached its last page")
		got, err := old.Query(t.Context(), &agentports.QueryToolParams{
			OrganizationID: person.OrganizationID,
			BusinessUnitID: person.BusinessUnitID,
			Actor:          person,
			Params:         oldArgs,
		})
		require.NoError(t, err)
		page := asMap(t, got)
		before = append(before, itemIDs(page)...)
		next, ok := page["nextOffset"]
		if !ok {
			break
		}
		oldArgs = map[string]any{"limit": 2, "offset": next}
	}

	require.Greater(t, len(converted), 2, "the walk must cross a page")
	assert.ElementsMatch(t, before, converted)
	seen := make(map[string]bool, len(converted))
	for _, id := range converted {
		assert.False(t, seen[id], "row %v read twice", id)
		seen[id] = true
	}
}

// publicRole is a person whose role reaches only public fields, so every
// internal field of a manufacturer, the non-null name among them, is hidden.
type publicRole struct {
	mocks.DenyResourcesPermissionEngine
}

func (publicRole) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{MaxSensitivity: permission.SensitivityPublic}, nil
}

func (r publicRole) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, r, userID, orgID, resources)
}

func TestGraphQLListTool_AWithheldFieldStaysWithheld(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &publicRole{})
	person := agentreadtest.Person()

	outcome, err := h.query(t, person, map[string]any{"limit": 2})
	require.NoError(t, err)

	items, _ := outcome["items"].([]any)
	require.Len(t, items, 2, "the rows are still read; only what they hold is withheld")
	for _, item := range items {
		row := item.(map[string]any)
		assert.NotContains(t, row, "name", "a withheld non-null field never reaches the tool")
		assert.NotContains(t, row, "status")
		assert.NotEmpty(t, row["id"])
	}
	assert.Contains(t, outcome["withheldByAccess"], "name")
	assert.NotEmpty(t, outcome["withheldNote"])

	_, err = h.query(t, person, map[string]any{"name": "Kenworth"})
	require.Error(t, err, "filtering on a hidden field would leak it a row at a time")
	assert.Contains(t, err.Error(), "name")
}

// A composed view is held to the same rule as a call: a filter or an order
// on a field the reader may not see is refused before anything is read.
func TestGraphQLListTool_AViewCannotFilterOnAWithheldField(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &publicRole{})
	person := agentreadtest.Person()
	params := &agentports.QueryToolParams{
		OrganizationID: person.OrganizationID,
		BusinessUnitID: person.BusinessUnitID,
		Actor:          person,
		Timezone:       "America/Chicago",
		RunID:          pulid.MustNew("ar_"),
	}
	ctx := readscope.With(t.Context(), nil)

	err := agentquerytoolservice.RunView(ctx, h.tool, params, []domaintypes.FieldFilter{{
		Field: "name", Operator: dbtype.OpContains, Value: "Ken",
	}}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")

	err = agentquerytoolservice.RunView(ctx, h.tool, params, nil,
		[]domaintypes.SortField{{Field: "name", Direction: dbtype.SortDirectionAsc}})
	require.Error(t, err, "ordering by a hidden field leaks it too")
	assert.Empty(t, h.repo.connections, "nothing is read for a refused view")
}
