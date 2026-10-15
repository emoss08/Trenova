package agentruntime

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func recommendationRuntime(
	t *testing.T,
	permissions *agenttest.StubPermissions,
	agents ...*agentdefinition.Definition,
) *Service {
	t.Helper()

	query := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		describedTool("list_shipments", "List shipments by status, billing state or dates."),
		describedTool("list_customers", "List customers by status, code or city."),
		&agenttest.StubQueryTool{
			ToolName: "list_invoices",
			Desc:     "List invoices by status or due date.",
			Resource: permission.ResourceInvoice,
		},
		&agenttest.StubQueryTool{
			ToolName: "list_expiring_credentials",
			Desc:     "Worker credentials falling due: medical cards, licences, hazmat.",
			Resource: permission.ResourceWorker,
		},
		&agenttest.StubQueryTool{
			ToolName: "list_workers",
			Desc:     "List workers (drivers) by status or employment type.",
			Resource: permission.ResourceWorker,
		},
	}}
	action := &stubActionRegistry{}

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byID[definition.ID] = definition
	}

	return &Service{
		logger:      zap.NewNop(),
		queryTools:  query,
		actionTools: action,
		permissions: permissions,
		definitions: &listedDefinitions{byID: byID},
		catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		}),
	}
}

type recordingDefinitions struct {
	listedDefinitions

	last *agentdefinitionports.ListAgentDefinitionRequest
}

func (r *recordingDefinitions) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	r.last = req

	return r.listedDefinitions.List(ctx, req)
}

func agentHolding(name string, tools ...string) *agentdefinition.Definition {
	definition := chatAgent(name)
	definition.ToolNames = tools

	return definition
}

func recommend(
	t *testing.T,
	service *Service,
	text string,
	current pulid.ID,
) *agentports.AgentRecommendation {
	t.Helper()

	recommendation, err := service.RecommendAgent(t.Context(), &agentports.AgentRecommendationRequest{
		Actor:          testActor(),
		Text:           text,
		CurrentAgentID: current,
	})
	require.NoError(t, err)

	return recommendation
}

func TestRecommendAgent_PicksTheAgentHoldingTheToolTheQuestionNeeds(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "list_shipments", "list_customers")
	compliance := agentHolding("Compliance assistant", "list_expiring_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance)

	recommendation := recommend(t, service, "when does the driver's medical card expire?", dispatch.ID)

	require.NotNil(t, recommendation)
	assert.Equal(t, compliance.ID, recommendation.Agent.ID)
	assert.False(t, recommendation.Current)
	assert.Equal(t, []string{"list_expiring_credentials"}, recommendation.Tools)
}

func TestRecommendAgent_KeepsTheCurrentAgentWhenItFitsAsWellAsAnyOther(t *testing.T) {
	t.Parallel()

	first := agentHolding("Billing assistant", "list_invoices")
	second := agentHolding("Accounts receivable", "list_invoices")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, first, second)

	recommendation := recommend(t, service, "which invoices are past due?", second.ID)

	require.NotNil(t, recommendation)
	assert.Equal(t, second.ID, recommendation.Agent.ID)
	assert.True(t, recommendation.Current)
}

func TestRecommendAgent_NeverRecommendsAnAgentThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	restricted := agentHolding("Compliance assistant", "list_expiring_credentials")
	restricted.AccessMode = agentdefinition.AccessRoles
	open := agentHolding("Fleet helper", "list_workers")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, restricted, open)

	recommendation := recommend(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, recommendation)
	assert.Equal(t, open.ID, recommendation.Agent.ID,
		"an agent restricted to roles the person does not hold is never offered")

	service.permissions = &agenttest.StubPermissions{GrantedAgents: []pulid.ID{restricted.ID}}
	recommendation = recommend(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, recommendation)
	assert.Equal(t, restricted.ID, recommendation.Agent.ID,
		"granting one of the person's roles the agent makes it eligible")
}

func TestRecommendAgent_IgnoresToolsThePersonMayNotRun(t *testing.T) {
	t.Parallel()

	compliance := agentHolding("Compliance assistant", "list_expiring_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{
		Denied: map[string]bool{permission.ResourceWorker.String() + ":" + string(permission.OpRead): true},
	}, compliance)

	assert.Nil(t, recommend(t, service, "driver medical card expiry", pulid.Nil),
		"an agent is not recommended for a tool the person could not use through it")
}

func TestRecommendAgent_RecommendsNobodyToAPersonWhoMayNotStartAConversation(t *testing.T) {
	t.Parallel()

	compliance := agentHolding("Compliance assistant", "list_expiring_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{
		Denied: map[string]bool{
			permission.ResourceAssistant.String() + ":" + string(permission.OpCreate): true,
		},
	}, compliance)

	assert.Nil(t, recommend(t, service, "driver medical card expiry", pulid.Nil))
}

func TestRecommendAgent_LeavesOutDisabledAndUnconversationalAgentsByAskingForChatAgents(
	t *testing.T,
) {
	t.Parallel()

	disabled := agentHolding("Compliance assistant", "list_expiring_credentials")
	disabled.Enabled = false
	definitions := &recordingDefinitions{
		listedDefinitions: listedDefinitions{
			byID: map[pulid.ID]*agentdefinition.Definition{disabled.ID: disabled},
		},
	}
	service := recommendationRuntime(t, &agenttest.StubPermissions{})
	service.definitions = definitions

	_ = recommend(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, definitions.last)
	assert.True(t, definitions.last.EnabledOnly)
	assert.True(t, definitions.last.ChatOnly)
	assert.NotNil(t, definitions.last.Audience)
}

func TestRecommendAgent_PrefersTheFocusedAgentOnATie(t *testing.T) {
	t.Parallel()

	generalist := agentHolding("Operations helper", "list_invoices", "list_shipments", "list_customers")
	specialist := agentHolding("Billing assistant", "list_invoices")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, generalist, specialist)

	recommendation := recommend(t, service, "which invoices are past due?", pulid.Nil)

	require.NotNil(t, recommendation)
	assert.Equal(t, specialist.ID, recommendation.Agent.ID)
}

func TestRecommendAgent_RecommendsNobodyWhenNothingMatches(t *testing.T) {
	t.Parallel()

	billing := agentHolding("Billing assistant", "list_invoices")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)

	assert.Nil(t, recommend(t, service, "zzqx flarp", pulid.Nil))
	assert.Nil(t, recommend(t, service, "   ", pulid.Nil))
}

func TestRecommendAgent_RecommendsNothingToAnythingButAPerson(t *testing.T) {
	t.Parallel()

	billing := agentHolding("Billing assistant", "list_invoices")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)
	actor := testActor()
	actor.PrincipalType = permissionports.PrincipalTypeAPIKey

	recommendation, err := service.RecommendAgent(t.Context(), &agentports.AgentRecommendationRequest{
		Actor: actor,
		Text:  "which invoices are past due?",
	})

	require.NoError(t, err)
	assert.Nil(t, recommendation)
}

func TestRecommendAgent_RefusesAQuestionPastTheLimit(t *testing.T) {
	t.Parallel()

	billing := agentHolding("Billing assistant", "list_invoices")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)

	recommendation, err := service.RecommendAgent(t.Context(), &agentports.AgentRecommendationRequest{
		Actor: testActor(),
		Text:  strings.Repeat("a", agentports.MaxAgentRecommendationTextRunes+1),
	})

	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "text", validation.Field)
	assert.Nil(t, recommendation)
}

func TestRecommendAgent_NamesAtMostThreeMatchedTools(t *testing.T) {
	t.Parallel()

	wide := agentHolding(
		"Operations helper",
		"list_invoices",
		"list_shipments",
		"list_customers",
		"list_workers",
		"list_expiring_credentials",
	)
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, wide)

	recommendation := recommend(
		t,
		service,
		"list invoices shipments customers workers and expiring credentials by status",
		pulid.Nil,
	)

	require.NotNil(t, recommendation)
	assert.LessOrEqual(t, len(recommendation.Tools), agentports.MaxAgentRecommendationTools)
}

func ranking(pairs ...any) []agenttoolcatalog.Match {
	out := make([]agenttoolcatalog.Match, 0, len(pairs)/2)
	for idx := 0; idx+1 < len(pairs); idx += 2 {
		out = append(out, agenttoolcatalog.Match{
			Name:  pairs[idx].(string),
			Score: float64(pairs[idx+1].(int)),
		})
	}

	return out
}

func names(ranked []agenttoolcatalog.Match) []string {
	out := make([]string, 0, len(ranked))
	for _, match := range ranked {
		out = append(out, match.Name)
	}

	return out
}

func heldSets(tools ...[]string) []map[string]struct{} {
	sets := make([]map[string]struct{}, 0, len(tools))
	for _, names := range tools {
		set := make(map[string]struct{}, len(names))
		for _, name := range names {
			set[name] = struct{}{}
		}
		sets = append(sets, set)
	}

	return sets
}

func TestBestCandidate_KeepsTheCurrentAgentWhenItHoldsTheBestMatch(t *testing.T) {
	t.Parallel()

	current := agentHolding("Billing assistant", "list_invoices")
	wide := agentHolding("Operations helper", "list_customers", "list_shipments", "list_workers")
	ranked := ranking("list_invoices", 4, "list_customers", 3, "list_shipments", 2, "list_workers", 1)

	best := bestCandidate(
		[]*agentdefinition.Definition{current, wide},
		heldSets(current.ToolNames, wide.ToolNames),
		names(ranked),
		ranked,
		current.ID,
	)

	require.NotNil(t, best)
	assert.Equal(t, current.ID, best.definition.ID,
		"an agent that scores higher on lesser matches does not manage the question better")
	assert.True(t, best.current)
}

func TestBestCandidate_SuggestsOnlyAnAgentHoldingTheBestMatch(t *testing.T) {
	t.Parallel()

	current := agentHolding("Dispatch helper", "list_tractors")
	wide := agentHolding("Operations helper", "list_customers", "list_shipments", "list_workers")
	billing := agentHolding("Billing assistant", "list_invoices")
	ranked := ranking("list_invoices", 4, "list_customers", 3, "list_shipments", 2, "list_workers", 1)

	best := bestCandidate(
		[]*agentdefinition.Definition{current, wide, billing},
		heldSets(current.ToolNames, wide.ToolNames, billing.ToolNames),
		names(ranked),
		ranked,
		current.ID,
	)

	require.NotNil(t, best)
	assert.Equal(t, billing.ID, best.definition.ID)
	assert.False(t, best.current)
}

func TestRecommendAgent_SuggestsNobodyElseOnceThePersonSwitchedToTheSuggestion(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "list_shipments", "list_customers")
	compliance := agentHolding("Compliance assistant", "list_expiring_credentials")
	fleet := agentHolding("Fleet helper", "list_expiring_credentials", "list_workers")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance, fleet)
	question := "when does the driver's medical card expire?"

	first := recommend(t, service, question, dispatch.ID)
	require.NotNil(t, first)
	require.False(t, first.Current)

	again := recommend(t, service, question, first.Agent.ID)
	require.NotNil(t, again)
	assert.Equal(t, first.Agent.ID, again.Agent.ID)
	assert.True(t, again.Current, "the agent switched to is kept, not traded for another holder")
}

func TestBestCandidate_KeepsTheCurrentAgentWhenItHoldsAToolTiedForTheBestMatch(t *testing.T) {
	t.Parallel()

	current := agentHolding("Dispatch reader", "get_shipment", "list_shipments")
	writer := agentHolding("Shipment builder", "create_shipment")
	ranked := ranking("create_shipment", 4, "get_shipment", 4, "list_shipments", 4)

	best := bestCandidate(
		[]*agentdefinition.Definition{writer, current},
		heldSets(writer.ToolNames, current.ToolNames),
		names(ranked),
		ranked,
		current.ID,
	)

	require.NotNil(t, best)
	assert.Equal(t, current.ID, best.definition.ID,
		"a tool tied with the first match is as good a match; catalog order does not decide")
	assert.True(t, best.current)
}

func TestBestCandidate_WeighsToolsTiedOnScoreEqually(t *testing.T) {
	t.Parallel()

	current := agentHolding("Fleet helper", "list_tractors")
	builder := agentHolding("Shipment builder", "create_shipment")
	reader := agentHolding("Dispatch reader", "get_shipment", "list_shipments")
	ranked := ranking("create_shipment", 4, "get_shipment", 4, "list_shipments", 4, "list_tractors", 1)

	best := bestCandidate(
		[]*agentdefinition.Definition{builder, reader, current},
		heldSets(builder.ToolNames, reader.ToolNames, current.ToolNames),
		names(ranked),
		ranked,
		current.ID,
	)

	require.NotNil(t, best)
	assert.Equal(t, reader.ID, best.definition.ID,
		"two tools tied at the top outweigh one, whichever the catalog lists first")
}

func TestRankWeights_GivesTiedScoresTheSameWeight(t *testing.T) {
	t.Parallel()

	weights := rankWeights(ranking("a", 4, "b", 4, "c", 2, "d", 2, "e", 1))

	assert.Equal(t, []float64{1, 1, 0.5, 0.5, 1.0 / 3}, weights)
}

type pagedDefinitions struct {
	listedDefinitions

	all   []*agentdefinition.Definition
	pages int
}

func (p *pagedDefinitions) List(
	_ context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	p.pages++
	offset := req.Filter.Pagination.Offset
	end := min(offset+req.Filter.Pagination.Limit, len(p.all))
	if offset > len(p.all) {
		offset = len(p.all)
	}

	return &pagination.ListResult[*agentdefinition.Definition]{
		Items: p.all[offset:end],
		Total: len(p.all),
	}, nil
}

func TestRecommendAgent_ReadsPastTheFirstHundredAgents(t *testing.T) {
	t.Parallel()

	all := make([]*agentdefinition.Definition, 0, 150)
	for idx := range 149 {
		all = append(all, agentHolding(fmt.Sprintf("Agent %03d", idx), "list_customers"))
	}
	compliance := agentHolding("Zeta compliance", "list_expiring_credentials")
	all = append(all, compliance)
	definitions := &pagedDefinitions{all: all}
	service := recommendationRuntime(t, &agenttest.StubPermissions{})
	service.definitions = definitions

	recommendation := recommend(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, recommendation)
	assert.Equal(t, compliance.ID, recommendation.Agent.ID)
	assert.Equal(t, 2, definitions.pages)
}
