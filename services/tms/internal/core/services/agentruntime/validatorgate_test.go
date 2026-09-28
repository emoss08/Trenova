package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A tool that can check its own arguments is asked at every tier before
// anything is filed, simulated or run. It used to be asked only for a held
// write: an automatic write, and an agent in simulation, went straight to
// the service and learned there what a person would have been told first.
func TestRun_AToolThatSaysItWouldFailIsRefusedAtEveryTier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		tier       agent.AutonomyTier
		definition func(tool string) *agentdefinition.Definition
		simulation bool
	}{
		{
			name: "auto execute",
			tier: agent.TierAutoExecute,
			definition: func(tool string) *agentdefinition.Definition {
				return autoDefinition(tool)
			},
		},
		{
			name: "act with approval",
			tier: agent.TierActWithApproval,
			definition: func(tool string) *agentdefinition.Definition {
				definition := testDefinition(tool)
				definition.AutonomyCeiling = agent.TierActWithApproval

				return definition
			},
		},
		{
			name: "propose",
			tier: agent.TierPropose,
			definition: func(tool string) *agentdefinition.Definition {
				return testDefinition(tool)
			},
		},
		{
			name: "simulation",
			tier: agent.TierAutoExecute,
			definition: func(tool string) *agentdefinition.Definition {
				definition := autoDefinition(tool)
				definition.SimulationMode = true

				return definition
			},
			simulation: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tool := &validatingActionTool{
				stubActionToolAlias: actionTool("update_report", tc.tier, nil),
				invalid:             errors.New("the report no longer has a column named lateness"),
			}
			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				toolTurn("update_report", map[string]any{"definitionId": "rd_1"}),
				textTurn("I will fix the column."),
			}}
			rt := newRuntime(completion, &stubQueryRegistry{},
				&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
			actor := testActor()

			result, err := rt.Run(t.Context(), &serviceports.RunRequest{
				Definition: tc.definition("update_report"),
				Actor:      actor,
				Input:      "update",
			})
			require.NoError(t, err)

			assert.Equal(t, 1, tool.checked, "the tool was asked once")
			assert.Equal(t, actor.OrganizationID, tool.lastSeen.OrganizationID)
			assert.Zero(t, tool.Calls, "nothing ran")
			assert.Empty(t, result.Actions, "nothing was filed, simulated or run")
			assert.Equal(t, 1, result.ToolCallsUsed)

			refusals := toolMessages(result)
			require.Len(t, refusals, 1)
			assert.True(t, refusals[0].ToolFailed)
			assert.Contains(t, refusals[0].Content, "would fail as called")
			assert.Contains(t, refusals[0].Content, "lateness")
			if tc.tier == agent.TierAutoExecute {
				assert.Contains(t, refusals[0].Content, "was not run")
			} else {
				assert.Contains(t, refusals[0].Content, "was not proposed")
			}
			if tc.simulation {
				assert.NotContains(t, refusals[0].Content, "Simulated",
					"a call that would fail is refused, not simulated")
			}
		})
	}
}

func TestRun_AToolThatValidatesRunsWhenItsCheckPasses(t *testing.T) {
	t.Parallel()

	tool := &validatingActionTool{
		stubActionToolAlias: actionTool("update_report", agent.TierAutoExecute, nil),
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", map[string]any{"definitionId": "rd_1"}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: autoDefinition("update_report"),
		Actor:      testActor(),
		Input:      "update",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.checked)
	assert.Equal(t, 1, tool.Calls)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}
