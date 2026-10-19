package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An agent an organization built by hand holds memory, escalation and review
without anyone ticking them.

The Report Builder and Homepage Widget Builder agents were saved with a
hand-picked list that named none of the four, so neither could recall a
correction nor raise its hand when stuck. No template or seed reaches an agent
with no template, so the only place this can be true is the definition itself.
*/
func TestDefinition_HoldsTheCoreToolsWithoutSelectingThem(t *testing.T) {
	t.Parallel()

	definition := validDefinition()
	definition.ToolNames = []string{"report_list"}

	for _, tool := range agentdefinition.CoreTools() {
		assert.Truef(t, definition.AllowsTool(tool), "an agent must hold %s", tool)
	}
	assert.True(t, definition.AllowsTool("report_list"))
	assert.False(t, definition.AllowsTool("report_create_dashboard"))

	assert.Equal(t,
		append(agentdefinition.CoreTools(), "report_list"),
		definition.EffectiveToolNames(),
	)
}

func TestDefinition_EffectiveToolNamesDoesNotRepeatACoreToolAlsoSelected(t *testing.T) {
	t.Parallel()

	definition := validDefinition()
	definition.ToolNames = []string{"agent_create_memory", "report_list"}

	names := definition.EffectiveToolNames()
	seen := make(map[string]int, len(names))
	for _, name := range names {
		seen[name]++
	}
	assert.Equal(t, 1, seen["agent_create_memory"])
	assert.Len(t, names, len(agentdefinition.CoreTools())+1)
}

// A tier or a daily cap on a core tool is a setting on a tool the agent holds,
// so it must validate even though the tool is not in the selection.
func TestDefinition_AcceptsTiersAndLimitsOnACoreTool(t *testing.T) {
	t.Parallel()

	definition := validDefinition()
	definition.ToolNames = nil
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{"agent_create_memory": agent.TierAutoExecute}
	definition.ToolDailyLimits = map[string]int{"agent_raise_exception": 5}

	multiErr := errortypes.NewMultiError()
	definition.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestWithoutCoreTools_KeepsOnlyTheSelection(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]string{"report_list", "report_create"},
		agentdefinition.WithoutCoreTools(
			[]string{
				"agent_list_memories",
				"report_list",
				"agent_create_memory",
				"report_create",
				"agent_raise_exception",
			},
		),
	)
}

// No template lists a core tool: it would be stored per agent and shown twice
// in the builder, once locked and once as a choice.
func TestTemplates_ListNoCoreTool(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		for _, tool := range template.StarterTools() {
			assert.Falsef(t, agentdefinition.IsCoreTool(tool),
				"%s lists the core tool %s", template.Label(), tool)
		}
	}
}

func TestHoldingReads_AddsEveryReadButOnesTheAgentSwitchedOff(t *testing.T) {
	t.Parallel()

	definition := &agentdefinition.Definition{
		ToolNames:         []string{"shipment_get", "dispatch_assign_move"},
		DisabledToolNames: []string{"list_settlements"},
	}

	reading := definition.HoldingReads([]string{
		"shipment_get",
		"customer_get",
		"list_settlements",
		agentdefinition.CoreToolFindInTrenova,
		"customer_get",
	})

	assert.Equal(t, []string{"shipment_get", "dispatch_assign_move", "customer_get"}, reading.ToolNames)
	assert.Equal(
		t,
		[]string{"shipment_get", "dispatch_assign_move"},
		definition.ToolNames,
		"the stored agent keeps the tools it was configured with",
	)
}
