package agentsafetyservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryCountsChangesByAudienceAndNamesWhatRunsAndWhoIsOpen(t *testing.T) {
	t.Parallel()

	open := agentWith("Customer desk", "email_customer", "assign_move")
	restricted := agentWith("Billing desk", "email_customer")
	restricted.AccessMode = agentdefinition.AccessRoles
	svc := testService(&fakeDefinitions{all: []*agentdefinition.Definition{open, restricted}}, &tenant.AgentControl{})

	summary, err := svc.Summary(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	assert.Equal(t, []services.AgentEgressCount{
		{Egress: agent.EgressNone, Count: 1},
		{Egress: agent.EgressPersonal, Count: 1},
		{Egress: agent.EgressInternal, Count: 1},
		{Egress: agent.EgressExternalRecipient, Count: 1},
	}, summary.EgressCounts)
	assert.Equal(t, []pulid.ID{open.ID}, summary.OpenSensitiveAgentIDs)
	assert.Equal(t, summary.OpenWithSensitive, len(summary.OpenSensitiveAgentIDs))
	assert.Len(t, summary.UnattendedTools, summary.RunWithoutPerson)
	assign, ok := svc.ToolPolicy("assign_move")
	require.True(t, ok)
	assert.Contains(t, summary.UnattendedTools, assign.Title)
}

func TestToolHoldersListsEachAgentOnceUnderEveryToolItHolds(t *testing.T) {
	t.Parallel()

	first := agentWith("Customer desk", "email_customer", "assign_move")
	second := agentWith("Billing desk", "email_customer")
	svc := testService(&fakeDefinitions{all: []*agentdefinition.Definition{first, second}}, &tenant.AgentControl{})

	holders, err := svc.ToolHolders(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	byTool := make(map[string][]pulid.ID, len(holders))
	for _, entry := range holders {
		byTool[entry.PolicyName] = entry.AgentIDs
	}
	assert.Equal(t, []pulid.ID{first.ID, second.ID}, byTool["email_customer"])
	assert.Equal(t, []pulid.ID{first.ID}, byTool["assign_move"])
	assert.NotContains(t, byTool, "unregistered_tool")
	assert.NotContains(t, byTool, "get_shipment")
}

type fakeRules struct {
	rules map[string]*agent.ToolRuleOverride
}

func (f *fakeRules) For(context.Context, pagination.TenantInfo) (map[string]*agent.ToolRuleOverride, error) {
	return f.rules, nil
}

func TestTheOrganizationsToolRulesShapeEverySafetyRead(t *testing.T) {
	t.Parallel()

	held := &agent.ToolRuleOverride{ToolName: "assign_move", MaxTier: agent.TierPropose, Reason: "hold", Version: 3}
	reset := &agent.ToolRuleOverride{ToolName: "email_customer", Reason: "released", Version: 2}
	desk := agentWith("Dispatch desk", "assign_move", "email_customer")
	svc := testService(&fakeDefinitions{all: []*agentdefinition.Definition{desk}}, &tenant.AgentControl{})
	svc.rules = &fakeRules{rules: map[string]*agent.ToolRuleOverride{"assign_move": held, "email_customer": reset}}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	desk.OrganizationID, desk.BusinessUnitID = tenantInfo.OrgID, tenantInfo.BuID

	summary, err := svc.Summary(t.Context(), tenantInfo)
	require.NoError(t, err)
	assign, ok := svc.ToolPolicy("assign_move")
	require.True(t, ok)
	assert.NotContains(t, summary.UnattendedTools, assign.Title, "a tool held to proposals never runs alone")

	page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{TenantInfo: tenantInfo})
	require.NoError(t, err)
	views := make(map[string]services.AgentToolPolicyView, len(page.Edges))
	for _, edge := range page.Edges {
		views[edge.View.Policy.Name] = edge.View
	}
	assert.Equal(t, agent.TierPropose, views["assign_move"].Policy.MaxTier)
	require.NotNil(t, views["assign_move"].Declared)
	assert.Equal(t, agent.TierAutoExecute, views["assign_move"].Declared.MaxTier)
	assert.Same(t, held, views["assign_move"].Override)
	assert.Same(t, reset, views["email_customer"].Override, "a reset rule keeps its version for the next edit")
	assert.Equal(t, agent.TierAutoExecute, views["email_customer"].Policy.MaxTier)
	assert.Nil(t, views["get_shipment"].Override)

	assessed := svc.Assess(t.Context(), &services.AssessAgentSafetyRequest{
		Subject: &services.AgentSafetySubject{Agent: desk, Control: &tenant.AgentControl{}},
	})
	for _, tool := range assessed {
		if tool.PolicyName == "assign_move" {
			assert.Equal(t, agent.AutonomyProposeOnly, tool.Clean.Answer)
			require.NotNil(t, tool.Policy)
			assert.Same(t, held, tool.Policy.Override)
		}
	}

	impacts, err := svc.RuleImpact(t.Context(), &services.RuleImpactRequest{
		TenantInfo: tenantInfo,
		ToolName:   "email_customer",
		Before:     policy("email_customer", agent.ToolKindAction, agent.EgressExternalRecipient),
		After: func() services.ToolPolicy {
			p := policy("email_customer", agent.ToolKindAction, agent.EgressExternalRecipient)
			p.MaxTier, p.DefaultTier = agent.TierPropose, agent.TierPropose
			return p
		}(),
	})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	assert.Equal(t, desk.ID, impacts[0].AgentID)
	assert.NotEqual(t, impacts[0].Before, impacts[0].After)
	assert.Equal(t, agent.AutonomyProposeOnly, impacts[0].After)
}
