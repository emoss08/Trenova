package agentruntime

import (
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decisionObserver struct {
	seen  []agentruntimeports.ToolObservation
	err   error
	title string
}

func (o *decisionObserver) observe(
	observation agentruntimeports.ToolObservation,
) (*agentruntimeports.ShownArtifact, error) {
	if _, requested := observation.Data.(agentruntimeports.DecisionRequest); !requested {
		return nil, nil
	}
	o.seen = append(o.seen, observation)
	if o.err != nil {
		return nil, o.err
	}

	return &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "decision_request",
		Title: o.title,
	}, nil
}

func waitingOutcome(status agent.ProposalStatus) agentruntimeports.ProposalOutcome {
	return agentruntimeports.ProposalOutcome{
		ProposalID: pulid.MustNew("aprop_"),
		ToolName:   "shipment_create",
		Rationale:  "Copy PRO-100 for Tuesday.",
		Status:     status,
	}
}

type decisionRun struct {
	result     *agentruntimeports.RunResult
	completion *scriptedCompletion
	observer   *decisionObserver
}

func runDecision(
	t *testing.T,
	req *agentruntimeports.RunRequest,
	observer *decisionObserver,
	turns ...*llmports.ChatCompletionResult,
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

func chatRequest(proposals ...agentruntimeports.ProposalOutcome) *agentruntimeports.RunRequest {
	pending := make([]agentturn.PendingProposal, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal.Pending() {
			pending = append(pending, agentturn.PendingProposal{
				ProposalID: proposal.ProposalID,
				ToolName:   proposal.ToolName,
				Rationale:  proposal.Rationale,
			})
		}
	}

	return &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Context:    agentturn.RuntimeContext{PendingProposals: pending},
		Input:      "Approved",
		ThreadID:   pulid.MustNew("athr_"),
		Proposals:  proposals,
		Publishes:  true,
	}
}

func decisionCall(id pulid.ID) *llmports.ChatCompletionResult {
	return toolTurn(requestDecisionName, map[string]any{"proposalId": id.String()})
}

func toolResult(result *agentruntimeports.RunResult, name string) (string, bool) {
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
	offeredIn := func(req *agentruntimeports.RunRequest) bool {
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
	delegated.Delegation = &agentruntimeports.Delegation{CallID: "call_parent", StepScope: "d1"}
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
	assert.Equal(t, agentruntimeports.DecisionRequest{
		ProposalID: waiting.ProposalID,
		ToolName:   "shipment_create",
	}, observer.seen[0].Data)

	content, failed := toolResult(run.result, requestDecisionName)
	assert.False(t, failed)
	assert.Contains(t, content, "The approval box under the conversation now shows")
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
		call     *llmports.ChatCompletionResult
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
	twice := &llmports.ChatCompletionResult{ToolCalls: []llmports.ToolCall{
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
	assert.Contains(t, contents[1], "already in the approval box")
}

func TestRequestDecision_SaysWhenTheCardCouldNotBeShown(t *testing.T) {
	t.Parallel()

	waiting := waitingOutcome(agent.ProposalStatusPending)
	observer := &decisionObserver{err: errors.New("its card could not be saved")}

	run := runDecision(t, chatRequest(waiting), observer,
		decisionCall(waiting.ProposalID), textTurn("It is waiting above."))

	content, failed := toolResult(run.result, requestDecisionName)
	assert.True(t, failed)
	assert.Contains(
		t,
		content,
		"could not be opened in the approval box: its card could not be saved",
	)
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

	outcome := rt.PublishStep(observer.observe, &llmports.ToolCall{
		ID:        "call_decide",
		Name:      requestDecisionName,
		Arguments: map[string]any{"proposalId": id.String()},
	})

	require.Len(t, observer.seen, 1)
	assert.Equal(t, agentruntimeports.DecisionRequest{ProposalID: id}, observer.seen[0].Data)
	assert.False(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "The approval box under the conversation now shows")
	assert.Equal(t, "The approval box could not be opened just now. Tell the person the "+
		"proposal is still waiting on their decision in this conversation, and that "+
		"typing does not approve it.", UnkeptOutcome(requestDecisionName).Content)
}

func planStep(planID pulid.ID) agentruntimeports.ProposalOutcome {
	step := waitingOutcome(agent.ProposalStatusPending)
	step.ToolName = "invoice_post"
	step.PlanID = planID

	return step
}

func TestRequestDecision_ShowsAPlanAsOneCard(t *testing.T) {
	t.Parallel()

	planID := pulid.MustNew("apl_")
	first, second := planStep(planID), planStep(planID)
	observer := &decisionObserver{title: "Billing: 2 changes"}

	run := runDecision(t, chatRequest(first, second), observer,
		toolTurn(requestDecisionName, map[string]any{"planId": planID.String()}),
		textTurn("The plan is above."),
	)

	require.Len(t, observer.seen, 1)
	assert.Equal(t, agentruntimeports.DecisionRequest{
		ProposalID:  first.ProposalID,
		ProposalIDs: []pulid.ID{first.ProposalID, second.ProposalID},
		PlanID:      planID,
		ToolName:    agentplan.PlanToolName,
	}, observer.seen[0].Data)
	content, failed := toolResult(run.result, requestDecisionName)
	assert.False(t, failed)
	assert.Contains(t, content, "The approval box under the conversation now shows")
}

func TestRequestDecision_ShowsSeveralProposalsOfOneToolAsOneCard(t *testing.T) {
	t.Parallel()

	first := waitingOutcome(agent.ProposalStatusPending)
	second := waitingOutcome(agent.ProposalStatusPending)
	observer := &decisionObserver{title: "Create shipment"}

	run := runDecision(t, chatRequest(first, second), observer,
		toolTurn(requestDecisionName, map[string]any{
			"proposalIds": []any{first.ProposalID.String(), second.ProposalID.String()},
		}),
		textTurn("Both are above."),
	)

	require.Len(t, observer.seen, 1)
	assert.Equal(t, agentruntimeports.DecisionRequest{
		ProposalID:  first.ProposalID,
		ProposalIDs: []pulid.ID{first.ProposalID, second.ProposalID},
		ToolName:    "shipment_create",
	}, observer.seen[0].Data)
	_, failed := toolResult(run.result, requestDecisionName)
	assert.False(t, failed)
}

func TestRequestDecision_RefusesABunchItCannotShowAsOne(t *testing.T) {
	t.Parallel()

	shipment := waitingOutcome(agent.ProposalStatusPending)
	other := waitingOutcome(agent.ProposalStatusPending)
	other.ToolName = "invoice_post"
	decided := waitingOutcome(agent.ProposalStatusExecuted)
	planID := pulid.MustNew("apl_")
	step := planStep(planID)

	cases := []struct {
		name     string
		args     map[string]any
		contains string
	}{
		{
			name: "two tools",
			args: map[string]any{"proposalIds": []any{
				shipment.ProposalID.String(), other.ProposalID.String(),
			}},
			contains: "one tool at a time",
		},
		{
			name: "one already decided",
			args: map[string]any{"proposalIds": []any{
				shipment.ProposalID.String(), decided.ProposalID.String(),
			}},
			contains: "no longer waiting",
		},
		{
			name: "a plan's step among them",
			args: map[string]any{"proposalIds": []any{
				shipment.ProposalID.String(), step.ProposalID.String(),
			}},
			contains: "planId",
		},
		{
			name:     "a plan with nothing waiting",
			args:     map[string]any{"planId": pulid.MustNew("apl_").String()},
			contains: "no plan",
		},
		{
			name: "both a plan and proposals",
			args: map[string]any{
				"planId":     planID.String(),
				"proposalId": shipment.ProposalID.String(),
			},
			contains: "only one of",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			observer := &decisionObserver{title: "Card"}
			run := runDecision(t, chatRequest(shipment, other, decided, step), observer,
				toolTurn(requestDecisionName, tc.args), textTurn("Sorry."))

			content, failed := toolResult(run.result, requestDecisionName)
			assert.True(t, failed)
			assert.Contains(t, content, tc.contains)
			assert.Empty(t, observer.seen)
		})
	}
}

func TestPublishStep_ShowsAPlanADurableTurnAskedFor(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	observer := &decisionObserver{title: "Billing: 2 changes"}
	planID := pulid.MustNew("apl_")
	ids := []pulid.ID{pulid.MustNew("aprop_"), pulid.MustNew("aprop_")}

	rt.PublishStep(observer.observe, &llmports.ToolCall{
		ID:        "call_plan",
		Name:      requestDecisionName,
		Arguments: map[string]any{"planId": planID.String()},
	})
	rt.PublishStep(observer.observe, &llmports.ToolCall{
		ID:   "call_many",
		Name: requestDecisionName,
		Arguments: map[string]any{
			"proposalIds": []any{ids[0].String(), ids[1].String()},
		},
	})

	require.Len(t, observer.seen, 2)
	assert.Equal(t, agentruntimeports.DecisionRequest{PlanID: planID}, observer.seen[0].Data)
	assert.Equal(t, agentruntimeports.DecisionRequest{ProposalID: ids[0], ProposalIDs: ids},
		observer.seen[1].Data)
}
