package services

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvenanceFields_AreOmittedUntilTheyAreKnown(t *testing.T) {
	t.Parallel()

	action, err := sonic.MarshalString(PendingAction{ToolName: "assign_move"})
	require.NoError(t, err)
	for _, key := range []string{"proposalId", "traceId", "spanId", "tierSource", "executedVersion"} {
		assert.NotContains(t, action, `"`+key+`"`)
	}

	outcome, err := sonic.MarshalString(RunStepOutcome{Content: "done"})
	require.NoError(t, err)
	assert.NotContains(t, outcome, `"reason"`)

	attribution, err := sonic.MarshalString(AIUsageAttribution{RunID: pulid.ID("ar_1")})
	require.NoError(t, err)
	for _, key := range []string{"OwnerKind", "OwnerID", "DelegateCallID", "DefinitionVersion"} {
		assert.NotContains(t, attribution, `"`+key+`"`)
	}
}

func TestProvenanceFields_RoundTrip(t *testing.T) {
	t.Parallel()

	version := int64(0)
	executed := int64(7)
	action := PendingAction{
		ToolName:        "assign_move",
		ProposalID:      pulid.ID("ap_1"),
		TraceID:         "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:          "00f067aa0ba902b7",
		TierSource:      agent.TierSourceTrustEarned,
		ExecutedVersion: &executed,
	}
	attribution := AIUsageAttribution{
		OwnerKind:         RunStepOwnerAssistantTurn,
		OwnerID:           pulid.ID("atrn_1"),
		DelegateCallID:    "call_1",
		DefinitionVersion: &version,
	}

	var decodedAction PendingAction
	encoded, err := sonic.Marshal(action)
	require.NoError(t, err)
	require.NoError(t, sonic.Unmarshal(encoded, &decodedAction))
	assert.Equal(t, action, decodedAction)

	var decodedAttribution AIUsageAttribution
	encoded, err = sonic.Marshal(attribution)
	require.NoError(t, err)
	require.NoError(t, sonic.Unmarshal(encoded, &decodedAttribution))
	assert.Equal(t, attribution, decodedAttribution)
	require.NotNil(t, decodedAttribution.DefinitionVersion,
		"version zero is a version, not an absent one")
	assert.Zero(t, *decodedAttribution.DefinitionVersion)
}

func TestProvenanceFields_DecodeFromARecordedPayload(t *testing.T) {
	t.Parallel()

	var action PendingAction
	require.NoError(t, sonic.UnmarshalString(
		`{"toolName":"assign_move","arguments":{},"tier":"Propose","executed":false}`,
		&action,
	))
	assert.True(t, action.ProposalID.IsNil())
	assert.Empty(t, action.TraceID)
	assert.Empty(t, action.TierSource)
	assert.Nil(t, action.ExecutedVersion)

	var outcome RunStepOutcome
	require.NoError(t, sonic.UnmarshalString(`{"content":"done","failed":true}`, &outcome))
	assert.Empty(t, outcome.Reason)
}
