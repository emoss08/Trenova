package agentsafetyservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryCountsChangesByAudienceAndNamesWhatRunsAndWhoIsOpen(t *testing.T) {
	t.Parallel()

	open := agentWith("Customer desk", "customer_send_email", "dispatch_assign_move")
	restricted := agentWith("Billing desk", "customer_send_email")
	restricted.AccessMode = agentdefinition.AccessRoles
	svc := testService(
		&fakeDefinitions{all: []*agentdefinition.Definition{open, restricted}},
		&agentcontrol.AgentControl{},
	)

	summary, err := svc.Summary(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	assert.Equal(t, []agenttoolports.AgentEgressCount{
		{Egress: agent.EgressNone, Count: 1},
		{Egress: agent.EgressPersonal, Count: 1},
		{Egress: agent.EgressInternal, Count: 1},
		{Egress: agent.EgressExternalRecipient, Count: 1},
	}, summary.EgressCounts)
	assert.Equal(t, []pulid.ID{open.ID}, summary.OpenSensitiveAgentIDs)
	assert.Equal(t, summary.OpenWithSensitive, len(summary.OpenSensitiveAgentIDs))
	assert.Len(t, summary.UnattendedTools, summary.RunWithoutPerson)
	assign, ok := svc.ToolPolicy("dispatch_assign_move")
	require.True(t, ok)
	assert.Contains(t, summary.UnattendedTools, assign.Title)
}

func TestToolHoldersListsEachAgentOnceUnderEveryToolItHolds(t *testing.T) {
	t.Parallel()

	first := agentWith("Customer desk", "customer_send_email", "dispatch_assign_move")
	second := agentWith("Billing desk", "customer_send_email")
	svc := testService(
		&fakeDefinitions{all: []*agentdefinition.Definition{first, second}},
		&agentcontrol.AgentControl{},
	)

	holders, err := svc.ToolHolders(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	byTool := make(map[string][]pulid.ID, len(holders))
	for _, entry := range holders {
		byTool[entry.PolicyName] = entry.AgentIDs
	}
	assert.Equal(t, []pulid.ID{first.ID, second.ID}, byTool["customer_send_email"])
	assert.Equal(t, []pulid.ID{first.ID}, byTool["dispatch_assign_move"])
	assert.NotContains(t, byTool, "unregistered_tool")
	assert.NotContains(t, byTool, "shipment_get")
}

type fakeRules struct {
	rules map[string]*agenttooltrust.ToolRuleOverride
}

func (f *fakeRules) For(
	context.Context,
	pagination.TenantInfo,
) (map[string]*agenttooltrust.ToolRuleOverride, error) {
	return f.rules, nil
}

func TestTheOrganizationsToolRulesShapeEverySafetyRead(t *testing.T) {
	t.Parallel()

	held := &agenttooltrust.ToolRuleOverride{
		ToolName: "dispatch_assign_move",
		MaxTier:  agent.TierPropose,
		Reason:   "hold",
		Version:  3,
	}
	reset := &agenttooltrust.ToolRuleOverride{
		ToolName: "customer_send_email",
		Reason:   "released",
		Version:  2,
	}
	desk := agentWith("Dispatch desk", "dispatch_assign_move", "customer_send_email")
	svc := testService(
		&fakeDefinitions{all: []*agentdefinition.Definition{desk}},
		&agentcontrol.AgentControl{},
	)
	svc.rules = &fakeRules{
		rules: map[string]*agenttooltrust.ToolRuleOverride{
			"dispatch_assign_move": held,
			"customer_send_email":  reset,
		},
	}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	desk.OrganizationID, desk.BusinessUnitID = tenantInfo.OrgID, tenantInfo.BuID

	summary, err := svc.Summary(t.Context(), tenantInfo)
	require.NoError(t, err)
	assign, ok := svc.ToolPolicy("dispatch_assign_move")
	require.True(t, ok)
	assert.NotContains(
		t,
		summary.UnattendedTools,
		assign.Title,
		"a tool held to proposals never runs alone",
	)

	page, err := svc.ListToolPolicies(
		t.Context(),
		&agenttoolports.ListAgentToolPoliciesRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	views := make(map[string]agenttoolports.AgentToolPolicyView, len(page.Edges))
	for _, edge := range page.Edges {
		views[edge.View.Policy.Name] = edge.View
	}
	assert.Equal(t, agent.TierPropose, views["dispatch_assign_move"].Policy.MaxTier)
	require.NotNil(t, views["dispatch_assign_move"].Declared)
	assert.Equal(t, agent.TierAutoExecute, views["dispatch_assign_move"].Declared.MaxTier)
	assert.Same(t, held, views["dispatch_assign_move"].Override)
	assert.Same(
		t,
		reset,
		views["customer_send_email"].Override,
		"a reset rule keeps its version for the next edit",
	)
	assert.Equal(t, agent.TierAutoExecute, views["customer_send_email"].Policy.MaxTier)
	assert.Nil(t, views["shipment_get"].Override)

	assessed := svc.Assess(t.Context(), &agenttoolports.AssessAgentSafetyRequest{
		Subject: &agenttoolports.AgentSafetySubject{
			Agent:   desk,
			Control: &agentcontrol.AgentControl{},
		},
	})
	for _, tool := range assessed {
		if tool.PolicyName == "dispatch_assign_move" {
			assert.Equal(t, agenttooltrust.AutonomyProposeOnly, tool.Clean.Answer)
			require.NotNil(t, tool.Policy)
			assert.Same(t, held, tool.Policy.Override)
		}
	}

	impacts, err := svc.RuleImpact(t.Context(), &agenttoolports.RuleImpactRequest{
		TenantInfo: tenantInfo,
		ToolName:   "customer_send_email",
		Before: policy(
			"customer_send_email",
			agenttooltrust.ToolKindAction,
			agent.EgressExternalRecipient,
		),
		After: func() agenttoolports.ToolPolicy {
			p := policy(
				"customer_send_email",
				agenttooltrust.ToolKindAction,
				agent.EgressExternalRecipient,
			)
			p.MaxTier, p.DefaultTier = agent.TierPropose, agent.TierPropose
			return p
		}(),
	})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	assert.Equal(t, desk.ID, impacts[0].AgentID)
	assert.NotEqual(t, impacts[0].Before, impacts[0].After)
	assert.Equal(t, agenttooltrust.AutonomyProposeOnly, impacts[0].After)
}
