package agentruntime

import (
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decisionObserver struct {
	seen  []serviceports.ToolObservation
	err   error
	title string
}

func (o *decisionObserver) observe(
	observation serviceports.ToolObservation,
) (*serviceports.ShownArtifact, error) {
	if _, requested := observation.Data.(serviceports.DecisionRequest); !requested {
		return nil, nil
	}
	o.seen = append(o.seen, observation)
	if o.err != nil {
		return nil, o.err
	}

	return &serviceports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "decision_request",
		Title: o.title,
	}, nil
}

func waitingOutcome(status agent.ProposalStatus) serviceports.ProposalOutcome {
	return serviceports.ProposalOutcome{
		ProposalID: pulid.MustNew("aprop_"),
		ToolName:   "create_shipment",
		Rationale:  "Copy PRO-100 for Tuesday.",
		Status:     status,
	}
}

type decisionRun struct {
	result     *serviceports.RunResult
	completion *scriptedCompletion
	observer   *decisionObserver
}

func runDecision(
	t *testing.T,
	req *serviceports.RunRequest,
	observer *decisionObserver,
	turns ...*serviceports.ChatCompletionResult,
) decisionRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	if observer != nil {
		req.ToolObserver = observer.observe
	}
	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return decisionRun{result: result, completion: completion, observer: observer}
}

func chatRequest(proposals ...serviceports.ProposalOutcome) *serviceports.RunRequest {
	pending := make([]agentdefinition.PendingProposal, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal.Pending() {
			pending = append(pending, agentdefinition.PendingProposal{
				ProposalID: proposal.ProposalID,
				ToolName:   proposal.ToolName,
				Rationale:  proposal.Rationale,
			})
		}
	}

	return &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Context:    agentdefinition.RuntimeContext{PendingProposals: pending},
		Input:      "Approved",
		ThreadID:   pulid.MustNew("athr_"),
		Proposals:  proposals,
		Publishes:  true,
	}
}

func decisionCall(id pulid.ID) *serviceports.ChatCompletionResult {
	return toolTurn(requestDecisionName, map[string]any{"proposalId": id.String()})
}

func toolResult(result *serviceports.RunResult, name string) (string, bool) {
	for _, message := range result.Messages {
		if message.ToolName == name {
			return message.Content, message.ToolFailed
		}
	}

	return "", false
}

func TestRequestDecision_IsOfferedOnlyWhileAProposalWaitsInAConversation(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	offeredIn := func(req *serviceports.RunRequest) bool {
		run := runDecision(t, req, nil, textTurn("Done."))

		return slices.Contains(offered(run.completion.Requests[0]), requestDecisionName)
	}

	assert.True(t, offeredIn(chatRequest(waiting)))
	assert.False(t, offeredIn(chatRequest()), "nothing waits")
	assert.False(t, offeredIn(chatRequest(waitingOutcome(agent.ProposalStatusExecuted))))

	unattended := chatRequest(waiting)
	unattended.Unattended = true
	assert.False(t, offeredIn(unattended), "nobody is there to decide")

	delegated := chatRequest(waiting)
	delegated.Delegation = &serviceports.Delegation{CallID: "call_parent", StepScope: "d1"}
	assert.False(t, offeredIn(delegated), "a delegate is not talking to the person")

	nowhere := chatRequest(waiting)
	nowhere.Publishes = false
	assert.False(t, offeredIn(nowhere), "there is nowhere to show the card")
}

func TestRequestDecision_ShowsTheCardAgainAndSaysTypingDecidesNothing(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	observer := &decisionObserver{title: "Create shipment"}

	run := runDecision(t, chatRequest(waiting), observer,
		decisionCall(waiting.ProposalID),
		textTurn("The card is above: approve it there."),
	)

	require.Len(t, observer.seen, 1)
	assert.Equal(t, serviceports.DecisionRequest{
		ProposalID: waiting.ProposalID,
		ToolName:   "create_shipment",
	}, observer.seen[0].Data)

	content, failed := toolResult(run.result, requestDecisionName)
	assert.False(t, failed)
	assert.Contains(t, content, "in front of the person again")
	assert.Contains(t, content, "does not decide it")
	assert.Empty(t, run.result.Actions, "asking for a decision makes none")
	assert.Contains(t, run.completion.Requests[0].System, "call request_decision")
	assert.Contains(t, run.completion.Requests[0].System, waiting.ProposalID.String())
}

func TestRequestDecision_RefusesWhatIsNotWaitingHere(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	decided := waitingOutcome(agent.ProposalStatusRejected)

	cases := []struct {
		name     string
		call     *serviceports.ChatCompletionResult
		contains string
	}{
		{
			name:     "an id that is not in the conversation",
			call:     decisionCall(pulid.MustNew("aprop_")),
			contains: "There is no proposal",
		},
		{
			name:     "a proposal already decided",
			call:     decisionCall(decided.ProposalID),
			contains: "no longer waiting on the person: it is rejected",
		},
		{
			name:     "no id at all",
			call:     toolTurn(requestDecisionName, map[string]any{"proposalId": "yes"}),
			contains: "proposalId must be the id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			observer := &decisionObserver{title: "Create shipment"}
			run := runDecision(t, chatRequest(waiting, decided), observer,
				tc.call, textTurn("Sorry."))

			content, failed := toolResult(run.result, requestDecisionName)
			assert.True(t, failed)
			assert.Contains(t, content, tc.contains)
			assert.Empty(t, observer.seen, "nothing is shown for a refused request")
		})
	}
}

func TestRequestDecision_ShowsOneCardOncePerTurn(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	observer := &decisionObserver{title: "Create shipment"}
	twice := &serviceports.ChatCompletionResult{ToolCalls: []serviceports.ToolCall{
		{ID: "call_a", Name: requestDecisionName, Arguments: map[string]any{
			"proposalId": waiting.ProposalID.String(),
		}},
		{ID: "call_b", Name: requestDecisionName, Arguments: map[string]any{
			"proposalId": waiting.ProposalID.String(),
		}},
	}}

	run := runDecision(t, chatRequest(waiting), observer, twice, textTurn("Above."))

	assert.Len(t, observer.seen, 1)
	contents := make([]string, 0, 2)
	for _, message := range run.result.Messages {
		if message.ToolName == requestDecisionName {
			contents = append(contents, message.Content)
		}
	}
	require.Len(t, contents, 2)
	assert.Contains(t, contents[1], "already in front of the person")
}

func TestRequestDecision_SaysWhenTheCardCouldNotBeShown(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	observer := &decisionObserver{err: errors.New("its card could not be saved")}

	run := runDecision(t, chatRequest(waiting), observer,
		decisionCall(waiting.ProposalID), textTurn("It is waiting above."))

	content, failed := toolResult(run.result, requestDecisionName)
	assert.True(t, failed)
	assert.Contains(t, content, "could not be shown again: its card could not be saved")
	assert.Contains(t, content, "typing does not approve it")
}

func TestRequestDecision_NotOfferedIsRefused(t *testing.T) {
	t.Parallel()

	run := runDecision(t, chatRequest(), &decisionObserver{},
		decisionCall(pulid.MustNew("aprop_")), textTurn("Nothing waits."))

	content, failed := toolResult(run.result, requestDecisionName)
	assert.True(t, failed)
	assert.Equal(t, unofferedDecisionRefusal, content)
}

func TestPublishStep_ShowsTheCardADurableTurnAskedFor(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	observer := &decisionObserver{title: "Create shipment"}
	id := pulid.MustNew("aprop_")

	outcome := rt.PublishStep(observer.observe, &serviceports.ToolCall{
		ID:        "call_decide",
		Name:      requestDecisionName,
		Arguments: map[string]any{"proposalId": id.String()},
	})

	require.Len(t, observer.seen, 1)
	assert.Equal(t, serviceports.DecisionRequest{ProposalID: id}, observer.seen[0].Data)
	assert.False(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "in front of the person again")
	assert.Equal(t, "The card could not be shown again just now. Tell the person the "+
		"proposal is still waiting on its card earlier in this conversation, and that "+
		"typing does not approve it.", UnkeptOutcome(requestDecisionName).Content)
}
