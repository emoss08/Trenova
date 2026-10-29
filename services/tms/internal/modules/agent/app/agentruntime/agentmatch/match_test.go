package agentmatch

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrecommendationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
) *matchWorld {
	t.Helper()

	return runtimeWithTools(t, permissions, []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "shipment_list",
			Desc:     "List shipments by status, billing state or dates.",
			Resource: permission.ResourceShipment,
		},
		&agenttest.StubQueryTool{
			ToolName: "customer_list",
			Desc:     "List customers by status, code or city.",
			Resource: permission.ResourceCustomer,
		},
		&agenttest.StubQueryTool{
			ToolName: "invoice_list",
			Desc:     "List invoices by status or due date.",
			Resource: permission.ResourceInvoice,
		},
		&agenttest.StubQueryTool{
			ToolName: "worker_list_credentials",
			Desc:     "Worker credentials falling due: medical cards, licences, hazmat.",
			Resource: permission.ResourceWorker,
		},
		&agenttest.StubQueryTool{
			ToolName: "worker_list",
			Desc:     "List workers (drivers) by status or employment type.",
			Resource: permission.ResourceWorker,
		},
	}, agents...)
}

func runtimeWithTools(
	t *testing.T,
	permissions *agenttest.StubPermissions,
	tools []agenttoolports.AgentQueryTool,
	agents ...*agentdefinition.Definition,
) *matchWorld {
	t.Helper()

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byID[definition.ID] = definition
	}

	return &matchWorld{
		query:       &agenttest.StubQueryRegistry{Tools: tools},
		action:      &agenttest.StubActionRegistry{},
		permissions: permissions,
		definitions: &runtimetest.ListedDefinitions{ByID: byID},
	}
}

// matchWorld is what a match test runs against; build reads it into a
// Matcher, so a test may change it between matches.
type matchWorld struct {
	query       *agenttest.StubQueryRegistry
	action      *agenttest.StubActionRegistry
	permissions permissionports.PermissionEngine
	definitions agentdefinitionports.AgentDefinitionRepository
	vectorizer  retrievalports.QueryVectorizer
	vectors     retrievalports.CatalogVectorIndex
	rateLimits  kernelports.RateLimitStore
}

func (w *matchWorld) build() *Matcher {
	return NewMatcher(MatcherParams{
		Logger:         zap.NewNop(),
		QueryTools:     w.query,
		ActionTools:    w.action,
		Permissions:    w.permissions,
		Definitions:    w.definitions,
		Vectorizer:     w.vectorizer,
		CatalogVectors: w.vectors,
		RateLimits:     w.rateLimits,
		Catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  w.query,
			ActionTools: w.action,
		}),
	})
}

type recordingDefinitions struct {
	runtimetest.ListedDefinitions

	last *agentdefinitionports.ListAgentDefinitionRequest
}

func (r *recordingDefinitions) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	r.last = req

	return r.ListedDefinitions.List(ctx, req)
}

func match(
	t *testing.T,
	service *matchWorld,
	text string,
	current pulid.ID,
	contextKinds ...string,
) *agentrecommendationports.AgentMatch {
	t.Helper()

	found, err := service.build().
		MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
			Actor:          runtimetest.Actor(),
			Text:           text,
			CurrentAgentID: current,
			ContextKinds:   contextKinds,
		})
	require.NoError(t, err)
	require.NotNil(t, found)

	return found
}

func toolNames(tools []agentrecommendationports.AgentMatchTool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}

	return out
}

func TestMatchAgent_PicksTheAgentHoldingTheToolTheQuestionNeeds(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "customer_list")
	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance)

	found := match(t, service, "when does the driver's medical card expire?", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome)
	require.NotNil(t, found.Agent)
	assert.Equal(t, compliance.ID, found.Agent.ID)
	assert.Equal(t, []agentrecommendationports.AgentMatchTool{{
		Name:    "worker_list_credentials",
		Summary: "Worker credentials falling due: medical cards, licences, hazmat.",
	}}, found.Tools)
}

func TestMatchAgent_KeepsTheCurrentAgentWhenItFitsAsWellAsAnyOther(t *testing.T) {
	t.Parallel()

	first := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	second := runtimetest.AgentHolding("Accounts receivable", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, first, second)

	found := match(t, service, "which invoices are past due?", second.ID)

	assert.Equal(t, agentsuggestion.MatchCurrentAgent, found.Outcome)
	require.NotNil(t, found.Agent)
	assert.Equal(t, second.ID, found.Agent.ID)
}

func TestMatchAgent_NeverSuggestsAnAgentThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	restricted := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	restricted.AccessMode = agentdefinition.AccessRoles
	open := runtimetest.AgentHolding("Fleet helper", "worker_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, restricted, open)

	found := match(t, service, "driver medical card expiry", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchNoAgent, found.Outcome,
		"an agent restricted to roles the person does not hold is never offered, "+
			"and the agent holding only a lesser match is not offered in its place")
	assert.Nil(t, found.Agent)
	assert.Equal(t, []string{"worker_list_credentials"}, toolNames(found.Tools))

	service.permissions = &agenttest.StubPermissions{GrantedAgents: []pulid.ID{restricted.ID}}
	found = match(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, found.Agent)
	assert.Equal(t, restricted.ID, found.Agent.ID,
		"granting one of the person's roles the agent makes it eligible")
}

func TestMatchAgent_SaysNoAgentCanHelpWhenThePersonMayNotRunTheTool(t *testing.T) {
	t.Parallel()

	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{
		Denied: map[string]bool{
			permission.ResourceWorker.String() + ":" + string(permission.OpRead): true,
		},
	}, compliance)

	found := match(t, service, "driver medical card expiry", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchNoAgent, found.Outcome,
		"an agent holding a tool the person could not use through it does not count")
	assert.Nil(t, found.Agent)
}

func TestMatchAgent_SaysNoAgentCanHelpWhenNoAgentHoldsTheBestTool(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "customer_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch)

	found := match(t, service, "which invoices are past due?", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchNoAgent, found.Outcome)
	assert.Nil(t, found.Agent)
	assert.Equal(t, []agentrecommendationports.AgentMatchTool{{
		Name:    "invoice_list",
		Summary: "List invoices by status or due date.",
	}}, found.Tools, "the tools that would answer it are named for whoever can add them")
}

func TestMatchAgent_SaysNoAgentWhenOnlyALesserToolIsHeld(t *testing.T) {
	t.Parallel()

	ledger := runtimetest.AgentHolding("Ledger helper", "summarize_ledger")
	service := runtimeWithTools(t, &agenttest.StubPermissions{}, []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "invoice_list",
			Desc:     "List invoices by status or due date.",
			Resource: permission.ResourceInvoice,
		},
		&agenttest.StubQueryTool{
			ToolName: "summarize_ledger",
			Desc:     "Summarize the ledger, including invoices.",
			Resource: permission.ResourceInvoice,
		},
	}, ledger)

	found := match(t, service, "which invoices are past due?", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchNoAgent, found.Outcome,
		"an agent holding only a lesser match is not where the question belongs")
	assert.Nil(t, found.Agent)
	assert.Equal(t, []string{"invoice_list"}, toolNames(found.Tools))
}

func TestMatchAgent_FindsTheAgentWhoseToolTiesBeyondTheFirstFewMatches(t *testing.T) {
	t.Parallel()

	tools := make([]agenttoolports.AgentQueryTool, 0, recommendationToolLimit+2)
	for idx := range recommendationToolLimit + 2 {
		tools = append(tools, &agenttest.StubQueryTool{
			ToolName: fmt.Sprintf("invoice_report_%02d", idx),
			Desc:     "Invoice report of invoices past due.",
			Resource: permission.ResourceInvoice,
		})
	}
	last := fmt.Sprintf("invoice_report_%02d", recommendationToolLimit+1)
	billing := runtimetest.AgentHolding("Billing assistant", last)
	service := runtimeWithTools(t, &agenttest.StubPermissions{}, tools, billing)

	found := match(t, service, "which invoices are past due?", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome,
		"a held tool tied with the best matches counts however many tie with it")
	require.NotNil(t, found.Agent)
	assert.Equal(t, billing.ID, found.Agent.ID)
}

func TestMatchAgent_SaysNoAgentWhenEveryHeldToolRanksBelowTheFirstFewMatches(t *testing.T) {
	t.Parallel()

	tools := make([]agenttoolports.AgentQueryTool, 0, recommendationToolLimit+1)
	for idx := range recommendationToolLimit {
		tools = append(tools, &agenttest.StubQueryTool{
			ToolName: fmt.Sprintf("invoice_report_%02d", idx),
			Desc:     "Invoice report of invoices past due.",
			Resource: permission.ResourceInvoice,
		})
	}
	tools = append(tools, &agenttest.StubQueryTool{
		ToolName: "summarize_ledger",
		Desc:     "Summarize the ledger, including invoices.",
		Resource: permission.ResourceInvoice,
	})
	ledger := runtimetest.AgentHolding("Ledger helper", "summarize_ledger")
	service := runtimeWithTools(t, &agenttest.StubPermissions{}, tools, ledger)

	found := match(t, service, "which invoices are past due?", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchNoAgent, found.Outcome,
		"a tool that only touches the question does not make its agent the one for it")
	assert.Nil(t, found.Agent)
}

func TestMatchAgent_MatchesNothingForAPersonWhoMayNotStartAConversation(t *testing.T) {
	t.Parallel()

	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	service := recommendationRuntime(t, &agenttest.StubPermissions{
		Denied: map[string]bool{
			permission.ResourceAssistant.String() + ":" + string(permission.OpCreate): true,
		},
	}, compliance)

	assert.Equal(t, agentsuggestion.MatchNone,
		match(t, service, "driver medical card expiry", pulid.Nil).Outcome)
}

func TestMatchAgent_LeavesOutDisabledAndUnconversationalAgentsByAskingForChatAgents(
	t *testing.T,
) {
	t.Parallel()

	disabled := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	disabled.Enabled = false
	definitions := &recordingDefinitions{
		ListedDefinitions: runtimetest.ListedDefinitions{
			ByID: map[pulid.ID]*agentdefinition.Definition{disabled.ID: disabled},
		},
	}
	service := recommendationRuntime(t, &agenttest.StubPermissions{})
	service.definitions = definitions

	_ = match(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, definitions.last)
	assert.True(t, definitions.last.EnabledOnly)
	assert.True(t, definitions.last.ChatOnly)
	assert.NotNil(t, definitions.last.Audience)
}

func TestMatchAgent_PrefersTheFocusedAgentOnATie(t *testing.T) {
	t.Parallel()

	generalist := runtimetest.AgentHolding(
		"Operations helper",
		"invoice_list",
		"shipment_list",
		"customer_list",
	)
	specialist := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, generalist, specialist)

	found := match(t, service, "which invoices are past due?", pulid.Nil)

	require.NotNil(t, found.Agent)
	assert.Equal(t, specialist.ID, found.Agent.ID)
}

func TestMatchAgent_MatchesNothingWhenNoToolMatches(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)

	assert.Equal(t, agentsuggestion.MatchNone, match(t, service, "zzqx flarp", pulid.Nil).Outcome)
	assert.Equal(t, agentsuggestion.MatchNone, match(t, service, "   ", pulid.Nil).Outcome)
}

func TestMatchAgent_MatchesNothingForAnythingButAPerson(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)
	actor := runtimetest.Actor()
	actor.PrincipalType = permissionports.PrincipalTypeAPIKey

	found, err := service.build().
		MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
			Actor: actor,
			Text:  "which invoices are past due?",
		})

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, agentsuggestion.MatchNone, found.Outcome)
	assert.Nil(t, found.Agent)
}

func TestMatchAgent_RefusesAQuestionPastTheLimit(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)

	found, err := service.build().
		MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
			Actor: runtimetest.Actor(),
			Text:  strings.Repeat("a", agentrecommendationports.MaxAgentRecommendationTextRunes+1),
		})

	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "text", validation.Field)
	assert.Nil(t, found)
}

func TestMatchAgent_NamesAtMostThreeMatchedTools(t *testing.T) {
	t.Parallel()

	wide := runtimetest.AgentHolding(
		"Operations helper",
		"invoice_list",
		"shipment_list",
		"customer_list",
		"worker_list",
		"worker_list_credentials",
	)
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, wide)

	found := match(
		t,
		service,
		"list invoices shipments customers workers and expiring credentials by status",
		pulid.Nil,
	)

	require.NotNil(t, found.Agent)
	assert.Len(t, found.Tools, agentrecommendationports.MaxAgentRecommendationTools)
}

func TestMatchAgent_KeepsTheCurrentAgentWhenOneOfItsDelegatesHoldsTheTool(t *testing.T) {
	t.Parallel()

	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "customer_list")
	dispatch.DelegateIDs = []pulid.ID{compliance.ID}
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance)

	found := match(t, service, "when does the driver's medical card expire?", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchCurrentAgent, found.Outcome,
		"an agent that can hand the task to one that holds the tool manages the question")
	require.NotNil(t, found.Agent)
	assert.Equal(t, dispatch.ID, found.Agent.ID)
	require.NotNil(t, found.Delegate, "the match names the agent the work would be handed to")
	assert.Equal(t, compliance.ID, found.Delegate.ID)
	assert.Equal(t, []string{"worker_list_credentials"}, toolNames(found.Tools))
}

func TestMatchAgent_DoesNotCountADelegateThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	compliance.AccessMode = agentdefinition.AccessRoles
	fleet := runtimetest.AgentHolding("Fleet helper", "worker_list_credentials", "worker_list")
	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "customer_list")
	dispatch.DelegateIDs = []pulid.ID{compliance.ID}
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance, fleet)

	found := match(t, service, "when does the driver's medical card expire?", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome)
	require.NotNil(t, found.Agent)
	assert.Equal(t, fleet.ID, found.Agent.ID)
}

func TestMatchAgent_RanksToolsForWhatTheQuestionIsAbout(t *testing.T) {
	t.Parallel()

	shipments := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	customers := runtimetest.AgentHolding("Yard accounts", "customer_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, shipments, customers)
	question := "list them by status"

	plain := match(t, service, question, pulid.Nil)
	require.Equal(t, agentsuggestion.MatchNone, plain.Outcome,
		"without context the words name no tool, only a word in their descriptions")

	about := match(t, service, question, pulid.Nil, permission.ResourceCustomer.String())

	require.NotNil(t, about.Agent)
	assert.Equal(t, customers.ID, about.Agent.ID,
		"a question asked about a customer ranks the customer tools first")
}

func TestMatchAgent_IgnoresAKindOfRecordItDoesNotKnow(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)

	found := match(
		t,
		service,
		"which invoices are past due?",
		billing.ID,
		"spaceship",
		"assistant_thread",
	)

	assert.Equal(t, agentsuggestion.MatchCurrentAgent, found.Outcome,
		"a page the server cannot place is a hint to drop, not a reason to refuse the match")
}

func TestMatchAgent_RefusesMoreKindsOfRecordThanAQuestionCanName(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)
	kinds := make([]string, agentrecommendationports.MaxAgentMatchContextKinds+1)
	for idx := range kinds {
		kinds[idx] = permission.ResourceCustomer.String()
	}

	_, err := service.build().MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
		Actor:        runtimetest.Actor(),
		Text:         "which invoices are past due?",
		ContextKinds: kinds,
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "contextKinds", validation.Field)
}

func TestMatchAgent_DoesNotEmbedAQuestionThatNamesNothingAToolDoes(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)
	vectorizer := &runtimetest.FixedVectorizer{Vector: runtimetest.UsableQuery()}
	service.vectorizer = vectorizer

	found := match(t, service, "good morning, hope the weekend went well", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchNone, found.Outcome)
	assert.Empty(t, vectorizer.Calls, "small talk costs no embedding")

	_ = match(t, service, "which invoices are past due?", pulid.Nil)
	assert.Len(t, vectorizer.Calls, 1)
}

func greetingRuntime(t *testing.T) (*matchWorld, *agentdefinition.Definition) {
	t.Helper()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	reports := runtimetest.AgentHolding("Report analyst", "report_create_schedule")
	service := runtimeWithTools(t, &agenttest.StubPermissions{}, []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "shipment_list",
			Desc:     "List shipments by status, billing state or dates.",
			Resource: permission.ResourceShipment,
		},
		&agenttest.StubQueryTool{
			ToolName: "report_create_schedule",
			Desc:     "Email a saved report on a schedule, as in \"every Monday morning\".",
			Resource: permission.ResourceReport,
		},
	}, dispatch, reports)

	return service, dispatch
}

func TestMatchAgent_MatchesNothingWhenOnlyAnIncidentalWordReachesATool(t *testing.T) {
	t.Parallel()

	service, dispatch := greetingRuntime(t)

	found := match(t, service, "good morning team, hope the weekend went well", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchNone, found.Outcome,
		"a word from a tool's example names nothing the tool does")
}

func TestMatchAgent_MatchesNothingWhenNoToolComesCloseInMeaning(t *testing.T) {
	t.Parallel()

	service, dispatch := greetingRuntime(t)
	vectorizer := &runtimetest.FixedVectorizer{Vector: runtimetest.UsableQuery()}
	service.vectorizer = vectorizer
	service.vectors = &runtimetest.FixedSimilarities{ByKey: map[string]float64{
		"shipment_list":          0.12,
		"report_create_schedule": 0.23,
	}}

	found := match(t, service, "good morning team, hope the weekend went well", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchNone, found.Outcome)
	assert.Len(t, vectorizer.Calls, 1)
}

func TestMatchAgent_MatchesByMeaningWhenTheWordsAreOnlyInADescription(t *testing.T) {
	t.Parallel()

	service, dispatch := greetingRuntime(t)
	service.vectorizer = &runtimetest.FixedVectorizer{Vector: runtimetest.UsableQuery()}
	service.vectors = &runtimetest.FixedSimilarities{ByKey: map[string]float64{
		"shipment_list":          0.18,
		"report_create_schedule": 0.62,
	}}

	found := match(t, service, "have the numbers emailed every monday morning", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome)
	require.NotNil(t, found.Agent)
	assert.Equal(t, "Report analyst", found.Agent.Name)
}

type limitedRates struct {
	allowed bool
	keys    []string
}

func (l *limitedRates) Check(
	_ context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	decisions := make([]kernelports.RateLimitDecision, 0, len(requests))
	for _, request := range requests {
		l.keys = append(l.keys, request.Key)
		decisions = append(decisions, kernelports.RateLimitDecision{Allowed: l.allowed})
	}

	return decisions, nil
}

func TestMatchAgent_MatchesByKeywordOncePastThePersonsEmbeddingRate(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, billing)
	vectorizer := &runtimetest.FixedVectorizer{Vector: runtimetest.UsableQuery()}
	rates := &limitedRates{allowed: false}
	service.vectorizer = vectorizer
	service.rateLimits = rates

	found := match(t, service, "which invoices are past due?", dispatch.ID)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome)
	require.NotNil(t, found.Agent)
	assert.Equal(t, billing.ID, found.Agent.ID)
	assert.Empty(t, vectorizer.Calls, "a person over the rate is matched without an embedding")
	require.Len(t, rates.keys, 1)
	assert.Contains(t, rates.keys[0], runtimetest.Actor().OrganizationID.Prefix())
}

func TestMatchAgent_SpendsNoEmbeddingRateWithoutAnEmbeddingProvider(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, billing)
	rates := &limitedRates{allowed: true}
	service.rateLimits = rates

	found := match(t, service, "which invoices are past due?", pulid.Nil)

	assert.Equal(t, agentsuggestion.MatchBetterAgent, found.Outcome)
	assert.Empty(t, rates.keys, "nothing is embedded, so nothing is counted against the rate")
}

func TestMatchAgent_ReadsAPersonsAgentsOnceWhileTheyType(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	definitions := &countingDefinitions{
		ListedDefinitions: runtimetest.ListedDefinitions{
			ByID: map[pulid.ID]*agentdefinition.Definition{billing.ID: billing},
		},
	}
	service := recommendationRuntime(t, &agenttest.StubPermissions{})
	service.definitions = definitions
	matcher := service.build()
	actor := runtimetest.Actor()

	for _, text := range []string{"which invoices", "which invoices are past due?"} {
		found, err := matcher.MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
			Actor:          actor,
			Text:           text,
			CurrentAgentID: billing.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, agentsuggestion.MatchCurrentAgent, found.Outcome)
	}

	assert.Equal(t, 1, definitions.lists, "the agents are read once and kept for a moment")

	other := runtimetest.Actor()
	_, err := matcher.MatchAgent(t.Context(), &agentrecommendationports.AgentMatchRequest{
		Actor: other,
		Text:  "which invoices are past due?",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, definitions.lists, "another person's agents are read for them")
}

type countingDefinitions struct {
	runtimetest.ListedDefinitions

	lists int
}

func (c *countingDefinitions) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	c.lists++

	return c.ListedDefinitions.List(ctx, req)
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

	current := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	wide := runtimetest.AgentHolding(
		"Operations helper",
		"customer_list",
		"shipment_list",
		"worker_list",
	)
	ranked := ranking("invoice_list", 4, "customer_list", 3, "shipment_list", 2, "worker_list", 1)

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

	current := runtimetest.AgentHolding("Dispatch helper", "equipment_list_tractors")
	wide := runtimetest.AgentHolding(
		"Operations helper",
		"customer_list",
		"shipment_list",
		"worker_list",
	)
	billing := runtimetest.AgentHolding("Billing assistant", "invoice_list")
	ranked := ranking("invoice_list", 4, "customer_list", 3, "shipment_list", 2, "worker_list", 1)

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

func TestMatchAgent_SuggestsNobodyElseOnceThePersonSwitchedToTheSuggestion(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "customer_list")
	compliance := runtimetest.AgentHolding("Compliance assistant", "worker_list_credentials")
	fleet := runtimetest.AgentHolding("Fleet helper", "worker_list_credentials", "worker_list")
	service := recommendationRuntime(t, &agenttest.StubPermissions{}, dispatch, compliance, fleet)
	question := "when does the driver's medical card expire?"

	first := match(t, service, question, dispatch.ID)
	require.Equal(t, agentsuggestion.MatchBetterAgent, first.Outcome)
	require.NotNil(t, first.Agent)

	again := match(t, service, question, first.Agent.ID)
	require.NotNil(t, again.Agent)
	assert.Equal(t, first.Agent.ID, again.Agent.ID)
	assert.Equal(t, agentsuggestion.MatchCurrentAgent, again.Outcome,
		"the agent switched to is kept, not traded for another holder")
}

func TestBestCandidate_KeepsTheCurrentAgentWhenItHoldsAToolTiedForTheBestMatch(t *testing.T) {
	t.Parallel()

	current := runtimetest.AgentHolding("Dispatch reader", "shipment_get", "shipment_list")
	writer := runtimetest.AgentHolding("Shipment builder", "shipment_create")
	ranked := ranking("shipment_create", 4, "shipment_get", 4, "shipment_list", 4)

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

	current := runtimetest.AgentHolding("Fleet helper", "equipment_list_tractors")
	builder := runtimetest.AgentHolding("Shipment builder", "shipment_create")
	reader := runtimetest.AgentHolding("Dispatch reader", "shipment_get", "shipment_list")
	ranked := ranking(
		"shipment_create",
		4,
		"shipment_get",
		4,
		"shipment_list",
		4,
		"equipment_list_tractors",
		1,
	)

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
	runtimetest.ListedDefinitions

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
		Items:      p.all[offset:end],
		Total:      len(p.all),
		TotalExact: true,
	}, nil
}

func TestMatchAgent_ReadsPastTheFirstHundredAgents(t *testing.T) {
	t.Parallel()

	all := make([]*agentdefinition.Definition, 0, 150)
	for idx := range 149 {
		all = append(all, runtimetest.AgentHolding(fmt.Sprintf("Agent %03d", idx), "customer_list"))
	}
	compliance := runtimetest.AgentHolding("Zeta compliance", "worker_list_credentials")
	all = append(all, compliance)
	definitions := &pagedDefinitions{all: all}
	service := recommendationRuntime(t, &agenttest.StubPermissions{})
	service.definitions = definitions

	found := match(t, service, "driver medical card expiry", pulid.Nil)

	require.NotNil(t, found.Agent)
	assert.Equal(t, compliance.ID, found.Agent.ID)
	assert.Equal(t, 2, definitions.pages)
}
