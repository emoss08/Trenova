package agentruntime

import (
	"context"
	"regexp"
	"slices"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	req *agentports.ChatCompletionRequest,
) (*agentports.ChatCompletionResult, error) {
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
	req *agentports.ChatCompletionRequest,
	_ agentports.ChatStreamSink,
) (*agentports.ChatCompletionResult, error) {
	return c.CompleteChat(ctx, req)
}

func proposingRuntime(completion *scriptedCompletion) *Service {
	tool := &agenttest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   shipmentSchema(),
	}

	return newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
}

func withdrawingRuntime(completion *withdrawingCompletion) *Service {
	rt := proposingRuntime(completion.scriptedCompletion)
	rt.completion = completion

	return rt
}

func proposingRequest() *agentports.RunRequest {
	return &agentports.RunRequest{
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
		scriptedCompletion: &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
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

	assert.Contains(t, offered(completion.Requests[0]), withdrawProposalName,
		"declared before the first proposal, so the tool list never changes")
	assertSameToolsEveryCall(t, completion.Requests)
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

			completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
				toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
				toolTurn(withdrawProposalName, map[string]any{"proposalId": tc.id}),
				textTurn("Done."),
			}}
			req := proposingRequest()
			req.Proposals = []agentports.ProposalOutcome{earlier}

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

func TestWithdrawProposal_IsRefusedUntilTheTurnHasFiledOne(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn(withdrawProposalName, map[string]any{"proposalId": pulid.MustNew("ap_").String()}),
		textTurn("Nothing to take back."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	assert.Contains(t, offered(completion.Requests[0]), withdrawProposalName)
	content, failed := toolResult(result, withdrawProposalName)
	assert.True(t, failed)
	assert.Equal(t, unofferedWithdrawal, content)
}

func TestWithdrawProposal_IsNotDeclaredWhereNothingCanBeProposed(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		textTurn("Nothing to do."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agentports.AgentQueryTool{queryTool("get_shipment", map[string]any{}, nil)},
	}, &stubActionRegistry{}, nil)

	_, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "where is S-1",
	})
	require.NoError(t, err)
	assert.NotContains(t, offered(completion.Requests[0]), withdrawProposalName)
}

/*
The tool list is the head of the cached prompt, and on Opus, Sonnet and Haiku
5.5 every replayed thinking block is bound to it. A turn that proposes, then
keeps working, then withdraws and proposes again sends the same tools on every
model call.
*/
func TestTurn_SendsTheSameToolsOnEveryModelCall(t *testing.T) {
	t.Parallel()

	completion := &withdrawingCompletion{
		scriptedCompletion: &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("create_shipment", map[string]any{"customerId": "cus_1", "notes": "a"}),
			toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
			nil,
			textTurn("I proposed the shipment."),
		}},
		withdrawAt: 2,
	}

	_, err := withdrawingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)
	require.Len(t, completion.Requests, 4)
	assertSameToolsEveryCall(t, completion.Requests)
}

func assertSameToolsEveryCall(t *testing.T, requests []*agentports.ChatCompletionRequest) {
	t.Helper()

	require.NotEmpty(t, requests)
	first, err := sonic.ConfigStd.Marshal(requests[0].Tools)
	require.NoError(t, err)
	for idx, request := range requests[1:] {
		next, marshalErr := sonic.ConfigStd.Marshal(request.Tools)
		require.NoError(t, marshalErr)
		assert.JSONEq(t, string(first), string(next), "model call %d changed the tools", idx+2)
	}
}

func TestWithdrawProposal_IsNeverOfferedToADelegate(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
		textTurn("Proposed."),
	}}
	req := proposingRequest()
	req.Delegation = &agentports.Delegation{CallID: "call_parent", StepScope: "d1"}

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
		scriptedCompletion: &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
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
