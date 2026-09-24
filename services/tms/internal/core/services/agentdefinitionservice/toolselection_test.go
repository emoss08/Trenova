package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testRegistries() (*agentruntimetest.StubActionRegistry, *agentruntimetest.StubQueryRegistry) {
	actions := &agentruntimetest.StubActionRegistry{Tools: []serviceports.AgentTool{
		&agentruntimetest.StubActionTool{
			ToolName: "reassign_move",
			Resource: permission.ResourceShipmentMove,
			Tier:     agent.TierPropose,
		},
		&agentruntimetest.StubActionTool{
			ToolName: "correct_charge_code",
			Resource: permission.ResourceBillingQueue,
			Tier:     agent.TierPropose,
		},
		&agentruntimetest.StubActionTool{
			ToolName: "update_customer",
			Resource: permission.ResourceCustomer,
			Tier:     agent.TierActWithApproval,
		},
	}}
	queries := &agentruntimetest.StubQueryRegistry{Tools: []serviceports.AgentQueryTool{
		&agentruntimetest.StubQueryTool{
			ToolName: "get_shipment",
			Resource: permission.ResourceShipment,
		},
	}}

	return actions, queries
}

func definition(template agentdefinition.Template, tools ...string) *agentdefinition.Definition {
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
	validateToolSelection(toolSelection{definition: d, actions: actions, queries: queries}, multiErr)

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
		validate(t, definition(agentdefinition.TemplateCustomerAssistant, "correct_charge_code")),
	)
	assert.Empty(
		t,
		validate(
			t,
			definition(agentdefinition.TemplateGeneralAssistant, "reassign_move", "get_shipment"),
		),
	)
	assert.Empty(t, validate(t, definition("", "update_customer")))
}

func TestValidateToolSelection_RefusesUnknownTools(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(agentdefinition.TemplateDispatchAssistant, "exec_shell"))
	assert.True(t, errs["toolNames[0]"], "a tool that does not exist cannot be configured")
}

func TestValidateToolSelection_ReportsEachOffendingToolByIndex(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(
		agentdefinition.TemplateDispatchAssistant,
		"reassign_move",
		"get_shipment",
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
	byName := make(map[string]serviceports.ToolCatalogEntry, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}

	assert.Equal(t, serviceports.ToolCatalogKindQuery, byName["get_shipment"].Kind)
	assert.Equal(t, permission.OpRead, byName["get_shipment"].Operation)
	assert.Equal(t, serviceports.ToolCatalogKindAction, byName["update_customer"].Kind)
	assert.Equal(t, agent.TierActWithApproval, byName["update_customer"].DefaultAutonomyTier)
	assert.Equal(t, permission.ResourceCustomer, byName["update_customer"].Resource)
}

func TestBuildToolCatalog_SaysWhatEachToolDoes(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	catalog := buildToolCatalog(actions, queries, nil)

	byName := make(map[string]serviceports.ToolCatalogEntry, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}

	assert.Equal(t, agent.ToolEffectLookup, byName["get_shipment"].Effect)
	assert.Equal(t, agent.ToolEffectChange, byName["update_customer"].Effect)
	assert.Equal(t, agent.ToolEffectChange, byName["reassign_move"].Effect)
}

func TestRegisteredStarterTools_DropsToolsNotYetRegistered(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	tools := registeredStarterTools(agentdefinition.TemplateDispatchAssistant, actions, queries)

	assert.Equal(t, []string{"get_shipment"}, tools,
		"starter tools that no registry provides are not offered")
}

type stubDefinitionRepo struct {
	repositories.AgentDefinitionRepository

	existing *agentdefinition.Definition
	deleted  []pulid.ID
}

func (r *stubDefinitionRepo) GetByID(
	_ context.Context,
	_ repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return r.existing, nil
}

func (r *stubDefinitionRepo) Delete(
	_ context.Context,
	req repositories.DeleteAgentDefinitionRequest,
) error {
	r.deleted = append(r.deleted, req.ID)

	return nil
}

type stubAudit struct{ serviceports.AuditService }

func (stubAudit) LogAction(_ *serviceports.LogActionParams, _ ...serviceports.LogOption) error {
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
		repositories.DeleteAgentDefinitionRequest{ID: repo.existing.ID},
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, repo.deleted)
}

func TestApply_ClearsTriggerFieldsThatDoNotBelongToTheMode(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	apply(d, &serviceports.SaveAgentDefinitionRequest{
		Name:            " Night desk ",
		TriggerMode:     agentdefinition.TriggerEvent,
		CronExpression:  "* * * * *",
		IntervalSeconds: 300,
		EventKinds:      []agent.EventKind{agent.EventShipmentCreated},
		ToolNames:       []string{" get_shipment "},
		ToolTiers:       map[string]agent.AutonomyTier{"get_shipment": agent.TierAutoExecute},
	})

	assert.Equal(t, "Night desk", d.Name)
	assert.Empty(t, d.CronExpression)
	assert.Zero(t, d.IntervalSeconds)
	assert.Equal(t, []agent.EventKind{agent.EventShipmentCreated}, d.EventKinds)
	assert.Equal(t, []string{"get_shipment"}, d.ToolNames)
	assert.Equal(t, agentdefinition.DefaultMaxToolCalls, d.MaxToolCalls)
	assert.Equal(t, agentdefinition.OutputConversational, d.OutputMode)
}

// A saved selection never stores a core tool. A client that still sends one —
// an older builder, or a definition loaded before the core set existed — is not
// an error, but it must not come back as a choice the organization made.
func TestApply_StoresNoCoreTool(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	apply(d, &serviceports.SaveAgentDefinitionRequest{
		Name:      "Report builder",
		ToolNames: []string{"recall_memory", " list_reports ", "remember", "create_report"},
	})

	assert.Equal(t, []string{"list_reports", "create_report"}, d.ToolNames)
}

func TestBuildToolCatalog_MarksTheCoreTools(t *testing.T) {
	t.Parallel()

	actions, queries := testRegistries()
	queries.Tools = append(
		queries.Tools,
		&agentruntimetest.StubQueryTool{
			ToolName: "recall_memory",
			Resource: permission.ResourceAgentMemory,
		},
	)

	byName := make(map[string]serviceports.ToolCatalogEntry)
	for _, entry := range buildToolCatalog(actions, queries, nil) {
		byName[entry.Name] = entry
	}

	assert.True(t, byName["recall_memory"].Core)
	assert.False(t, byName["get_shipment"].Core)
}

type cappedStubTool struct {
	*agentruntimetest.StubActionTool
}

func (t cappedStubTool) Policy() serviceports.ToolPolicy {
	policy := t.StubActionTool.Policy()
	policy.MaxTier = agent.TierActWithApproval
	policy.Egress = []agent.EgressClass{agent.EgressExternalRecipient}

	return policy
}

// A person approves anything that leaves the organization, so an agent cannot
// be set to send it on its own.
func TestValidateToolSelection_RefusesATierAboveAToolsCeiling(t *testing.T) {
	t.Parallel()

	actions := &agentruntimetest.StubActionRegistry{Tools: []serviceports.AgentTool{
		cappedStubTool{&agentruntimetest.StubActionTool{
			ToolName: "email_customer",
			Resource: permission.ResourceCustomer,
			Tier:     agent.TierActWithApproval,
		}},
	}}
	queries := &agentruntimetest.StubQueryRegistry{}

	for tier, refused := range map[agent.AutonomyTier]bool{
		agent.TierPropose:         false,
		agent.TierActWithApproval: false,
		agent.TierAutoExecute:     true,
	} {
		d := definition("", "email_customer")
		d.ToolTiers = map[string]agent.AutonomyTier{"email_customer": tier}
		multiErr := errortypes.NewMultiError()
		validateToolSelection(toolSelection{definition: d, actions: actions, queries: queries}, multiErr)

		fields := make(map[string]bool)
		for _, e := range multiErr.Errors {
			fields[e.Field] = true
		}
		assert.Equal(t, refused, fields["toolTiers.email_customer"], tier)
	}
}

type policyStubTool struct {
	*agentruntimetest.StubActionTool

	policy serviceports.ToolPolicy
}

func (t policyStubTool) Policy() serviceports.ToolPolicy { return t.policy }

func tierMessages(
	t *testing.T,
	tool policyStubTool,
	tier agent.AutonomyTier,
) map[string]string {
	t.Helper()

	actions := &agentruntimetest.StubActionRegistry{Tools: []serviceports.AgentTool{tool}}
	d := definition("", tool.ToolName)
	d.ToolTiers = map[string]agent.AutonomyTier{tool.ToolName: tier}
	multiErr := errortypes.NewMultiError()
	validateToolSelection(toolSelection{
		definition: d,
		actions:    actions,
		queries:    &agentruntimetest.StubQueryRegistry{},
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
false for create_shipment, held to approval because a new load commits a
customer's freight.
*/
func TestValidateToolSelection_ExplainsTheCapWithTheToolsRationale(t *testing.T) {
	t.Parallel()

	booking := policyStubTool{StubActionTool: &agentruntimetest.StubActionTool{
		ToolName: "create_shipment",
	}}
	booking.policy = booking.StubActionTool.Policy()
	booking.policy.MaxTier = agent.TierActWithApproval
	booking.policy.Rationale = "A new load commits a customer's freight."

	mail := policyStubTool{StubActionTool: &agentruntimetest.StubActionTool{
		ToolName: "email_customer",
	}}
	mail.policy = mail.StubActionTool.Policy()
	mail.policy.Egress = []agent.EgressClass{agent.EgressExternalRecipient}
	mail.policy.Rationale = "Emails the customer's contacts a message the model wrote."

	booked := tierMessages(t, booking, agent.TierAutoExecute)["toolTiers.create_shipment"]
	assert.Contains(t, booked, "A new load commits a customer's freight.")
	assert.Contains(t, booked, string(agent.TierActWithApproval))
	assert.NotContains(t, booked, "outside the organization")

	mailed := tierMessages(t, mail, agent.TierAutoExecute)["toolTiers.email_customer"]
	assert.Contains(t, mailed, "Emails the customer's contacts a message the model wrote.")
	assert.Contains(t, mailed, string(agent.TierActWithApproval))

	assert.Empty(t, tierMessages(t, booking, agent.TierActWithApproval))
	assert.Empty(t, tierMessages(t, mail, agent.TierActWithApproval))
}
