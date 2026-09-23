package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type privateActionTool struct {
	*agentruntimetest.StubActionTool

	private bool
}

func (t *privateActionTool) PrivateToCaller(context.Context, serviceports.ToolExecuteParams) bool {
	return t.private
}

func runPrivate(
	t *testing.T,
	tool *privateActionTool,
	configure func(*serviceports.RunRequest),
) *serviceports.RunResult {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_report", map[string]any{"name": "Late loads"}),
		textTurn("Saved."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	req := &serviceports.RunRequest{
		Definition: testDefinition("create_report"),
		Actor:      testActor(),
		Input:      "Save a report of late loads",
	}
	if configure != nil {
		configure(req)
	}

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)

	return result
}

// A report saved to the person's own list is theirs to save. An agent held
// to proposing used to put an approval card between them and what they
// asked for.
func TestRun_APrivateWriteRunsWhateverTheAgentsCeiling(t *testing.T) {
	t.Parallel()

	tool := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        true,
	}

	result := runPrivate(t, tool, nil)

	assert.Equal(t, agent.TierAutoExecute, result.Actions[0].Tier)
	assert.True(t, result.Actions[0].Executed)
}

func TestRun_AWriteOthersSeeStillWaitsAtTheCeiling(t *testing.T) {
	t.Parallel()

	tool := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        false,
	}

	result := runPrivate(t, tool, nil)

	assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)
}

// An administrator who set a tier for the tool on this agent chose it on
// purpose, and an unattended run has nobody the write could be private to.
func TestRun_APrivateWriteKeepsAnAgentsOwnSettingAndNeedsAPerson(t *testing.T) {
	t.Parallel()

	newTool := func() *privateActionTool {
		return &privateActionTool{
			StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
			private:        true,
		}
	}

	set := runPrivate(t, newTool(), func(req *serviceports.RunRequest) {
		req.Definition.ToolTiers = map[string]agent.AutonomyTier{"create_report": agent.TierPropose}
	})
	assert.Equal(t, agent.TierPropose, set.Actions[0].Tier)

	unattended := runPrivate(t, newTool(), func(req *serviceports.RunRequest) {
		req.Unattended = true
	})
	assert.Equal(t, agent.TierPropose, unattended.Actions[0].Tier)
}
