package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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

func TestToolPolicyViewToModel(t *testing.T) {
	t.Parallel()

	condition := &services.TierCondition{
		Description: "Runs only as far as its mailbox allows.",
		Limit: func(context.Context, services.ToolExecuteParams) agent.AutonomyTier {
			return agent.TierPropose
		},
	}
	view := services.AgentToolPolicyView{
		Policy: services.ToolPolicy{
			Name:          "reply_to_inbound_message",
			Kind:          agent.ToolKindAction,
			Scope:         agent.ToolScopeTenant,
			DefaultTier:   agent.TierActWithApproval,
			MaxTier:       agent.TierActWithApproval,
			Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
			Condition:     condition,
			Effect:        agent.ToolEffectChange,
			ReadsExternal: agent.ExternalReadAlways,
			Source:        agent.TaintSourceInboundMessage,
			Rationale:     "A reply reaches the sender.",
		},
		Title: "Reply to inbound message",
		Needs: &services.ToolGrant{
			Resource:  permission.ResourceInboundMessage,
			Operation: permission.OpUpdate,
		},
		Promotable:  agent.TierActWithApproval,
		Leaves:      true,
		Explanation: "Runs at the tier the agent sets for it, never past ActWithApproval.",
	}

	model := toolPolicyViewToModel(&view)

	assert.Equal(t, "reply_to_inbound_message", model.ID)
	assert.Equal(t, "reply_to_inbound_message", model.Name)
	assert.Equal(t, "Reply to inbound message", model.Title)
	assert.True(t, model.HasCondition)
	assert.False(t, model.HasClassify)
	require.NotNil(t, model.ConditionDescription)
	assert.Equal(t, condition.Description, *model.ConditionDescription)
	require.NotNil(t, model.Source)
	assert.Equal(t, agent.TaintSourceInboundMessage, *model.Source)
	require.NotNil(t, model.Needs)
	assert.Equal(t, "inbound_message", model.Needs.Resource)
	assert.Equal(t, "update", model.Needs.Operation)
	assert.True(t, model.LeavesOrganization)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, model.Egress)

	view.Policy.Egress[0] = agent.EgressNone
	assert.Equal(t, agent.EgressExternalRecipient, model.Egress[0],
		"the model keeps its own copy of the classes")

	selfScoped := toolPolicyViewToModel(&services.AgentToolPolicyView{
		Policy: services.ToolPolicy{Name: "add_home_widget", Kind: agent.ToolKindAction},
	})
	assert.Nil(t, selfScoped.Needs)
	assert.Nil(t, selfScoped.Source)
	assert.Nil(t, selfScoped.ConditionDescription)
	assert.Equal(t, agent.ToolEffectChange, selfScoped.Effect)
	assert.Equal(t, agent.ExternalReadNever, selfScoped.ReadsExternal)
	assert.Equal(t, agent.TierPropose, selfScoped.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, selfScoped.MaxTier)
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

func TestMemtableRequestFromGraphQLCarriesEveryTableInput(t *testing.T) {
	t.Parallel()

	first := 50
	after := "cursor"
	query := "move"
	req := memtableRequestFromGraphQL(t.Context(), &gqlmodel.DataTableConnectionInput{
		First: &first,
		After: &after,
		Query: &query,
		FieldFilters: []*gqlmodel.FieldFilterInput{
			{Field: "egress", Operator: "in", Value: []any{"None", "Internal"}},
		},
		FilterGroups: []*gqlmodel.FilterGroupInput{{Filters: []*gqlmodel.FieldFilterInput{
			{Field: "kind", Operator: "eq", Value: "Query"},
		}}},
		Sort: []*gqlmodel.SortFieldInput{{Field: "maxTier", Direction: "desc"}},
	})

	assert.Equal(t, 50, req.First)
	assert.Equal(t, "cursor", req.After)
	assert.Equal(t, "move", req.Query)
	require.Len(t, req.FieldFilters, 1)
	assert.Equal(t, "egress", req.FieldFilters[0].Field)
	assert.Equal(t, []string{"None", "Internal"}, req.FieldFilters[0].Value)
	require.Len(t, req.FilterGroups, 1)
	assert.Equal(t, "kind", req.FilterGroups[0].Filters[0].Field)
	assert.Equal(t, []domaintypes.SortField{
		{Field: "maxTier", Direction: dbtype.SortDirectionDesc},
	}, req.Sort)
	assert.True(t, req.IncludeTotalCount, "outside a request the count is kept")

	assert.Equal(t,
		memtable.Request{IncludeTotalCount: true},
		memtableRequestFromGraphQL(t.Context(), nil),
	)
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
