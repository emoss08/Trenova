package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

const (
	validTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	validSpanID  = "00f067aa0ba902b7"
)

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func baseRunStep() *agent.AgentRunStep {
	return &agent.AgentRunStep{
		OwnerKind: string(agent.RunOwnerAgentRun),
		OwnerID:   pulid.MustNew("ar_"),
		Kind:      "Tool",
		Status:    "Started",
		StepKey:   "step-key",
	}
}

func baseRun() *agent.AgentRun {
	return &agent.AgentRun{
		AgentType:        agent.TypeGeneral,
		SubjectType:      agent.SubjectShipment,
		SubjectID:        pulid.MustNew("shp_"),
		Trigger:          agent.RunTriggerManual,
		Status:           agent.RunStatusPending,
		PromptVersion:    "v1",
		InputContextHash: "hash",
	}
}

func TestRunOwnerKind_CoversBothOwners(t *testing.T) {
	t.Parallel()

	for _, kind := range agent.AllRunOwnerKinds() {
		assert.True(t, kind.IsValid(), kind)
	}
	assert.Len(t, agent.AllRunOwnerKinds(), 2)
	assert.False(t, agent.RunOwnerKind("Delegate").IsValid())
}

func TestTierSource_CoversEveryWayATierIsSet(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []agent.TierSource{
		agent.TierSourcePolicyDefault,
		agent.TierSourcePersonSetting,
		agent.TierSourceTrustEarned,
		agent.TierSourcePersonalExemption,
	}, agent.AllTierSources())
	for _, source := range agent.AllTierSources() {
		assert.True(t, source.IsValid(), source)
	}
	assert.False(t, agent.TierSource("").IsValid())
	assert.False(t, agent.TierSource("Earned").IsValid())
}

func TestAgentRunStep_Validate_AcceptsARowWithoutTraceLinks(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	baseRunStep().Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestAgentRunStep_Validate_ChecksTraceLinks(t *testing.T) {
	t.Parallel()

	valid := baseRunStep()
	version := int64(0)
	valid.TraceID = validTraceID
	valid.SpanID = validSpanID
	valid.AgentDefinitionID = pulid.MustNew("adef_")
	valid.AgentDefinitionVersion = &version
	valid.DelegateCallID = "call_1"

	multiErr := errortypes.NewMultiError()
	valid.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	invalid := baseRunStep()
	negative := int64(-1)
	invalid.TraceID = "4BF92F3577B34DA6A3CE929D0E0E4736"
	invalid.SpanID = "0000000000000000"
	invalid.AgentDefinitionVersion = &negative

	multiErr = errortypes.NewMultiError()
	invalid.Validate(multiErr)
	assert.ElementsMatch(t,
		[]string{"traceId", "spanId", "agentDefinitionVersion"},
		fieldsOf(multiErr))
}

func TestAgentProposal_Validate_ChecksTraceLinks(t *testing.T) {
	t.Parallel()

	proposal := baseProposal()
	proposal.Evidence = []agent.EvidenceRef{{Type: "shipment", ID: "shp_1"}}
	proposal.TraceID = validTraceID
	proposal.SpanID = validSpanID
	proposal.StepKey = "step-key"
	proposal.ExecutedByUserID = pulid.MustNew("usr_")

	multiErr := errortypes.NewMultiError()
	proposal.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	negative := int64(-1)
	proposal.TraceID = "trace"
	proposal.SpanID = "span"
	proposal.ExecutedTargetVersion = &negative

	multiErr = errortypes.NewMultiError()
	proposal.Validate(multiErr)
	assert.ElementsMatch(t,
		[]string{"traceId", "spanId", "executedTargetVersion"},
		fieldsOf(multiErr))
}

func TestAgentDecision_Validate_ChecksTraceID(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	decision := &agent.AgentDecision{
		ProposalID:      &proposalID,
		DecidedByUserID: pulid.MustNew("usr_"),
		Decision:        agent.DecisionAccepted,
		ReasonCode:      "looks_right",
		TraceID:         validTraceID,
	}

	multiErr := errortypes.NewMultiError()
	decision.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	decision.TraceID = "00000000000000000000000000000000"
	multiErr = errortypes.NewMultiError()
	decision.Validate(multiErr)
	assert.Equal(t, []string{"traceId"}, fieldsOf(multiErr))
}

func TestAgentRun_Validate_DelegateRunNamesItsParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*agent.AgentRun)
		fields []string
	}{
		{
			name:   "a run nobody handed a task passes",
			mutate: func(*agent.AgentRun) {},
		},
		{
			name: "a delegate run with its parent and call passes",
			mutate: func(r *agent.AgentRun) {
				r.TraceID = validTraceID
				r.TurnID = pulid.MustNew("atrn_")
				r.ParentOwnerKind = agent.RunOwnerAssistantTurn
				r.ParentOwnerID = pulid.MustNew("atrn_")
				r.DelegateCallID = "call_1"
			},
		},
		{
			name:   "a parent kind without an id fails",
			mutate: func(r *agent.AgentRun) { r.ParentOwnerKind = agent.RunOwnerAgentRun },
			fields: []string{"parentOwnerId"},
		},
		{
			name: "a parent id without a kind fails",
			mutate: func(r *agent.AgentRun) {
				r.ParentOwnerID = pulid.MustNew("ar_")
			},
			fields: []string{"parentOwnerId"},
		},
		{
			name:   "a delegate call without a parent fails",
			mutate: func(r *agent.AgentRun) { r.DelegateCallID = "call_1" },
			fields: []string{"delegateCallId"},
		},
		{
			name: "an unknown parent kind fails",
			mutate: func(r *agent.AgentRun) {
				r.ParentOwnerKind = agent.RunOwnerKind("Delegate")
				r.ParentOwnerID = pulid.MustNew("ar_")
			},
			fields: []string{"parentOwnerKind"},
		},
		{
			name:   "a malformed trace id fails",
			mutate: func(r *agent.AgentRun) { r.TraceID = "abc" },
			fields: []string{"traceId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			run := baseRun()
			tt.mutate(run)
			multiErr := errortypes.NewMultiError()
			run.Validate(multiErr)

			if len(tt.fields) == 0 {
				assert.False(t, multiErr.HasErrors(), multiErr.Error())
				return
			}
			assert.ElementsMatch(t, tt.fields, fieldsOf(multiErr))
		})
	}
}

func TestInsertStampsUpdatedAtForTheAuditProjector(t *testing.T) {
	t.Parallel()

	run := baseRun()
	require.NoError(t, run.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.NotZero(t, run.UpdatedAt)
	assert.Equal(t, run.CreatedAt, run.UpdatedAt)

	proposal := baseProposal()
	require.NoError(t, proposal.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.NotZero(t, proposal.UpdatedAt)
	assert.Equal(t, proposal.CreatedAt, proposal.UpdatedAt)
}
