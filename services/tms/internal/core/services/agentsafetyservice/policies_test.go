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
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
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

func eq(field string, value any) domaintypes.FieldFilter {
	return domaintypes.FieldFilter{Field: field, Operator: dbtype.OpEqual, Value: value}
}

func listRules(
	t *testing.T,
	svc *Service,
	table memtable.Request,
) *services.AgentToolPolicyPage {
	t.Helper()

	page, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		Table: table,
	})
	require.NoError(t, err)

	return page
}

/*
The catalog is in memory, so the connection pages it there: ordered by name
whatever order the catalog was built in, each page picking up where the one
before stopped, and nothing repeated or skipped from the first page to the
last.
*/
func TestListToolPoliciesPagesByNameWithoutRepeatingOrSkipping(t *testing.T) {
	t.Parallel()

	svc := manyPoliciesService(132, &fakeDefinitions{})

	seen := make([]string, 0, 132)
	after := ""
	pages := 0
	for {
		page := listRules(t, svc, memtable.Request{First: 25, After: after})
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

	page := listRules(t, svc, memtable.Request{})
	assert.Len(t, page.Edges, pagination.DefaultLimit)
	assert.True(t, page.HasNextPage)

	page = listRules(t, svc, memtable.Request{First: 10_000})
	assert.Len(t, page.Edges, pagination.MaxLimit)
	assert.True(t, page.HasNextPage)
}

// The count is taken only when the caller selected it, and covers every match.
func TestListToolPoliciesCountsOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	svc := manyPoliciesService(132, &fakeDefinitions{})

	first := listRules(t, svc, memtable.Request{First: 25})
	assert.Nil(t, first.TotalCount)

	counted := listRules(t, svc, memtable.Request{
		First:             25,
		After:             first.Edges[24].Cursor,
		IncludeTotalCount: true,
	})
	require.NotNil(t, counted.TotalCount)
	assert.Equal(t, 132, *counted.TotalCount, "the count covers the pages already read")
	assert.Equal(t, "tool_025", counted.Edges[0].View.Policy.Name)

	narrowed := listRules(t, svc, memtable.Request{
		First:             25,
		Query:             "TOOL_01",
		IncludeTotalCount: true,
	})
	require.NotNil(t, narrowed.TotalCount)
	assert.Equal(t, 10, *narrowed.TotalCount)
	assert.False(t, narrowed.HasNextPage)
}

/*
The table sends the GraphQL enum names the client knows (CustomerVisible,
Query); the catalog holds the domain values (customer_visible, query). Both
spellings name the same class.
*/
func TestListToolPoliciesFiltersByClassResourceKindAndSearch(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})
	list := func(table memtable.Request) []string {
		t.Helper()
		table.IncludeTotalCount = true
		page := listRules(t, svc, table)
		require.NotNil(t, page.TotalCount)
		assert.Len(t, page.Edges, *page.TotalCount)

		return pageNames(page)
	}

	assert.Equal(t,
		[]string{"add_home_widget", "assign_move", "email_customer", "get_shipment"},
		list(memtable.Request{}),
	)
	assert.Equal(t, []string{"email_customer"}, list(memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq("egress", "ExternalRecipient")},
	}))
	assert.Equal(t, []string{"email_customer"}, list(memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq("egress", "external_recipient")},
	}))
	assert.Equal(t, []string{"get_shipment"}, list(memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq("kind", "Query")},
	}))
	assert.Equal(t, []string{"add_home_widget"}, list(memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq("resource", GeneralResource)},
	}))
	assert.Equal(t,
		[]string{"assign_move", "email_customer", "get_shipment"},
		list(memtable.Request{
			FieldFilters: []domaintypes.FieldFilter{
				eq("resource", permission.ResourceShipment.String()),
			},
		}),
	)
	assert.Equal(t, []string{"email_customer"}, list(memtable.Request{
		Query: "  Email Cust ",
	}))
	assert.Empty(t, list(memtable.Request{
		Query:        "email",
		FieldFilters: []domaintypes.FieldFilter{eq("egress", "None")},
	}))
	assert.Equal(t,
		[]string{"email_customer", "get_shipment"},
		list(memtable.Request{
			FieldFilters: []domaintypes.FieldFilter{{
				Field:    "egress",
				Operator: dbtype.OpIn,
				Value:    []any{"None", "ExternalRecipient"},
			}},
		}),
	)
}

// Filters in one group are alternatives; groups and plain filters all hold.
func TestListToolPoliciesOrsWithinAGroupAndAndsAcrossThem(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	page := listRules(t, svc, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq("kind", "Action")},
		FilterGroups: []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
			eq("egress", "Personal"),
			eq("egress", "ExternalRecipient"),
		}}},
	})
	assert.Equal(t, []string{"add_home_widget", "email_customer"}, pageNames(page))
}

func TestListToolPoliciesSortsByClassAndTierThenName(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	byClass := listRules(t, svc, memtable.Request{
		Sort: []domaintypes.SortField{{Field: "egress", Direction: dbtype.SortDirectionDesc}},
	})
	assert.Equal(t,
		[]string{"email_customer", "assign_move", "add_home_widget", "get_shipment"},
		pageNames(byClass),
	)

	byTier := listRules(t, svc, memtable.Request{
		Sort: []domaintypes.SortField{{Field: "maxTier", Direction: dbtype.SortDirectionAsc}},
	})
	assert.Equal(t, "email_customer", pageNames(byTier)[0],
		"work that leaves the organization is held to approval, the lowest tier here")
}

/*
Runs without a person is answered over every agent: a tool that changes
something is kept when at least one agent may run it unattended, and a
lookup is never counted, since reading changes nothing. The survey is read
only when the filter, a sort or the column asks for it.
*/
func TestListToolPoliciesFiltersByWhatRunsWithoutAPerson(t *testing.T) {
	t.Parallel()

	defs := &fakeDefinitions{all: []*agentdefinition.Definition{
		agentWith("Dispatcher", "get_shipment", "assign_move", "email_customer"),
	}}
	svc := surveyService(defs, &fakeTrust{})

	page := listRules(t, svc, memtable.Request{
		FieldFilters:      []domaintypes.FieldFilter{eq(FieldRunsWithoutPerson, true)},
		IncludeTotalCount: true,
	})
	assert.Equal(t, []string{"assign_move"}, pageNames(page))
	assert.Equal(t, 1, *page.TotalCount)
	require.NotNil(t, page.Edges[0].RunsWithoutPerson)
	assert.True(t, *page.Edges[0].RunsWithoutPerson)

	page = listRules(t, svc, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{eq(FieldRunsWithoutPerson, "false")},
	})
	assert.Equal(t,
		[]string{"add_home_widget", "email_customer", "get_shipment"},
		pageNames(page),
	)

	listed := defs.pages
	unasked := listRules(t, svc, memtable.Request{})
	assert.Equal(t, listed, defs.pages, "no agent is read when nobody asked")
	assert.Nil(t, unasked.Edges[0].RunsWithoutPerson)

	shown, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		WithAttendance: true,
	})
	require.NoError(t, err)
	for _, edge := range shown.Edges {
		require.NotNil(t, edge.RunsWithoutPerson, edge.View.Policy.Name)
		assert.Equal(t, edge.View.Policy.Name == "assign_move", *edge.RunsWithoutPerson)
	}
}

func TestListToolPoliciesRejectsABadCursorAndBadFilters(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	for name, table := range map[string]memtable.Request{
		"garbage cursor": {After: "%%%"},
		"foreign cursor": {After: pagination.EncodeOffsetCursor("shipment", 2)},
		"unknown class": {
			FieldFilters: []domaintypes.FieldFilter{eq("egress", "everywhere")},
		},
		"unknown kind": {FieldFilters: []domaintypes.FieldFilter{eq("kind", "spell")}},
		"unknown field": {
			FieldFilters: []domaintypes.FieldFilter{eq("rationale", "x")},
		},
		"bad operator": {FieldFilters: []domaintypes.FieldFilter{{
			Field:    "egress",
			Operator: dbtype.OpGreaterThan,
			Value:    "None",
		}}},
		"unsortable": {
			Sort: []domaintypes.SortField{{Field: "rationale", Direction: dbtype.SortDirectionAsc}},
		},
		"bad direction": {
			Sort: []domaintypes.SortField{{Field: "name", Direction: "sideways"}},
		},
		"search too long": {Query: string(make([]byte, memtable.MaxQueryLength+1))},
	} {
		_, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
			Table: table,
		})
		require.Error(t, err, name)
	}

	_, err := svc.ListToolPolicies(t.Context(), &services.ListAgentToolPoliciesRequest{
		Table: memtable.Request{After: "%%%"},
	})
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "after", validation.Field)
}

func TestListToolPoliciesPastTheLastRuleIsAnEmptyPage(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})
	page := listRules(t, svc, memtable.Request{
		After: pagination.EncodeOffsetCursor(toolRuleCursorScope, 40),
	})
	assert.Empty(t, page.Edges)
	assert.False(t, page.HasNextPage)
}

/*
Every tool every compared agent holds is one list: ordered by agent, then
what runs without a person first, and filtered on the answers the same way
the rules are, including the clean and tainted answers.
*/
func TestListAgentToolsListsEveryPickedAgentsTools(t *testing.T) {
	t.Parallel()

	dispatcher := agentWith("Dispatcher", "get_shipment", "email_customer")
	auditor := agentWith("Auditor", "assign_move")
	auditor.SimulationMode = true
	defs := &fakeDefinitions{all: []*agentdefinition.Definition{dispatcher, auditor}}
	svc := surveyService(defs, &fakeTrust{})

	list := func(table memtable.Request) *services.AgentToolSafetyPage {
		t.Helper()
		page, err := svc.ListAgentTools(t.Context(), &services.ListAgentToolSafetyRequest{
			AgentIDs: []pulid.ID{dispatcher.ID, auditor.ID},
			Table:    table,
		})
		require.NoError(t, err)

		return page
	}
	rows := func(page *services.AgentToolSafetyPage) []string {
		out := make([]string, 0, len(page.Edges))
		for _, edge := range page.Edges {
			out = append(out, edge.Node.AgentName+"/"+edge.Node.PolicyName)
		}

		return out
	}

	all := list(memtable.Request{IncludeTotalCount: true})
	assert.Equal(t,
		[]string{"Auditor/assign_move", "Dispatcher/get_shipment", "Dispatcher/email_customer"},
		rows(all),
	)
	require.NotNil(t, all.TotalCount)
	assert.Equal(t, 3, *all.TotalCount)
	assert.Equal(t, dispatcher.ID.String()+":get_shipment", all.Edges[1].Node.RowID())

	assert.Equal(t,
		[]string{"Dispatcher/get_shipment", "Dispatcher/email_customer"},
		rows(list(memtable.Request{
			FieldFilters: []domaintypes.FieldFilter{eq("agentId", dispatcher.ID.String())},
		})),
	)
	assert.Equal(t,
		[]string{"Dispatcher/email_customer"},
		rows(list(memtable.Request{
			FieldFilters: []domaintypes.FieldFilter{eq("clean", "NEEDS_APPROVAL")},
		})),
	)
	assert.Equal(t,
		[]string{"Auditor/assign_move"},
		rows(list(memtable.Request{
			FieldFilters: []domaintypes.FieldFilter{eq("heldBy", "simulation_mode")},
		})),
	)
}

func TestListAgentToolsNamesBetweenOneAndTenAgents(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	_, err := svc.ListAgentTools(t.Context(), &services.ListAgentToolSafetyRequest{})
	require.Error(t, err)

	tooMany := make([]pulid.ID, 0, MaxComparedAgents+1)
	for range MaxComparedAgents + 1 {
		tooMany = append(tooMany, pulid.MustNew("agdef_"))
	}
	_, err = svc.ListAgentTools(t.Context(), &services.ListAgentToolSafetyRequest{
		AgentIDs: tooMany,
	})
	require.Error(t, err)

	page, err := svc.ListAgentTools(t.Context(), &services.ListAgentToolSafetyRequest{
		AgentIDs: []pulid.ID{pulid.MustNew("agdef_")},
		Table:    memtable.Request{IncludeTotalCount: true},
	})
	require.NoError(t, err, "an agent that no longer exists holds nothing")
	assert.Empty(t, page.Edges)
	require.NotNil(t, page.TotalCount)
	assert.Zero(t, *page.TotalCount)
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
