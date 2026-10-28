package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testRegistries() (*agenttest.StubActionRegistry, *agenttest.StubQueryRegistry) {
	actions := &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{
		&agenttest.StubActionTool{
			ToolName: "reassign_move",
			Resource: permission.ResourceShipmentMove,
			Tier:     agent.TierPropose,
		},
		&agenttest.StubActionTool{
			ToolName: "billing_correct_charge_codes",
			Resource: permission.ResourceBillingQueue,
			Tier:     agent.TierPropose,
		},
		&agenttest.StubActionTool{
			ToolName: "customer_update",
			Resource: permission.ResourceCustomer,
			Tier:     agent.TierActWithApproval,
		},
	}}
	queries := &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "shipment_get",
			Resource: permission.ResourceShipment,
		},
	}}

	return actions, queries
}

func definition(template agenttemplate.Template, tools ...string) *agentdefinition.Definition {
	d := &agentdefinition.Definition{
		Name:            "Test",
		Template:        template,
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       tools,
	}
	d.ApplyDefaults()

	return d
}

func validate(t *testing.T, d *agentdefinition.Definition) map[string]bool {
	t.Helper()

	actions, queries := testRegistries()
	multiErr := errortypes.NewMultiError()
	validateToolSelection(
		toolSelection{definition: d, actions: actions, queries: queries},
		multiErr,
	)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

// A template is a starting point. An organization may hand any agent any tool
// the system provides, whatever template it began from.
func TestValidateToolSelection_AcceptsAnyRegisteredToolOnAnyTemplate(t *testing.T) {
	t.Parallel()

	assert.Empty(
		t,
		validate(
			t,
			definition(agenttemplate.TemplateCustomerAssistant, "billing_correct_charge_codes"),
		),
	)
	assert.Empty(
		t,
		validate(
			t,
			definition(agenttemplate.TemplateGeneralAssistant, "reassign_move", "shipment_get"),
		),
	)
	assert.Empty(t, validate(t, definition("", "customer_update")))
}

func TestValidateToolSelection_RefusesUnknownTools(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(agenttemplate.TemplateDispatchAssistant, "exec_shell"))
	assert.True(t, errs["toolNames[0]"], "a tool that does not exist cannot be configured")
}

func TestValidateToolSelection_ReportsEachOffendingToolByIndex(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(
		agenttemplate.TemplateDispatchAssistant,
		"reassign_move",
		"shipment_get",
		"exec_shell",
		"run_sql",
	))

	assert.False(t, errs["toolNames[0]"])
	assert.False(t, errs["toolNames[1]"])
	assert.True(t, errs["toolNames[2]"])
	assert.True(t, errs["toolNames[3]"])
}

func TestBuildToolCatalog_ListsEveryRegisteredToolWithItsKind(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	catalog := buildToolCatalog(actions, queries, nil)

	require.Len(t, catalog, 4)
	byName := make(map[string]agentdefinitionports.ToolCatalogEntry, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}

	assert.Equal(t, agentdefinitionports.ToolCatalogKindQuery, byName["shipment_get"].Kind)
	assert.Equal(t, permission.OpRead, byName["shipment_get"].Operation)
	assert.Equal(t, agentdefinitionports.ToolCatalogKindAction, byName["customer_update"].Kind)
	assert.Equal(t, agent.TierActWithApproval, byName["customer_update"].DefaultAutonomyTier)
	assert.Equal(t, permission.ResourceCustomer, byName["customer_update"].Resource)
}

func TestBuildToolCatalog_SaysWhatEachToolDoes(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	catalog := buildToolCatalog(actions, queries, nil)

	byName := make(map[string]agentdefinitionports.ToolCatalogEntry, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}

	assert.Equal(t, agent.ToolEffectLookup, byName["shipment_get"].Effect)
	assert.Equal(t, agent.ToolEffectChange, byName["customer_update"].Effect)
	assert.Equal(t, agent.ToolEffectChange, byName["reassign_move"].Effect)
}

func TestRegisteredStarterTools_DropsToolsNotYetRegistered(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	tools := registeredStarterTools(agenttemplate.TemplateBillingException, actions, queries)

	assert.Equal(t, []string{"shipment_get", "billing_correct_charge_codes"}, tools,
		"starter tools that no registry provides are not offered")
}

type stubDefinitionRepo struct {
	agentdefinitionports.AgentDefinitionRepository

	existing *agentdefinition.Definition
	deleted  []pulid.ID
}

func (r *stubDefinitionRepo) GetByID(
	_ context.Context,
	_ agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return r.existing, nil
}

func (r *stubDefinitionRepo) Delete(
	_ context.Context,
	req agentdefinitionports.DeleteAgentDefinitionRequest,
) error {
	r.deleted = append(r.deleted, req.ID)

	return nil
}

type stubAudit struct{ auditports.AuditService }

func (stubAudit) LogAction(_ *auditports.LogActionParams, _ ...auditports.LogOption) error {
	return nil
}

// A system agent is what the platform's own events fire; deleting it would
// leave an event with nothing to run. Disabling it is the supported way to
// switch it off.
func TestDelete_RefusesASystemAgent(t *testing.T) {
	t.Parallel()

	repo := &stubDefinitionRepo{existing: &agentdefinition.Definition{
		ID:        pulid.MustNew("agdef_"),
		Name:      "Billing exceptions",
		SystemKey: "billing_exception",
	}}
	svc := &Service{l: zap.NewNop(), repo: repo, audit: stubAudit{}}

	err := svc.Delete(
		t.Context(),
		agentdefinitionports.DeleteAgentDefinitionRequest{ID: repo.existing.ID},
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, repo.deleted)
}

func TestApply_ClearsTriggerFieldsThatDoNotBelongToTheMode(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	apply(d, &agentdefinitionports.SaveAgentDefinitionRequest{
		Name:            " Night desk ",
		TriggerMode:     agentdefinition.TriggerEvent,
		CronExpression:  "* * * * *",
		IntervalSeconds: 300,
		EventKinds:      []agent.EventKind{agent.EventShipmentCreated},
		ToolNames:       []string{" shipment_get "},
		ToolTiers:       map[string]agent.AutonomyTier{"shipment_get": agent.TierAutoExecute},
	})

	assert.Equal(t, "Night desk", d.Name)
	assert.Empty(t, d.CronExpression)
	assert.Zero(t, d.IntervalSeconds)
	assert.Equal(t, []agent.EventKind{agent.EventShipmentCreated}, d.EventKinds)
	assert.Equal(t, []string{"shipment_get"}, d.ToolNames)
	assert.Equal(t, agentdefinition.DefaultMaxToolCalls, d.MaxToolCalls)
	assert.Equal(t, agentdefinition.OutputConversational, d.OutputMode)
}

// A saved selection never stores a core tool. A client that still sends one —
// an older builder, or a definition loaded before the core set existed — is not
// an error, but it must not come back as a choice the organization made.
func TestApply_StoresNoCoreTool(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	apply(d, &agentdefinitionports.SaveAgentDefinitionRequest{
		Name: "Report builder",
		ToolNames: []string{
			"agent_list_memories",
			" report_list ",
			"agent_create_memory",
			"report_create",
		},
	})

	assert.Equal(t, []string{"report_list", "report_create"}, d.ToolNames)
}

func TestBuildToolCatalog_MarksTheCoreTools(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	queries.Tools = append(
		queries.Tools,
		&agenttest.StubQueryTool{
			ToolName: "agent_list_memories",
			Resource: permission.ResourceAgentMemory,
		},
	)

	byName := make(map[string]agentdefinitionports.ToolCatalogEntry)
	for _, entry := range buildToolCatalog(actions, queries, nil) {
		byName[entry.Name] = entry
	}

	assert.True(t, byName["agent_list_memories"].Core)
	assert.False(t, byName["shipment_get"].Core)
}

type cappedStubTool struct {
	*agenttest.StubActionTool
}

func (t cappedStubTool) Policy() agenttoolports.ToolPolicy {
	policy := t.StubActionTool.Policy()
	policy.MaxTier = agent.TierActWithApproval
	policy.Egress = []agent.EgressClass{agent.EgressExternalRecipient}

	return policy
}

// A person approves anything that leaves the organization, so an agent cannot
// be set to send it on its own.
func TestValidateToolSelection_RefusesATierAboveAToolsCeiling(t *testing.T) {
	t.Parallel()

	actions := &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{
		cappedStubTool{&agenttest.StubActionTool{
			ToolName: "customer_send_email",
			Resource: permission.ResourceCustomer,
			Tier:     agent.TierActWithApproval,
		}},
	}}
	queries := &agenttest.StubQueryRegistry{}

	for tier, refused := range map[agent.AutonomyTier]bool{
		agent.TierPropose:         false,
		agent.TierActWithApproval: false,
		agent.TierAutoExecute:     true,
	} {
		d := definition("", "customer_send_email")
		d.ToolTiers = map[string]agent.AutonomyTier{"customer_send_email": tier}
		multiErr := errortypes.NewMultiError()
		validateToolSelection(
			toolSelection{definition: d, actions: actions, queries: queries},
			multiErr,
		)

		fields := make(map[string]bool)
		for _, e := range multiErr.Errors {
			fields[e.Field] = true
		}
		assert.Equal(t, refused, fields["toolTiers.customer_send_email"], tier)
	}
}

type policyStubTool struct {
	*agenttest.StubActionTool

	policy agenttoolports.ToolPolicy
}

func (t policyStubTool) Policy() agenttoolports.ToolPolicy { return t.policy }

func tierMessages(
	t *testing.T,
	tool policyStubTool,
	tier agent.AutonomyTier,
) map[string]string {
	t.Helper()

	actions := &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}
	d := definition("", tool.ToolName)
	d.ToolTiers = map[string]agent.AutonomyTier{tool.ToolName: tier}
	multiErr := errortypes.NewMultiError()
	validateToolSelection(toolSelection{
		definition: d,
		actions:    actions,
		queries:    &agenttest.StubQueryRegistry{},
	}, multiErr)

	messages := make(map[string]string, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		messages[e.Field] = e.Message
	}

	return messages
}

/*
The refusal says why the tool stops where it does, in the tool's own words. It
used to say every capped tool "sends work outside the organization", which was
false for shipment_create, held to approval because a new load commits a
customer's freight.
*/
func TestValidateToolSelection_ExplainsTheCapWithTheToolsRationale(t *testing.T) {
	t.Parallel()

	booking := policyStubTool{StubActionTool: &agenttest.StubActionTool{
		ToolName: "shipment_create",
	}}
	booking.policy = booking.StubActionTool.Policy()
	booking.policy.MaxTier = agent.TierActWithApproval
	booking.policy.Rationale = "A new load commits a customer's freight."

	mail := policyStubTool{StubActionTool: &agenttest.StubActionTool{
		ToolName: "customer_send_email",
	}}
	mail.policy = mail.StubActionTool.Policy()
	mail.policy.Egress = []agent.EgressClass{agent.EgressExternalRecipient}
	mail.policy.Rationale = "Emails the customer's contacts a message the model wrote."

	booked := tierMessages(t, booking, agent.TierAutoExecute)["toolTiers.shipment_create"]
	assert.Contains(t, booked, "A new load commits a customer's freight.")
	assert.Contains(t, booked, string(agent.TierActWithApproval))
	assert.NotContains(t, booked, "outside the organization")

	mailed := tierMessages(t, mail, agent.TierAutoExecute)["toolTiers.customer_send_email"]
	assert.Contains(t, mailed, "Emails the customer's contacts a message the model wrote.")
	assert.Contains(t, mailed, string(agent.TierActWithApproval))

	assert.Empty(t, tierMessages(t, booking, agent.TierActWithApproval))
	assert.Empty(t, tierMessages(t, mail, agent.TierActWithApproval))
}

func TestApply_StoresTheMemoryBudgetAsSent(t *testing.T) {
	t.Parallel()

	budget := 12000
	d := &agentdefinition.Definition{}
	apply(
		d,
		&agentdefinitionports.SaveAgentDefinitionRequest{Name: "Rates", MemoryTokenBudget: &budget},
	)
	require.NotNil(t, d.MemoryTokenBudget)
	assert.Equal(t, 12000, d.EffectiveMemoryTokenBudget())

	apply(d, &agentdefinitionports.SaveAgentDefinitionRequest{Name: "Rates"})
	assert.Nil(t, d.MemoryTokenBudget, "clearing the field returns the agent to the default")
	assert.Equal(t, agentdefinition.DefaultMemoryTokenBudget, d.EffectiveMemoryTokenBudget())
}

func TestBuildToolCatalog_NamesTheMostEachChangeMayBeGiven(t *testing.T) {
	t.Parallel()

	actions := &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{
		&agenttest.StubActionTool{ToolName: "customer_update"},
		&agenttest.StubActionTool{ToolName: "pay_carrier", Egress: agent.EgressMoney},
	}}
	queries := &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "shipment_get"},
	}}

	byName := make(map[string]agentdefinitionports.ToolCatalogEntry)
	for _, entry := range buildToolCatalog(actions, queries, nil) {
		byName[entry.Name] = entry
	}

	assert.Equal(t, agent.TierAutoExecute, byName["customer_update"].MaxAutonomyTier)
	assert.Equal(t, agent.EgressMoney.Ceiling(), byName["pay_carrier"].MaxAutonomyTier)
	assert.Empty(t, byName["shipment_get"].MaxAutonomyTier)
}

// noPrerequisites is a tool that can name the tools it takes its arguments
// from and names none, as most receivable tools do.
type noPrerequisites struct {
	*agenttest.StubQueryTool
}

func (noPrerequisites) Prerequisites() []string { return nil }

// The client reads prerequisites as a list. A tool that can name some and
// names none was sent as null, and the AI Control tool picker refused the
// whole catalog.
func TestCatalogEntry_SendsNoPrerequisitesAsAnEmptyList(t *testing.T) {
	t.Parallel()

	for _, tool := range []catalogTool{
		noPrerequisites{&agenttest.StubQueryTool{ToolName: "list_open_invoices"}},
		&agenttest.StubQueryTool{ToolName: "shipment_get"},
	} {
		encoded, err := sonic.Marshal(catalogEntry(tool, agentdefinitionports.ToolCatalogKindQuery))
		require.NoError(t, err)
		assert.Contains(t, string(encoded), `"prerequisites":[]`, tool.Name())
	}
}
