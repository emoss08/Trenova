package agentquerytoolservice

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readOnly struct {
	permissionports.PermissionEngine

	denied []permission.Resource
}

func (r *readOnly) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: !slices.Contains(r.denied, permission.Resource(req.Resource)),
	}, nil
}

type cannedList struct {
	resource permission.Resource
	answers  map[string][]map[string]any

	mu    sync.Mutex
	asked []map[string]any
}

func (c *cannedList) Name() string                { return "list_" + c.resource.String() }
func (c *cannedList) Description() string         { return "" }
func (c *cannedList) ParamSchema() map[string]any { return map[string]any{} }
func (c *cannedList) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{Resource: c.resource}
}

func (c *cannedList) Query(_ context.Context, params *agentports.QueryToolParams) (any, error) {
	c.mu.Lock()
	c.asked = append(c.asked, params.Params)
	c.mu.Unlock()

	query, _ := params.Params["query"].(string)

	return map[string]any{"count": len(c.answers[query]), "items": c.answers[query]}, nil
}

func recordsTool(permissions permissionports.PermissionEngine) (
	*findShipmentRecordsTool,
	*cannedList,
	*cannedList,
	*cannedList,
) {
	customers := &cannedList{resource: permission.ResourceCustomer, answers: map[string][]map[string]any{
		"ACME": {{"id": "cus_1", "code": "ACME", "name": "Acme Manufacturing", "billingProfile": "x"}},
	}}
	locations := &cannedList{resource: permission.ResourceLocation, answers: map[string][]map[string]any{
		"1200 E 7th St": {{"id": "loc_1", "code": "TERM-LA", "name": "Los Angeles Terminal"}},
	}}
	commodities := &cannedList{resource: permission.ResourceCommodity, answers: map[string][]map[string]any{}}
	categories := &cannedList{resource: permission.ResourceLocationCategory, answers: map[string][]map[string]any{
		"": {{"id": "lc_1", "name": "Central Warehouse", "description": "Main storage"}},
	}}

	return &findShipmentRecordsTool{
		customers:   customers,
		locations:   locations,
		commodities: commodities,
		choices:     []recordChoice{{key: "locationCategories", tool: categories}},
		access:      newFieldAccess(permissions),
	}, customers, locations, commodities
}

func TestFindShipmentRecords_MatchesEverythingTheLoadNamesInOneCall(t *testing.T) {
	t.Parallel()

	tool, customers, locations, _ := recordsTool(&readOnly{})
	params := testParams(map[string]any{
		"customer": map[string]any{"name": "Acme", "code": "ACME"},
		"locations": []any{
			map[string]any{
				"name": "LA terminal", "addressLine1": "1200 E 7th St", "city": "Los Angeles",
				"state": "CA",
			},
			map[string]any{"name": "Summit Sports Boise", "city": "Boise", "state": "ID"},
		},
		"commodities": []any{"camping gear"},
	})
	params.Actor.PrincipalType = permissionports.PrincipalTypeUser

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	records := result.(*shipmentRecords)

	require.NotNil(t, records.Customer)
	assert.True(t, records.Customer.Found)
	assert.Equal(t, []map[string]any{
		{"id": "cus_1", "code": "ACME", "name": "Acme Manufacturing"},
	}, records.Customer.Matches, "matches keep the columns that identify a record")
	assert.Equal(t, "ACME", customers.asked[0]["query"], "a code is the surest match")

	require.Len(t, records.Locations, 2)
	assert.True(t, records.Locations[0].Found, "a place is found by its street when its name misses")
	assert.Len(t, locations.asked, 3)
	assert.False(t, records.Locations[1].Found)
	assert.Equal(t, "Summit Sports Boise, Boise ID", records.Locations[1].Asked)

	require.Len(t, records.Commodities, 1)
	assert.False(t, records.Commodities[0].Found)

	assert.Equal(t, []string{
		"location Summit Sports Boise, Boise ID", "commodity camping gear",
	}, records.Missing)
	assert.Equal(t, recordsSomeMissing, records.Next)
	assert.Equal(t, []map[string]any{
		{"id": "lc_1", "name": "Central Warehouse", "description": "Main storage"},
	}, records.Choices["locationCategories"])
}

func TestFindShipmentRecords_SaysWhenEverythingIsThere(t *testing.T) {
	t.Parallel()

	tool, _, _, _ := recordsTool(&readOnly{})
	params := testParams(map[string]any{"customer": map[string]any{"code": "ACME"}})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	records := result.(*shipmentRecords)
	assert.Empty(t, records.Missing)
	assert.Equal(t, recordsAllFound, records.Next)
}

func TestFindShipmentRecords_ReadsNothingThePersonMayNotRead(t *testing.T) {
	t.Parallel()

	tool, customers, _, _ := recordsTool(&readOnly{denied: []permission.Resource{
		permission.ResourceCustomer,
	}})
	params := testParams(map[string]any{"customer": map[string]any{"code": "ACME"}})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	records := result.(*shipmentRecords)
	assert.Empty(t, customers.asked)
	assert.False(t, records.Customer.Found)
	assert.Equal(t, recordsUnreadable, records.Customer.Note)
}
