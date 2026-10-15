package agentruntime

import (
	"context"
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
