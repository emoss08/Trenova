package agentsafetyresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolPolicyViewToModel(t *testing.T) {
	t.Parallel()

	condition := &agenttoolports.TierCondition{
		Description: "Runs only as far as its mailbox allows.",
		Limit: func(context.Context, *agenttoolports.ToolExecuteParams) agent.AutonomyTier {
			return agent.TierPropose
		},
	}
	view := agenttoolports.AgentToolPolicyView{
		Policy: agenttoolports.ToolPolicy{
			Name:          "inbox_send_reply",
			Kind:          agenttooltrust.ToolKindAction,
			Scope:         agenttooltrust.ToolScopeTenant,
			DefaultTier:   agent.TierActWithApproval,
			MaxTier:       agent.TierActWithApproval,
			Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
			Condition:     condition,
			Effect:        agenttooltrust.ToolEffectChange,
			ReadsExternal: agenttooltrust.ExternalReadAlways,
			Source:        agent.TaintSourceInboundMessage,
			Rationale:     "A reply reaches the sender.",
		},
		Title: "Reply to inbound message",
		Needs: &agenttoolports.ToolGrant{
			Resource:  permission.ResourceInboundMessage,
			Operation: permission.OpUpdate,
		},
		Promotable:  agent.TierActWithApproval,
		Leaves:      true,
		Explanation: "Runs at the tier the agent sets for it, never past ActWithApproval.",
	}

	model := ToolPolicyViewToModel(&view)

	assert.Equal(t, "inbox_send_reply", model.ID)
	assert.Equal(t, "inbox_send_reply", model.Name)
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

	selfScoped := ToolPolicyViewToModel(&agenttoolports.AgentToolPolicyView{
		Policy: agenttoolports.ToolPolicy{
			Name: "home_add_widget",
			Kind: agenttooltrust.ToolKindAction,
		},
	})
	assert.Nil(t, selfScoped.Needs)
	assert.Nil(t, selfScoped.Source)
	assert.Nil(t, selfScoped.ConditionDescription)
	assert.Equal(t, agenttooltrust.ToolEffectChange, selfScoped.Effect)
	assert.Equal(t, agenttooltrust.ExternalReadNever, selfScoped.ReadsExternal)
	assert.Equal(t, agent.TierPropose, selfScoped.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, selfScoped.MaxTier)
}

func TestMemtableRequestFromGraphQLCarriesEveryTableInput(t *testing.T) {
	t.Parallel()

	first := 50
	after := "cursor"
	query := "move"
	req := gqlconnection.MemtableRequestFromGraphQL(t.Context(), &gqlmodel.DataTableConnectionInput{
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
		gqlconnection.MemtableRequestFromGraphQL(t.Context(), nil),
	)
}
