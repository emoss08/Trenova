package agentsafetyresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	model := ToolPolicyViewToModel(&view)

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

	selfScoped := ToolPolicyViewToModel(&services.AgentToolPolicyView{
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

func TestMemtableRequestFromGraphQLCarriesEveryTableInput(t *testing.T) {
	t.Parallel()

	first := 50
	after := "cursor"
	query := "move"
	req := base.MemtableRequestFromGraphQL(t.Context(), &gqlmodel.DataTableConnectionInput{
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
		base.MemtableRequestFromGraphQL(t.Context(), nil),
	)
}
