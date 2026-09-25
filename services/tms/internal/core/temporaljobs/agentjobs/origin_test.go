package agentjobs

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRunPayload_OriginIsOptional(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.MarshalString(AgentRunPayload{})
	require.NoError(t, err)
	assert.NotContains(t, encoded, "traceOrigin")

	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	encoded, err = sonic.MarshalString(AgentRunPayload{Origin: traceparent})
	require.NoError(t, err)

	var decoded AgentRunPayload
	require.NoError(t, sonic.UnmarshalString(encoded, &decoded))
	assert.Equal(t, traceparent, decoded.Origin)
}
