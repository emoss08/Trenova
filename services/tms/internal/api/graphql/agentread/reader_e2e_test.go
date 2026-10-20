package agentread_test

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const manufacturerTable = "EquipmentManufacturerTable"

// manufacturers is a repository holding a fixed list, paged by id the way
// the cursor paging the resolver asks for reads it.
type manufacturers struct {
	equipmentmanufacturerports.EquipmentManufacturerRepository

	rows []*equipmentmanufacturer.EquipmentManufacturer

	mu   sync.Mutex
	seen []*equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest
}

func (m *manufacturers) ListConnection(
	_ context.Context,
	req *equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest,
) (*pagination.CursorListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	m.mu.Lock()
	m.seen = append(m.seen, req)
	m.mu.Unlock()

	start := 0
	if after := req.Cursor.Cursor.ID; after.IsNotNil() {
		for idx, row := range m.rows {
			if row.ID == after {
				start = idx + 1
			}
		}
	}
	end := min(start+req.Cursor.Limit+1, len(m.rows))

	return pagination.NewCursorListResult(m.rows[start:end], req.Cursor.Limit), nil
}

func (m *manufacturers) requests() []*equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.seen
}

func manufacturerReader(
	t *testing.T,
	engine *mocks.DenyResourcesPermissionEngine,
	repo *manufacturers,
) *agentread.Reader {
	t.Helper()

	return agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			EquipmentManufacturerService: equipmentmanufacturerservice.New(
				equipmentmanufacturerservice.Params{Logger: zap.NewNop(), Repo: repo},
			),
		},
	})
}

func seededManufacturers(n int) *manufacturers {
	repo := &manufacturers{}
	for i := range n {
		repo.rows = append(repo.rows, &equipmentmanufacturer.EquipmentManufacturer{
			ID:        pulid.MustNew("em_"),
			Name:      "Manufacturer " + string(rune('A'+i)),
			Status:    domaintypes.StatusActive,
			CreatedAt: int64(1_760_000_000 + i),
		})
	}

	return repo
}

func connectionOf(t *testing.T, result *agentports.GraphQLReadResult) map[string]any {
	t.Helper()

	require.Empty(t, result.Errors)
	connection, ok := result.Data["equipmentManufacturers"].(map[string]any)
	require.True(t, ok, "the read returns the connection")

	return connection
}

func nodeIDs(t *testing.T, connection map[string]any) []string {
	t.Helper()

	edges, _ := connection["edges"].([]any)
	ids := make([]string, 0, len(edges))
	for _, edge := range edges {
		node := edge.(map[string]any)["node"].(map[string]any)
		ids = append(ids, node["id"].(string))
	}

	return ids
}

func TestReader_RunsABoundOperationThroughTheReadExecutor(t *testing.T) {
	t.Parallel()

	repo := seededManufacturers(3)
	reader := manufacturerReader(t, &mocks.DenyResourcesPermissionEngine{}, repo)

	result, err := agentreadtest.Read(t.Context(), reader, agentreadtest.Person(),
		manufacturerTable, map[string]any{
			"includeTotalCount": false,
			"input": map[string]any{
				"first": 10,
				"query": "manufacturer",
				"fieldFilters": []any{
					map[string]any{"field": "status", "operator": "in", "value": []any{"Active"}},
				},
			},
		}, permission.SensitivityInternal)

	require.NoError(t, err)
	assert.Len(t, nodeIDs(t, connectionOf(t, result)), 3)
	assert.Positive(t, result.Cost, "the read is charged to the caller's budget")

	requests := repo.requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "manufacturer", requests[0].Filter.Query)
	require.Len(t, requests[0].Filter.FieldFilters, 1)
	assert.Equal(t, "status", requests[0].Filter.FieldFilters[0].Field)
	assert.False(t, requests[0].Cursor.IncludeTotalCount,
		"an operation that does not select totalCount runs no count")
}

func TestReader_PagesWithTheConnectionsCursorAndNeverRepeatsARow(t *testing.T) {
	t.Parallel()

	repo := seededManufacturers(5)
	reader := manufacturerReader(t, &mocks.DenyResourcesPermissionEngine{}, repo)
	person := agentreadtest.Person()
	page := func(after any) map[string]any {
		input := map[string]any{"first": 3}
		if after != nil {
			input["after"] = after
		}
		result, err := agentreadtest.Read(t.Context(), reader, person, manufacturerTable,
			map[string]any{"includeTotalCount": false, "input": input}, permission.SensitivityInternal)
		require.NoError(t, err)

		return connectionOf(t, result)
	}

	first := page(nil)
	info := first["pageInfo"].(map[string]any)
	require.Equal(t, true, info["hasNextPage"])
	second := page(info["endCursor"])

	firstIDs, secondIDs := nodeIDs(t, first), nodeIDs(t, second)
	assert.Len(t, firstIDs, 3)
	assert.Len(t, secondIDs, 2)
	for _, id := range secondIDs {
		assert.NotContains(t, firstIDs, id)
	}
	assert.Equal(t, false, second["pageInfo"].(map[string]any)["hasNextPage"])
}

func TestReader_RefusesWhatTheScreenRefuses(t *testing.T) {
	t.Parallel()

	repo := seededManufacturers(2)
	reader := manufacturerReader(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceEquipmentManufacturer.String()},
	}, repo)

	t.Run("actors", func(t *testing.T) {
		for name, actor := range map[string]*permissionports.RequestActor{
			"a person": agentreadtest.Person(),
			"an agent": agentreadtest.Agent(agentreadtest.Person()),
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				result, err := agentreadtest.Read(t.Context(), reader,
					actor, manufacturerTable,
					map[string]any{"input": map[string]any{"first": 10}},
					permission.SensitivityInternal)

				require.NoError(t, err)
				require.NotEmpty(t, result.Errors)
				assert.Equal(t, agentports.GraphQLCodeForbidden, result.Errors[0].Code)
			})
		}
	})
	assert.Empty(t, repo.requests(), "a refused read never reaches the repository")
}

func TestReader_RefusesAnOperationNoToolIsBoundTo(t *testing.T) {
	t.Parallel()

	reader := manufacturerReader(t, &mocks.DenyResourcesPermissionEngine{}, seededManufacturers(1))

	_, err := agentreadtest.Read(t.Context(), reader, agentreadtest.Person(), "ShipmentBoardTable",
		map[string]any{}, permission.SensitivityInternal)

	assert.ErrorIs(t, err, agentports.ErrUnknownReadOperation)
}
