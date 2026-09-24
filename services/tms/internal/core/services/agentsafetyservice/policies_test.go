package agentsafetyservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentaccessservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTrust struct {
	rows  map[pulid.ID][]*agent.ToolTrust
	calls int
}

func (f *fakeTrust) ListByDefinitionIDs(
	_ context.Context,
	req repositories.ListToolTrustByDefinitionsRequest,
) (map[pulid.ID][]*agent.ToolTrust, error) {
	f.calls++
	out := make(map[pulid.ID][]*agent.ToolTrust, len(req.AgentDefinitionIDs))
	for _, id := range req.AgentDefinitionIDs {
		if rows, ok := f.rows[id]; ok {
			out[id] = rows
		}
	}

	return out, nil
}

func manyPoliciesService(count int, defs *fakeDefinitions) *Service {
	policies := make([]services.ToolPolicy, 0, count)
	for idx := count - 1; idx >= 0; idx-- {
		policies = append(policies, policy(
			fmt.Sprintf("tool_%03d", idx),
			agent.ToolKindAction,
			agent.EgressInternal,
		))
	}

	return newService(serviceDeps{
		policies:    newFakePolicies(policies...),
		holder:      fakeHolder{},
		definitions: defs,
		controls:    &fakeControls{control: &tenant.AgentControl{}},
		trust:       &fakeTrust{},
		sensitive:   agentaccessservice.LeavesOrganizationRule(),
	})
}

func surveyService(defs *fakeDefinitions, trust *fakeTrust) *Service {
	lookup := policy("get_shipment", agent.ToolKindQuery, agent.EgressNone)
	lookup.Effect = agent.ToolEffectLookup
	personal := policy("add_home_widget", agent.ToolKindAction, agent.EgressPersonal)
	personal.Scope = agent.ToolScopeSelf

	return newService(serviceDeps{
		policies: newFakePolicies(
			lookup,
			policy("assign_move", agent.ToolKindAction, agent.EgressInternal),
			policy("email_customer", agent.ToolKindAction, agent.EgressExternalRecipient),
			personal,
		),
		holder:      fakeHolder{},
		definitions: defs,
		controls:    &fakeControls{control: &tenant.AgentControl{}},
		trust:       trust,
		sensitive:   agentaccessservice.LeavesOrganizationRule(),
	})
}

func pageNames(page *services.AgentToolPolicyPage) []string {
	out := make([]string, 0, len(page.Edges))
	for _, edge := range page.Edges {
		out = append(out, edge.View.Policy.Name)
	}

	return out
}

/*
The catalog is in memory, so the connection pages it there: ordered by name
whatever order the catalog was built in, each page picking up after the last
name of the one before, and nothing repeated or skipped from the first page
to the last.
*/
func TestListToolPoliciesPagesByNameWithoutRepeatingOrSkipping(t *testing.T) {
	t.Parallel()

	svc := manyPoliciesService(132, &fakeDefinitions{})

	seen := make([]string, 0, 132)
	after := ""
	pages := 0
	for {
		page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
			First: 25,
			After: after,
		})
		require.NoError(t, err)
		pages++
		seen = append(seen, pageNames(page)...)
		if !page.HasNextPage {
			break
		}
		after = page.Edges[len(page.Edges)-1].Cursor
	}

	assert.Equal(t, 6, pages)
	require.Len(t, seen, 132)
	for idx, name := range seen {
		assert.Equal(t, fmt.Sprintf("tool_%03d", idx), name)
	}
}

func TestListToolPoliciesDefaultsAndClampsThePageSize(t *testing.T) {
	t.Parallel()

	svc := manyPoliciesService(132, &fakeDefinitions{})

	page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{})
	require.NoError(t, err)
	assert.Len(t, page.Edges, DefaultToolPolicyPageSize)
	assert.True(t, page.HasNextPage)

	page, err = svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		First: 10_000,
	})
	require.NoError(t, err)
	assert.Len(t, page.Edges, pagination.MaxLimit)
	assert.True(t, page.HasNextPage)
}

// A count walks every rule, so it is taken only when the caller selected it.
func TestListToolPoliciesCountsOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	svc := manyPoliciesService(132, &fakeDefinitions{})

	first, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		First: 25,
	})
	require.NoError(t, err)
	assert.Nil(t, first.TotalCount)

	counted, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		First:             25,
		After:             first.Edges[24].Cursor,
		IncludeTotalCount: true,
	})
	require.NoError(t, err)
	require.NotNil(t, counted.TotalCount)
	assert.Equal(t, 132, *counted.TotalCount, "the count covers the pages already read")
	assert.Equal(t, "tool_025", counted.Edges[0].View.Policy.Name)

	narrowed, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		First:             25,
		Query:             "TOOL_01",
		IncludeTotalCount: true,
	})
	require.NoError(t, err)
	require.NotNil(t, narrowed.TotalCount)
	assert.Equal(t, 10, *narrowed.TotalCount)
	assert.False(t, narrowed.HasNextPage)
}

func TestListToolPoliciesFiltersByClassResourceKindAndSearch(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})
	list := func(req *services.ListAgentToolPoliciesRequest) []string {
		t.Helper()
		req.IncludeTotalCount = true
		page, err := svc.ListToolPolicies(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, page.TotalCount)
		assert.Len(t, page.Edges, *page.TotalCount)

		return pageNames(page)
	}

	assert.Equal(t,
		[]string{"add_home_widget", "assign_move", "email_customer", "get_shipment"},
		list(&services.ListAgentToolPoliciesRequest{}),
	)
	assert.Equal(t, []string{"email_customer"}, list(&services.ListAgentToolPoliciesRequest{
		Egress: agent.EgressExternalRecipient,
	}))
	assert.Equal(t, []string{"get_shipment"}, list(&services.ListAgentToolPoliciesRequest{
		Kind: agent.ToolKindQuery,
	}))
	assert.Equal(t, []string{"add_home_widget"}, list(&services.ListAgentToolPoliciesRequest{
		Resource: GeneralResource,
	}))
	assert.Equal(t,
		[]string{"assign_move", "email_customer", "get_shipment"},
		list(&services.ListAgentToolPoliciesRequest{
			Resource: permission.ResourceShipment.String(),
		}),
	)
	assert.Equal(t, []string{"email_customer"}, list(&services.ListAgentToolPoliciesRequest{
		Query: "  Email Cust ",
	}))
	assert.Empty(t, list(&services.ListAgentToolPoliciesRequest{
		Query:  "email",
		Egress: agent.EgressNone,
	}))
}

/*
Runs without a person is answered over every agent: a tool that changes
something is kept when at least one agent may run it unattended, and a
lookup is never counted, since reading changes nothing.
*/
func TestListToolPoliciesFiltersByWhatRunsWithoutAPerson(t *testing.T) {
	t.Parallel()

	defs := &fakeDefinitions{all: []*agentdefinition.Definition{
		agentWith("Dispatcher", "get_shipment", "assign_move", "email_customer"),
	}}
	svc := surveyService(defs, &fakeTrust{})

	alone := true
	page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		RunsWithoutPerson: &alone,
		IncludeTotalCount: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"assign_move"}, pageNames(page))
	assert.Equal(t, 1, *page.TotalCount)

	attended := false
	page, err = svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		RunsWithoutPerson: &attended,
	})
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"add_home_widget", "email_customer", "get_shipment"},
		pageNames(page),
	)
}

func TestListToolPoliciesRejectsABadCursorAndBadFilters(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	for name, req := range map[string]*services.ListAgentToolPoliciesRequest{
		"garbage cursor":  {After: "%%%"},
		"foreign cursor":  {After: pagination.EncodeKeyCursor("shipment", "assign_move")},
		"unknown class":   {Egress: agent.EgressClass("everywhere")},
		"unknown kind":    {Kind: agent.ToolKind("spell")},
		"search too long": {Query: string(make([]byte, MaxToolPolicyQueryLength+1))},
	} {
		_, err := svc.ListToolPolicies(t.Context(), req)
		require.Error(t, err, name)
	}

	_, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		After: "%%%",
	})
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "after", validation.Field)
}

// A cursor for a tool that has since left the catalog still resumes in order.
func TestListToolPoliciesResumesAfterARetiredTool(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})
	page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		After: pagination.EncodeKeyCursor(toolPolicyCursorScope, "b_retired_tool"),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"email_customer", "get_shipment"}, pageNames(page))
}

func TestSummaryCountsOverEveryToolAndAgent(t *testing.T) {
	t.Parallel()

	restricted := agentWith("Back office", "assign_move", "email_customer")
	restricted.AccessMode = agentdefinition.AccessRoles
	simulated := agentWith("Rehearsal", "email_customer")
	simulated.SimulationMode = true
	defs := &fakeDefinitions{all: []*agentdefinition.Definition{
		agentWith("Customer desk", "get_shipment", "email_customer"),
		restricted,
		simulated,
	}}
	trust := &fakeTrust{}
	svc := surveyService(defs, trust)

	summary, err := svc.Summary(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, 4, summary.ToolCount)
	assert.Equal(t, 1, summary.RunWithoutPerson, "assign_move runs unattended on Back office")
	assert.Equal(t, 1, summary.LeaveOrganization)
	assert.Equal(t, 2, summary.OpenWithSensitive)
	assert.Equal(t, []string{GeneralResource, permission.ResourceShipment.String()},
		summary.Resources)
	assert.Equal(t, 1, trust.calls, "trust is read for a page of agents at once")
}

func TestSummaryWithNoAgentsReadsNoControlOrTrust(t *testing.T) {
	t.Parallel()

	trust := &fakeTrust{}
	svc := surveyService(&fakeDefinitions{}, trust)

	summary, err := svc.Summary(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Zero(t, summary.RunWithoutPerson)
	assert.Zero(t, summary.OpenWithSensitive)
	assert.Zero(t, trust.calls)
}

func TestToolPolicyLooksUpOneRuleByName(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	view, ok := svc.ToolPolicy("email_customer")
	require.True(t, ok)
	assert.Equal(t, "Email customer", view.Title)

	_, ok = svc.ToolPolicy("unregistered_tool")
	assert.False(t, ok)
}
