package proposalrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type turnFixture struct {
	runs      *openedRuns
	proposals *capturingStore
	plans     *capturingPlans
	svc       *Service
	actor     *serviceports.RequestActor
	parent    *agentdefinition.Definition
	delegate  *agentdefinition.Definition
	turnID    pulid.ID
	threadID  pulid.ID
}

func newTurnFixture() *turnFixture {
	runs, proposals, plans := &openedRuns{}, &capturingStore{}, &capturingPlans{}

	return &turnFixture{
		runs:      runs,
		proposals: proposals,
		plans:     plans,
		svc:       NewWithStores(nil, runs, proposals).WithPlans(plans),
		actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		parent:   &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch"},
		delegate: &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Shipment Desk"},
		turnID:   pulid.MustNew("atrn_"),
		threadID: pulid.MustNew("athr_"),
	}
}

func (f *turnFixture) open(delegateCallID string) *OpenRunRequest {
	open := &OpenRunRequest{
		AgentType:        agent.TypeAssistantChat,
		SubjectType:      agent.SubjectAssistantThread,
		SubjectID:        f.threadID,
		Trigger:          agent.RunTriggerChat,
		Status:           agent.RunStatusCompleted,
		PromptVersion:    "assistant-chat/v2",
		InputContextHash: "hash",
		TurnID:           f.turnID,
	}
	if delegateCallID != "" {
		open.ParentOwnerKind = agent.RunOwnerAssistantTurn
		open.ParentOwnerID = f.turnID
		open.DelegateCallID = delegateCallID
	}

	return open
}

func calledAction(tool, callID string) serviceports.PendingAction {
	action := pendingAction(tool)
	action.ToolCallID = callID

	return action
}

/*
"Void and recreate": the Dispatch desk proposed voiding the old shipment and
handed the Shipment Desk the copy. Both writes wait on the person, so they are
one plan, run in the order the conversation asked for them, while each stays
the proposal of the agent that filed it.
*/
func TestRecord_FoldsADelegatesWritesIntoTheTurnsOnePlanInConversationOrder(t *testing.T) {
	t.Parallel()

	f := newTurnFixture()
	result, err := f.svc.Record(t.Context(), &RecordRequest{
		Actor:      f.actor,
		Definition: f.parent,
		Open:       f.open(""),
		Actions:    []serviceports.PendingAction{calledAction("void_shipment", "call_void")},
		Delegated: []DelegatedActions{{
			Definition: f.delegate,
			Open:       f.open("call_hand"),
			Actions: []serviceports.PendingAction{
				calledAction("duplicate_shipment", "call_copy"),
			},
		}},
		CallOrder: map[string]int{"call_void": 0, "call_hand": 1, "call_copy": 2},
		Evidence:  messageEvidence,
	})
	require.NoError(t, err)

	require.Len(t, f.runs.opened, 2, "one run for each agent that filed a write")
	parentRun, delegateRun := f.runs.opened[0], f.runs.opened[1]
	assert.Equal(t, f.parent.ID, parentRun.AgentDefinitionID)
	assert.Equal(t, f.delegate.ID, delegateRun.AgentDefinitionID)
	assert.Equal(t, agent.RunOwnerAssistantTurn, delegateRun.ParentOwnerKind)
	assert.Equal(t, "call_hand", delegateRun.DelegateCallID)

	require.Len(t, f.plans.created, 1, "one plan for the turn")
	plan := f.plans.created[0]
	assert.Equal(t, parentRun.ID, plan.RunID)
	assert.Equal(t, 2, plan.StepCount)
	assert.Same(t, plan, result.Plan)

	require.Len(t, f.proposals.created, 2)
	void, copied := f.proposals.created[0], f.proposals.created[1]
	assert.Equal(t, "void_shipment", void.ToolName)
	assert.Equal(t, parentRun.ID, void.RunID)
	assert.Equal(t, 1, void.PlanStep)
	assert.Equal(t, "duplicate_shipment", copied.ToolName)
	assert.Equal(t, delegateRun.ID, copied.RunID, "the delegate's write stays its own")
	assert.Equal(t, 2, copied.PlanStep)
	assert.Equal(t, plan.ID, *copied.PlanID)

	assert.Same(t, parentRun, result.Run)
	require.Len(t, result.Proposals, 1)
	require.Len(t, result.Delegated, 1)
	assert.Same(t, delegateRun, result.Delegated[0].Run)
	assert.Equal(t, []*agent.AgentProposal{copied}, result.Delegated[0].Proposals)
}

// Steps follow the conversation, not who filed them: a copy the delegate
// proposed before the parent proposed the void is step one.
func TestRecord_NumbersTheStepsByWhereTheConversationAskedForThem(t *testing.T) {
	t.Parallel()

	f := newTurnFixture()
	_, err := f.svc.Record(t.Context(), &RecordRequest{
		Actor:      f.actor,
		Definition: f.parent,
		Open:       f.open(""),
		Actions:    []serviceports.PendingAction{calledAction("void_shipment", "call_void")},
		Delegated: []DelegatedActions{{
			Definition: f.delegate,
			Open:       f.open("call_hand"),
			Actions: []serviceports.PendingAction{
				calledAction("duplicate_shipment", "call_copy"),
				calledAction("add_shipment_comment", "call_note"),
			},
		}},
		CallOrder: map[string]int{
			"call_hand": 0, "call_copy": 1, "call_note": 2, "call_void": 3,
		},
		Evidence: messageEvidence,
	})
	require.NoError(t, err)

	steps := make(map[string]int, len(f.proposals.created))
	for _, proposal := range f.proposals.created {
		steps[proposal.ToolName] = proposal.PlanStep
	}
	assert.Equal(t, map[string]int{
		"duplicate_shipment":   1,
		"add_shipment_comment": 2,
		"void_shipment":        3,
	}, steps)
	require.Len(t, f.plans.created, 1)
	assert.Equal(t, 3, f.plans.created[0].StepCount)
}

// #628: a hand-off that wrote nothing still leaves the delegate's run, so
// AI Control lists the work it did; the parent, which filed nothing, opens
// none.
func TestRecord_KeepsARunForAReadOnlyHandOff(t *testing.T) {
	t.Parallel()

	f := newTurnFixture()
	result, err := f.svc.Record(t.Context(), &RecordRequest{
		Actor:         f.actor,
		Definition:    f.parent,
		Open:          f.open(""),
		OpenEmptyRuns: true,
		Delegated: []DelegatedActions{{
			Definition: f.delegate,
			Open:       f.open("call_hand"),
		}},
	})
	require.NoError(t, err)

	require.Len(t, f.runs.opened, 1)
	run := f.runs.opened[0]
	assert.Equal(t, f.delegate.ID, run.AgentDefinitionID)
	assert.Equal(t, agent.RunStatusCompleted, run.Status)
	assert.Equal(t, "call_hand", run.DelegateCallID)
	assert.Empty(t, f.proposals.created)
	assert.Empty(t, f.plans.created)
	assert.Nil(t, result.Run)
	require.Len(t, result.Delegated, 1)
	assert.Same(t, run, result.Delegated[0].Run)
}

// Without the option, a hand-off that wrote nothing is left as it always was.
func TestRecord_OpensNoRunForAnEmptyHandOffUnlessAsked(t *testing.T) {
	t.Parallel()

	f := newTurnFixture()
	result, err := f.svc.Record(t.Context(), &RecordRequest{
		Actor:      f.actor,
		Definition: f.parent,
		Open:       f.open(""),
		Delegated:  []DelegatedActions{{Definition: f.delegate, Open: f.open("call_hand")}},
	})
	require.NoError(t, err)

	assert.Empty(t, f.runs.opened)
	assert.Empty(t, result.Delegated)
}

// One pending write stands alone, whoever filed it, and a write that already
// ran is no step of anything.
func TestRecord_ADelegatesOnlyPendingWriteStandsAlone(t *testing.T) {
	t.Parallel()

	f := newTurnFixture()
	executed := calledAction("add_shipment_comment", "call_note")
	executed.Executed = true
	_, err := f.svc.Record(t.Context(), &RecordRequest{
		Actor:      f.actor,
		Definition: f.parent,
		Open:       f.open(""),
		Actions:    []serviceports.PendingAction{executed},
		Delegated: []DelegatedActions{{
			Definition: f.delegate,
			Open:       f.open("call_hand"),
			Actions: []serviceports.PendingAction{
				calledAction("duplicate_shipment", "call_copy"),
			},
		}},
		Evidence: messageEvidence,
	})
	require.NoError(t, err)

	assert.Empty(t, f.plans.created)
	require.Len(t, f.proposals.created, 2)
	for _, proposal := range f.proposals.created {
		assert.Nil(t, proposal.PlanID)
	}
}
