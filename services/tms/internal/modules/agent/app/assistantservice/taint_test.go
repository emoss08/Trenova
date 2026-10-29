package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type taintingConversations struct {
	conversationports.ConversationRepository

	marked []conversationports.MarkThreadTaintedRequest
}

func (r *taintingConversations) MarkThreadTainted(
	_ context.Context,
	req conversationports.MarkThreadTaintedRequest,
) error {
	r.marked = append(r.marked, req)

	return nil
}

func inboundTaint() *agent.RunTaint {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "inbox_get_inbound_message",
		CallID:   "call_1",
		Ref:      &agentrecord.RecordRef{EntityType: agent.TaintEntityInboundMessage, ID: "imsg_1"},
	})

	return taint
}

func TestAdmit_ATurnOpensWithItsConversationsTaint(t *testing.T) {
	t.Parallel()

	svc := newService(
		&assistanttest.ScriptedCompletion{},
		&assistanttest.StubQueryRegistry{},
		&assistanttest.StubActionRegistry{},
	)
	inherited := inboundTaint()

	decision, req := svc.admit(t.Context(), &TurnRequest{
		Definition: assistanttest.Definition(),
		Actor:      assistanttest.Actor(),
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

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("The rate confirmation is for Acme."),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	actor := assistanttest.Actor()
	doc := ownDocument(conversations.Thread, actor)
	svc.documents = &stubDocuments{docs: map[pulid.ID]*document.Document{doc.ID: doc}}
	svc.contents = &stubContents{content: &documentcontent.Content{
		Status:      documentcontent.StatusExtracted,
		ContentText: "RATE CONFIRMATION Acme Foods $1,200",
	}}

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:              conversations.Thread.ID,
		Content:               "Does this match what we quoted?",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: []pulid.ID{doc.ID},
	}, actor, nil)
	require.NoError(t, err)

	require.Len(t, conversations.Tainted, 1)
	marks := conversations.Tainted[0].Taint.Marks
	require.Len(t, marks, 1)
	assert.Equal(t, agent.TaintSourceAttachment, marks[0].Source)
	assert.Equal(t, doc.ID.String(), marks[0].Ref.ID)
	assert.True(t, conversations.Thread.Tainted())
}

func TestSendMessage_AnUntaintedTurnWritesNoTaint(t *testing.T) {
	t.Parallel()

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("Twelve loads are late."),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:   conversations.Thread.ID,
		Content:    "How many loads are late?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Empty(t, conversations.Tainted)
	assert.False(t, conversations.Thread.Tainted())
}

func TestKeepThreadTaint_RecordsWhatTheTurnReadOnce(t *testing.T) {
	t.Parallel()

	conversations := &taintingConversations{}
	svc := &Service{logger: zap.NewNop(), conversations: conversations}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := assistanttest.Actor().TenantInfo()

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
		Result: agentruntimeports.RunResult{Taint: opened},
	}}

	assert.Same(t, opened, turnTaint(plan, nil),
		"a turn that failed before the runtime answered still read what it opened with")

	ran := &agent.RunTaint{}
	assert.Same(t, ran, turnTaint(plan, &agentruntimeports.RunResult{Taint: ran}))
}

func TestPersistProposals_ADelegatesWritesCarryItsTaint(t *testing.T) {
	t.Parallel()

	runs := &assistanttest.StubRunRepo{}
	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(runs, proposals, &assistanttest.StubConversationRepo{})
	delegateTaint := inboundTaint()

	params := proposalTestParams(nil, nil)
	params.Taint = &agent.RunTaint{}
	params.Delegations = []agentruntimeports.DelegatedRun{{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Report Builder",
			AutonomyCeiling: agent.TierPropose,
		},
		CallID: "call_1",
		Input:  "Share the report with the customer.",
		Actions: []agentruntimeports.PendingAction{{
			ToolName:  "share_report",
			Rationale: "The customer asked for it.",
			Tier:      agent.TierActWithApproval,
			Egress:    agent.EgressCustomerVisible,
			HeldBy:    []string{"egress_class"},
			Tainted:   true,
		}},
		Taint: delegateTaint,
	}}
	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, runs.Created, 1)
	assert.True(t, runs.Created[0].Tainted, "the delegate's run is recorded as tainted")
	require.Len(t, proposals.Created, 1)
	proposal := proposals.Created[0]
	assert.True(t, proposal.Tainted)
	assert.Equal(t, delegateTaint.Marks, proposal.Taint.Marks)
	assert.Equal(t, agent.EgressCustomerVisible, proposal.EgressClass)
}

func TestPersistProposals_TieTheRunsToTheTurnAndADelegatesToItsTask(t *testing.T) {
	t.Parallel()

	runs := &assistanttest.StubRunRepo{}
	svc := newProposalService(
		runs,
		&assistanttest.StubProposalRepo{},
		&assistanttest.StubConversationRepo{},
	)
	turnID := pulid.MustNew("atrn_")

	params := proposalTestParams(nil, nil)
	params.TurnID = turnID
	params.Actions = []agentruntimeports.PendingAction{{
		ToolName:  "dispatch_assign_move",
		Rationale: "Cover the move.",
		Tier:      agent.TierPropose,
	}}
	params.Delegations = []agentruntimeports.DelegatedRun{{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Report Builder",
			AutonomyCeiling: agent.TierPropose,
		},
		CallID: "call_task_3",
		Input:  "Build the report.",
		Actions: []agentruntimeports.PendingAction{{
			ToolName:  "report_create",
			Rationale: "Asked for it.",
			Tier:      agent.TierPropose,
		}},
	}}
	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, runs.Created, 2)
	own, delegated := runs.Created[0], runs.Created[1]
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
