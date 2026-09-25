package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type taintingConversations struct {
	repositories.ConversationRepository

	marked []repositories.MarkThreadTaintedRequest
}

func (r *taintingConversations) MarkThreadTainted(
	_ context.Context,
	req repositories.MarkThreadTaintedRequest,
) error {
	r.marked = append(r.marked, req)

	return nil
}

func inboundTaint() *agent.RunTaint {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "get_inbound_message",
		CallID:   "call_1",
		Ref:      &agent.RecordRef{EntityType: agent.TaintEntityInboundMessage, ID: "imsg_1"},
	})

	return taint
}

func TestAdmit_ATurnOpensWithItsConversationsTaint(t *testing.T) {
	t.Parallel()

	svc := newService(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{})
	inherited := inboundTaint()

	decision, req := svc.admit(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Who is on load 12345?",
		ThreadID:   pulid.MustNew("athr_"),
		Taint:      inherited,
	})

	require.True(t, decision.Allowed)
	require.NotNil(t, req)
	assert.Equal(t, inherited.Marks, req.Taint.Marks)
	assert.NotSame(t, inherited, req.Taint)

	turn := svc.runtime.OpenTurn(t.Context(), req)
	assert.True(t, turn.Taint().Tainted(),
		"a decision's follow-up in this thread opens tainted too")
}

// A file the person attached is content written outside the organization: the
// turn that reads it is tainted, and so is the conversation from then on.
func TestSendMessage_AnAttachmentTaintsTheConversation(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("The rate confirmation is for Acme."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()
	doc := ownDocument(conversations.thread, actor)
	svc.documents = &stubDocuments{docs: map[pulid.ID]*document.Document{doc.ID: doc}}
	svc.contents = &stubContents{content: &documentcontent.Content{
		Status:      documentcontent.StatusExtracted,
		ContentText: "RATE CONFIRMATION Acme Foods $1,200",
	}}

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:              conversations.thread.ID,
		Content:               "Does this match what we quoted?",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: []pulid.ID{doc.ID},
	}, actor, nil)
	require.NoError(t, err)

	require.Len(t, conversations.tainted, 1)
	marks := conversations.tainted[0].Taint.Marks
	require.Len(t, marks, 1)
	assert.Equal(t, agent.TaintSourceAttachment, marks[0].Source)
	assert.Equal(t, doc.ID.String(), marks[0].Ref.ID)
	assert.True(t, conversations.thread.Tainted())
}

func TestSendMessage_AnUntaintedTurnWritesNoTaint(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Twelve loads are late."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "How many loads are late?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Empty(t, conversations.tainted)
	assert.False(t, conversations.thread.Tainted())
}

func TestKeepThreadTaint_RecordsWhatTheTurnReadOnce(t *testing.T) {
	t.Parallel()

	conversations := &taintingConversations{}
	svc := &Service{logger: zap.NewNop(), conversations: conversations}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := testActor().TenantInfo()

	svc.keepThreadTaint(t.Context(), thread, inboundTaint(), tenant)
	svc.keepThreadTaint(t.Context(), thread, inboundTaint(), tenant)
	svc.keepThreadTaint(t.Context(), thread, &agent.RunTaint{}, tenant)
	svc.keepThreadTaint(t.Context(), thread, nil, tenant)

	require.Len(t, conversations.marked, 1, "only new outside content is written")
	marked := conversations.marked[0]
	assert.Equal(t, thread.ID, marked.ThreadID)
	assert.Equal(t, inboundTaint().Marks, marked.Taint.Marks)
	assert.Positive(t, marked.TaintedAt)
	assert.True(t, thread.Tainted())
}

func TestTurnTaint_FallsBackToWhatTheTurnOpenedWith(t *testing.T) {
	t.Parallel()

	opened := inboundTaint()
	plan := &TurnPlan{Turn: agentruntime.TurnState{
		Result: serviceports.RunResult{Taint: opened},
	}}

	assert.Same(t, opened, turnTaint(plan, nil),
		"a turn that failed before the runtime answered still read what it opened with")

	ran := &agent.RunTaint{}
	assert.Same(t, ran, turnTaint(plan, &serviceports.RunResult{Taint: ran}))
}

func TestPersistDelegatedProposals_CarryTheDelegatesTaint(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	proposals := &stubProposalRepo{}
	svc := newProposalService(runs, proposals, &stubConversationRepo{})
	delegateTaint := inboundTaint()

	params := proposalTestParams(nil, nil)
	params.Taint = &agent.RunTaint{}
	_, err := svc.persistDelegatedProposals(t.Context(), params, []serviceports.DelegatedRun{{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Report Builder",
			AutonomyCeiling: agent.TierPropose,
		},
		CallID: "call_1",
		Input:  "Share the report with the customer.",
		Actions: []serviceports.PendingAction{{
			ToolName:  "share_report",
			Rationale: "The customer asked for it.",
			Tier:      agent.TierActWithApproval,
			Egress:    agent.EgressCustomerVisible,
			HeldBy:    []string{"egress_class"},
			Tainted:   true,
		}},
		Taint: delegateTaint,
	}})
	require.NoError(t, err)

	require.Len(t, runs.created, 1)
	assert.True(t, runs.created[0].Tainted, "the delegate's run is recorded as tainted")
	require.Len(t, proposals.created, 1)
	proposal := proposals.created[0]
	assert.True(t, proposal.Tainted)
	assert.Equal(t, delegateTaint.Marks, proposal.Taint.Marks)
	assert.Equal(t, agent.EgressCustomerVisible, proposal.EgressClass)
}

func TestPersistProposals_TieTheRunsToTheTurnAndADelegatesToItsTask(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	svc := newProposalService(runs, &stubProposalRepo{}, &stubConversationRepo{})
	turnID := pulid.MustNew("atrn_")

	params := proposalTestParams(nil, nil)
	params.TurnID = turnID
	params.Actions = []serviceports.PendingAction{{
		ToolName:  "assign_move",
		Rationale: "Cover the move.",
		Tier:      agent.TierPropose,
	}}
	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	_, err = svc.persistDelegatedProposals(t.Context(), params, []serviceports.DelegatedRun{{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Report Builder",
			AutonomyCeiling: agent.TierPropose,
		},
		CallID: "call_task_3",
		Input:  "Build the report.",
		Actions: []serviceports.PendingAction{{
			ToolName:  "create_report",
			Rationale: "Asked for it.",
			Tier:      agent.TierPropose,
		}},
	}})
	require.NoError(t, err)

	require.Len(t, runs.created, 2)
	own, delegated := runs.created[0], runs.created[1]
	assert.Equal(t,
		aitrace.AnchorFor(aitrace.AnchorAssistantTurn, turnID.String()).TraceID.String(),
		own.TraceID)
	assert.Equal(t, turnID, own.TurnID)
	assert.Empty(t, own.ParentOwnerKind, "the turn's own run was handed nothing")
	assert.Empty(t, own.DelegateCallID)

	assert.Equal(t, aitrace.ForDelegate(turnID, "call_task_3").TraceID.String(), delegated.TraceID)
	assert.Equal(t, turnID, delegated.TurnID)
	assert.Equal(t, agent.RunOwnerAssistantTurn, delegated.ParentOwnerKind)
	assert.Equal(t, turnID, delegated.ParentOwnerID)
	assert.Equal(t, "call_task_3", delegated.DelegateCallID)
}
