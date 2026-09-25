package assistantjobs

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantTurnPayload_OriginIsOptional(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.MarshalString(AssistantTurnPayload{})
	require.NoError(t, err)
	assert.NotContains(t, encoded, "traceOrigin")

	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	encoded, err = sonic.MarshalString(AssistantTurnPayload{Origin: traceparent})
	require.NoError(t, err)

	var decoded AssistantTurnPayload
	require.NoError(t, sonic.UnmarshalString(encoded, &decoded))
	assert.Equal(t, traceparent, decoded.Origin)
}
