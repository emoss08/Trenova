package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedShadow struct {
	verdict agentshadow.Verdict
	err     error
	tenant  pagination.TenantInfo
}

func (f *fixedShadow) ForDefinition(
	_ context.Context,
	tenant pagination.TenantInfo,
	_ *agentdefinition.Definition,
) (agentshadow.Verdict, error) {
	f.tenant = tenant

	return f.verdict, f.err
}

func proposedContent(t *testing.T, shadow shadowVerdicts) string {
	t.Helper()

	tool := &agenttest.StubActionTool{
		ToolName: "home_add_widget",
		Tier:     agent.TierActWithApproval,
	}
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("home_add_widget", map[string]any{"key": "fleet-status", "version": 0}),
		textTurn("I drafted it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
	rt.shadow = shadow

	result, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("home_add_widget"),
		Actor:      testActor(),
		Input:      "add the fleet status widget to my home page",
	})
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)

	for _, message := range result.Messages {
		if message.ToolName == "home_add_widget" {
			return message.Content
		}
	}
	require.FailNow(t, "the proposal left no tool result")

	return ""
}

func TestDispatch_AProposalNothingHoldsAwaitsReview(t *testing.T) {
	t.Parallel()

	content := proposedContent(t, &fixedShadow{})

	assert.Contains(t, content, "It is awaiting a person's review at the")
	assert.NotContains(t, content, "no one can approve it yet")
}

func TestDispatch_AProposalUnderTheOrganizationPauseSaysAgentsMustBeResumed(t *testing.T) {
	t.Parallel()

	shadow := &fixedShadow{verdict: agentshadow.Verdict{Cause: agentshadow.CauseOrganization}}
	content := proposedContent(t, shadow)

	assert.Contains(t, content, "It has not run, and no one can approve it yet: every agent "+
		"in this organization is paused in AI Control, so a person must resume agents there "+
		"before anyone can approve it.")
	assert.Contains(t, content, "do not say it waits on their approval")
	assert.NotContains(t, content, "awaiting a person's review")
	assert.True(t, shadow.tenant.OrgID.IsNotNil(), "the hold is read for the caller's tenant")
	assert.True(t, shadow.tenant.BuID.IsNotNil(), "the hold is read for the caller's tenant")
}

func TestDispatch_AProposalOfAnAgentInShadowSaysItsSwitchMustBeTurnedOff(t *testing.T) {
	t.Parallel()

	content := proposedContent(t, &fixedShadow{verdict: agentshadow.Verdict{
		Cause:     agentshadow.CauseDefinition,
		AgentName: "Dispatch desk",
	}})

	assert.Contains(t, content, "It has not run, and no one can approve it yet: this agent is "+
		"in shadow, so its shadow switch must be turned off in AI Control before anyone can "+
		"approve it.")
	assert.NotContains(t, content, "awaiting a person's review")
}

func TestDispatch_AProposalWhoseHoldCannotBeReadKeepsTheUsualWording(t *testing.T) {
	t.Parallel()

	content := proposedContent(t, &fixedShadow{err: errors.New("agent control unreadable")})

	assert.Contains(t, content, "It is awaiting a person's review at the")
}
