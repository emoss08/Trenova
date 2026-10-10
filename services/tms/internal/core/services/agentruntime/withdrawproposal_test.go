package agentruntime

import (
	"context"
	"regexp"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var filedProposalID = regexp.MustCompile(`\(proposal (ap_[0-9A-Za-z]+)\)`)

// withdrawingCompletion answers from its script, except that the turn named
// in withdrawAt withdraws the first proposal the turn filed, read from that
// proposal's result as a model would read it.
type withdrawingCompletion struct {
	*scriptedCompletion

	withdrawAt int
}

func (c *withdrawingCompletion) CompleteChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	if c.CallCount != c.withdrawAt {
		return c.scriptedCompletion.CompleteChat(ctx, req)
	}
	c.LastReq = req
	c.Requests = append(c.Requests, req)
	c.CallCount++
	for _, message := range req.Messages {
		if match := filedProposalID.FindStringSubmatch(message.Content); match != nil {
			return toolTurn(withdrawProposalName, map[string]any{
				"proposalId": match[1],
				"reason":     "resent without the tractor type",
			}), nil
		}
	}

	return textTurn("Nothing to withdraw."), nil
}

func (c *withdrawingCompletion) StreamChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	_ serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	return c.CompleteChat(ctx, req)
}

func proposingRuntime(completion *scriptedCompletion) *Service {
	tool := &agentruntimetest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   shipmentSchema(),
	}

	return newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
}

func withdrawingRuntime(completion *withdrawingCompletion) *Service {
	rt := proposingRuntime(completion.scriptedCompletion)
	rt.completion = completion

	return rt
}

func proposingRequest() *serviceports.RunRequest {
	return &serviceports.RunRequest{
		Definition: testDefinition("create_shipment"),
		Actor:      testActor(),
		Input:      "enter the acme load",
		ThreadID:   pulid.MustNew("athr_"),
		Publishes:  true,
	}
}

/*
The corrected proposal replaces the first. Haiku proposed create_shipment, saw
it had sent a tractor type nobody asked for, proposed it again without one and
told the person to reject the first; the approval box offered both as one plan
and the load was entered twice.
*/
func TestWithdrawProposal_TakesBackTheProposalALaterOneReplaced(t *testing.T) {
	t.Parallel()

	completion := &withdrawingCompletion{
		scriptedCompletion: &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
			toolTurn("create_shipment", map[string]any{
				"customerId": "cus_1", "notes": "with a tractor type",
			}),
			toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
			nil,
			textTurn("I proposed the shipment."),
		}},
		withdrawAt: 2,
	}

	result, err := withdrawingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	require.Len(t, result.Actions, 2)
	assert.True(t, result.Actions[0].Withdrawn)
	assert.False(t, result.Actions[0].Waiting())
	assert.True(t, result.Actions[1].Waiting())

	content, failed := toolResult(result, withdrawProposalName)
	assert.False(t, failed)
	assert.Contains(t, content, "Withdrew proposal "+result.Actions[0].ProposalID.String())
	assert.Contains(t, content, "1 proposal is still waiting")

	assert.NotContains(t, offered(completion.Requests[0]), withdrawProposalName,
		"nothing is filed before the first proposal")
	assert.Contains(t, offered(completion.Requests[1]), withdrawProposalName)
}

func TestWithdrawProposal_RefusesWhatThisTurnDidNotFile(t *testing.T) {
	t.Parallel()

	earlier := waitingOutcome(agent.ProposalStatusPending)
	cases := []struct {
		name     string
		id       string
		contains string
	}{
		{name: "a proposal from an earlier turn", id: earlier.ProposalID.String(),
			contains: "filed in an earlier turn"},
		{name: "an id nobody filed", id: pulid.MustNew("ap_").String(),
			contains: "There is no proposal"},
		{name: "not an id", id: "the first one", contains: "proposalId must be the id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
				toolTurn(withdrawProposalName, map[string]any{"proposalId": tc.id}),
				textTurn("Done."),
			}}
			req := proposingRequest()
			req.Proposals = []serviceports.ProposalOutcome{earlier}

			result, err := proposingRuntime(completion).Run(t.Context(), req)
			require.NoError(t, err)

			content, failed := toolResult(result, withdrawProposalName)
			assert.True(t, failed)
			assert.Contains(t, content, tc.contains)
			require.Len(t, result.Actions, 1)
			assert.True(t, result.Actions[0].Waiting(), "a refused withdrawal takes nothing back")
		})
	}
}

func TestWithdrawProposal_IsOfferedOnlyOnceTheTurnHasFiledOne(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn(withdrawProposalName, map[string]any{"proposalId": pulid.MustNew("ap_").String()}),
		textTurn("Nothing to take back."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	assert.NotContains(t, offered(completion.Requests[0]), withdrawProposalName)
	content, failed := toolResult(result, withdrawProposalName)
	assert.True(t, failed)
	assert.Equal(t, unofferedWithdrawal, content)
}

func TestWithdrawProposal_IsNeverOfferedToADelegate(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
		textTurn("Proposed."),
	}}
	req := proposingRequest()
	req.Delegation = &serviceports.Delegation{CallID: "call_parent", StepScope: "d1"}

	_, err := proposingRuntime(completion).Run(t.Context(), req)
	require.NoError(t, err)

	for _, request := range completion.Requests {
		assert.False(t, slices.Contains(offered(request), withdrawProposalName))
	}
}

func TestWithdrawProposal_AWithdrawnProposalIsNoDuplicateOfItsReplacement(t *testing.T) {
	t.Parallel()

	same := map[string]any{"customerId": "cus_1"}
	completion := &withdrawingCompletion{
		scriptedCompletion: &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
			toolTurn("create_shipment", same),
			nil,
			toolTurn("create_shipment", same),
			textTurn("Proposed again."),
		}},
		withdrawAt: 1,
	}

	result, err := withdrawingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	require.Len(t, result.Actions, 2)
	assert.True(t, result.Actions[0].Withdrawn)
	assert.True(t, result.Actions[1].Waiting(),
		"the same change filed after its withdrawal is a new proposal, not a duplicate")
}
