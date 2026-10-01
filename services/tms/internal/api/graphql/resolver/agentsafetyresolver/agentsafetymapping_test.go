package agentsafetyresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsafetyservice"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Null reads every agent; an empty list names none. The two must not
// collapse into one another on the way to the service.
func TestAgentSafetyIDsKeepsNullApartFromEmpty(t *testing.T) {
	t.Parallel()

	all, err := agentSafetyIDs(nil)
	require.NoError(t, err)
	assert.Nil(t, all)

	none, err := agentSafetyIDs([]string{})
	require.NoError(t, err)
	require.NotNil(t, none)
	assert.Empty(t, none)

	_, err = agentSafetyIDs([]string{"not an id"})
	require.Error(t, err)
}

// The deprecated connection's own filters become the table's field filters,
// so both connections narrow the rules through the same code.
func TestToolPolicyConnectionRequestLeavesAbsentFiltersOpen(t *testing.T) {
	t.Parallel()

	assert.Equal(t, memtable.Request{},
		toolPolicyConnectionRequest(&gqlmodel.AgentToolPolicyConnectionInput{}))

	first := 25
	after := "cursor"
	query := "email"
	egress := agent.EgressExternalRecipient
	resource := "customer"
	kind := agent.ToolKindAction
	alone := false
	req := toolPolicyConnectionRequest(&gqlmodel.AgentToolPolicyConnectionInput{
		First:             &first,
		After:             &after,
		Query:             &query,
		Egress:            &egress,
		Resource:          &resource,
		Kind:              &kind,
		RunsWithoutPerson: &alone,
	})
	assert.Equal(t, 25, req.First)
	assert.Equal(t, "cursor", req.After)
	assert.Equal(t, "email", req.Query)
	assert.Equal(t, []domaintypes.FieldFilter{
		{Field: "egress", Operator: dbtype.OpEqual, Value: "external_recipient"},
		{Field: "resource", Operator: dbtype.OpEqual, Value: "customer"},
		{Field: "kind", Operator: dbtype.OpEqual, Value: "action"},
		{
			Field:    agentsafetyservice.FieldRunsWithoutPerson,
			Operator: dbtype.OpEqual,
			Value:    false,
		},
	}, req.FieldFilters)
}

func TestAgentToolSafetyPageToModelEndsOnTheLastEdge(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	out := agentToolSafetyPageToModel(&services.AgentToolSafetyPage{
		Edges: []services.AgentToolSafetyEdge{
			{Node: services.AgentToolSafety{AgentID: agentID, PolicyName: "a"}, Cursor: "c1"},
			{Node: services.AgentToolSafety{AgentID: agentID, PolicyName: "b"}, Cursor: "c2"},
		},
		HasNextPage: true,
	})
	require.Len(t, out.Edges, 2)
	assert.Equal(t, agentID.String()+":b", out.Edges[1].Node.RowID())
	require.NotNil(t, out.PageInfo.EndCursor)
	assert.Equal(t, "c2", *out.PageInfo.EndCursor)
	assert.True(t, out.PageInfo.HasNextPage)
	assert.Nil(t, out.TotalCount)
}

func TestToolPolicyPageToModelEndsOnTheLastEdge(t *testing.T) {
	t.Parallel()

	total := 40
	edge := func(name, cursor string) services.AgentToolPolicyEdge {
		return services.AgentToolPolicyEdge{
			View:   services.AgentToolPolicyView{Policy: services.ToolPolicy{Name: name}},
			Cursor: cursor,
		}
	}
	alone := true
	attended := edge("b", "c2")
	attended.RunsWithoutPerson = &alone
	out := toolPolicyPageToModel(&services.AgentToolPolicyPage{
		Edges:       []services.AgentToolPolicyEdge{edge("a", "c1"), attended},
		HasNextPage: true,
		TotalCount:  &total,
	})
	require.Len(t, out.Edges, 2)
	assert.Equal(t, "b", out.Edges[1].Node.Name)
	assert.Nil(t, out.Edges[0].Node.RunsWithoutPerson, "a rule asked nothing of says nothing")
	require.NotNil(t, out.Edges[1].Node.RunsWithoutPerson)
	assert.True(t, *out.Edges[1].Node.RunsWithoutPerson)
	assert.True(t, out.PageInfo.HasNextPage)
	require.NotNil(t, out.PageInfo.EndCursor)
	assert.Equal(t, "c2", *out.PageInfo.EndCursor)
	assert.Same(t, &total, out.TotalCount)

	empty := toolPolicyPageToModel(&services.AgentToolPolicyPage{})
	assert.Empty(t, empty.Edges)
	assert.Nil(t, empty.PageInfo.EndCursor)
	assert.Nil(t, empty.TotalCount)
}
