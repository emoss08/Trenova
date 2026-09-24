package proposalrecorder

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type openedRuns struct{ opened []*agent.AgentRun }

func (r *openedRuns) Create(_ context.Context, run *agent.AgentRun) (*agent.AgentRun, error) {
	run.ID = pulid.MustNew("arun_")
	r.opened = append(r.opened, run)

	return run, nil
}

func messageEvidence(serviceports.PendingAction, pulid.ID) []agent.EvidenceRef {
	return []agent.EvidenceRef{{Type: "message", ID: "amsg_1"}}
}

func readTaint() *agent.RunTaint {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceEDI,
		ToolName: "get_edi_inbound_file",
		CallID:   "call_1",
	})

	return taint
}

func TestRecord_KeepsTheRunsTaintAndEachProposalsEgress(t *testing.T) {
	t.Parallel()

	store := &capturingStore{}
	runs := &openedRuns{}
	taint := readTaint()
	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, err := NewWithStores(nil, runs, store).Record(t.Context(), &RecordRequest{
		Evidence: messageEvidence,
		Actor:    actor,
		Open: &OpenRunRequest{
			AgentType:        agent.TypeAssistantChat,
			SubjectType:      agent.SubjectAssistantThread,
			SubjectID:        pulid.MustNew("athr_"),
			Trigger:          agent.RunTriggerChat,
			PromptVersion:    "test",
			InputContextHash: "hash",
		},
		Taint: taint,
		Actions: []serviceports.PendingAction{
			{
				ToolName:  "add_shipment_comment",
				Arguments: map[string]any{},
				Rationale: "decided before the run read anything",
				Tier:      agent.TierAutoExecute,
				Executed:  true,
				Egress:    agent.EgressInternal,
			},
			{
				ToolName:  "post_customer_payment",
				Arguments: map[string]any{},
				Rationale: "decided after the run read the file",
				Tier:      agent.TierActWithApproval,
				Egress:    agent.EgressMoney,
				HeldBy:    []string{"tainted"},
				Tainted:   true,
			},
		},
	})
	require.NoError(t, err)

	require.Len(t, runs.opened, 1)
	run := runs.opened[0]
	assert.True(t, run.Tainted)
	require.NotNil(t, run.TaintedAt)
	assert.Equal(t, taint.Marks, run.Taint.Marks)

	require.Len(t, store.created, 2)
	before, after := store.created[0], store.created[1]
	assert.False(t, before.Tainted, "a write decided before the run read anything is clean")
	assert.Nil(t, before.Taint)
	assert.Equal(t, agent.EgressInternal, before.EgressClass)
	assert.Equal(t, []string{}, before.HeldBy)

	assert.True(t, after.Tainted)
	assert.Equal(t, taint.Marks, after.Taint.Marks)
	assert.NotSame(t, taint, after.Taint)
	assert.Equal(t, agent.EgressMoney, after.EgressClass)
	assert.Equal(t, []string{"tainted"}, after.HeldBy)
}

func TestRecord_ACleanRunIsNotMarked(t *testing.T) {
	t.Parallel()

	runs := &openedRuns{}
	_, err := NewWithStores(nil, runs, &capturingStore{}).Record(t.Context(), &RecordRequest{
		Evidence: messageEvidence,
		Actor: &serviceports.RequestActor{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Open: &OpenRunRequest{
			AgentType:        agent.TypeAssistantChat,
			SubjectType:      agent.SubjectAssistantThread,
			SubjectID:        pulid.MustNew("athr_"),
			PromptVersion:    "test",
			InputContextHash: "hash",
		},
		Taint:   &agent.RunTaint{},
		Actions: []serviceports.PendingAction{pendingAction("assign_move")},
	})
	require.NoError(t, err)

	require.Len(t, runs.opened, 1)
	assert.False(t, runs.opened[0].Tainted)
	assert.Nil(t, runs.opened[0].Taint)
	assert.Nil(t, runs.opened[0].TaintedAt)
}
