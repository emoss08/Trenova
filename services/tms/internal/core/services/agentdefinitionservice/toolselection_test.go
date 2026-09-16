package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTool struct {
	name     string
	resource permission.Resource
	tier     agent.AutonomyTier
}

func (f fakeTool) Name() string                              { return f.name }
func (f fakeTool) Description() string                       { return "fake" }
func (f fakeTool) ParamSchema() map[string]any               { return map[string]any{} }
func (f fakeTool) Reversible() bool                          { return true }
func (f fakeTool) PermissionResource() permission.Resource   { return f.resource }
func (f fakeTool) PermissionOperation() permission.Operation { return permission.OpUpdate }
func (f fakeTool) RequiresIdempotencyKey() bool              { return false }
func (f fakeTool) DefaultAutonomyTier() agent.AutonomyTier   { return f.tier }
func (f fakeTool) Execute(_ context.Context, _ serviceports.ToolExecuteParams) error {
	return nil
}

type fakeRegistry struct {
	tools []serviceports.AgentTool
}

func (r fakeRegistry) Get(name string) (serviceports.AgentTool, bool) {
	for _, tool := range r.tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r fakeRegistry) All() []serviceports.AgentTool { return r.tools }

func (r fakeRegistry) Descriptors() []serviceports.AgentToolDescriptor {
	return nil
}

func testRegistry() fakeRegistry {
	return fakeRegistry{tools: []serviceports.AgentTool{
		fakeTool{
			name:     "reassign_move",
			resource: permission.ResourceShipmentMove,
			tier:     agent.TierPropose,
		},
		fakeTool{
			name:     "correct_charge_code",
			resource: permission.ResourceBillingQueue,
			tier:     agent.TierPropose,
		},
		fakeTool{
			name:     "update_customer",
			resource: permission.ResourceCustomer,
			tier:     agent.TierActWithApproval,
		},
	}}
}

func definition(kind agentdefinition.Kind, tools ...string) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		Name:            "Test",
		Kind:            kind,
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       tools,
	}
}

func validate(t *testing.T, d *agentdefinition.Definition) map[string]bool {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	validateToolSelection(d, testRegistry(), multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestValidateToolSelection_AcceptsToolsInsideTheTemplateBound(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(agentdefinition.KindDispatchAssistant, "reassign_move"))
	assert.Empty(t, errs)
}

// The narrowing rule: a template's bound is an outer limit, and no configuration
// reaches past it.
func TestValidateToolSelection_RefusesToolsOutsideTheTemplateBound(t *testing.T) {
	t.Parallel()

	t.Run("customer assistant cannot hold a billing tool", func(t *testing.T) {
		t.Parallel()
		errs := validate(t, definition(agentdefinition.KindCustomerAssistant, "correct_charge_code"))
		assert.True(t, errs["toolNames[0]"])
	})

	t.Run("dispatch assistant cannot hold a customer tool", func(t *testing.T) {
		t.Parallel()
		errs := validate(t, definition(agentdefinition.KindDispatchAssistant, "update_customer"))
		assert.True(t, errs["toolNames[0]"])
	})

	t.Run("billing assistant cannot hold a move tool", func(t *testing.T) {
		t.Parallel()
		errs := validate(t, definition(agentdefinition.KindBillingAssistant, "reassign_move"))
		assert.True(t, errs["toolNames[0]"])
	})
}

func TestValidateToolSelection_RefusesUnknownTools(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(agentdefinition.KindDispatchAssistant, "exec_shell"))
	assert.True(t, errs["toolNames[0]"], "a tool that does not exist cannot be configured")
}

func TestValidateToolSelection_ReportsEachOffendingToolByIndex(t *testing.T) {
	t.Parallel()

	errs := validate(t, definition(
		agentdefinition.KindDispatchAssistant,
		"reassign_move",
		"correct_charge_code",
		"exec_shell",
	))

	assert.False(t, errs["toolNames[0]"], "the permitted tool should not be reported")
	assert.True(t, errs["toolNames[1]"])
	assert.True(t, errs["toolNames[2]"])
}

func TestAvailableTools_OffersOnlyThePermittedSet(t *testing.T) {
	t.Parallel()

	dispatch := AvailableTools(agentdefinition.KindDispatchAssistant, testRegistry())
	require.Len(t, dispatch, 1)
	assert.Equal(t, "reassign_move", dispatch[0].Name)

	billing := AvailableTools(agentdefinition.KindBillingAssistant, testRegistry())
	require.Len(t, billing, 1)
	assert.Equal(t, "correct_charge_code", billing[0].Name)
}

// A read-only template is offered nothing, so the configuration UI cannot present
// a choice that would then be refused on save.
func TestAvailableTools_OffersNothingForAReadOnlyTemplate(t *testing.T) {
	t.Parallel()

	assert.Empty(t, AvailableTools(agentdefinition.KindGeneralAssistant, testRegistry()))
	assert.Empty(t, AvailableTools(agentdefinition.Kind("Bogus"), testRegistry()))
}
