package agentsafetyresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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

func TestAgentToolSafetyPageToModelEndsOnTheLastEdge(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	out := agentToolSafetyPageToModel(&agenttoolports.AgentToolSafetyPage{
		Edges: []agenttoolports.AgentToolSafetyEdge{
			{Node: agenttoolports.AgentToolSafety{AgentID: agentID, PolicyName: "a"}, Cursor: "c1"},
			{Node: agenttoolports.AgentToolSafety{AgentID: agentID, PolicyName: "b"}, Cursor: "c2"},
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
	edge := func(name, cursor string) agenttoolports.AgentToolPolicyEdge {
		return agenttoolports.AgentToolPolicyEdge{
			View: agenttoolports.AgentToolPolicyView{
				Policy: agenttoolports.ToolPolicy{Name: name},
			},
			Cursor: cursor,
		}
	}
	alone := true
	attended := edge("b", "c2")
	attended.RunsWithoutPerson = &alone
	out := toolPolicyPageToModel(&agenttoolports.AgentToolPolicyPage{
		Edges:       []agenttoolports.AgentToolPolicyEdge{edge("a", "c1"), attended},
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

	empty := toolPolicyPageToModel(&agenttoolports.AgentToolPolicyPage{})
	assert.Empty(t, empty.Edges)
	assert.Nil(t, empty.PageInfo.EndCursor)
	assert.Nil(t, empty.TotalCount)
}
