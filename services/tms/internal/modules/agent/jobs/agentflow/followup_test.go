package agentflow

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunContext_KeepsATurnTrenovaWroteAcrossTheActivityBoundary(t *testing.T) {
	t.Parallel()

	followUp := &agentruntimeports.RunRequest{
		Input:          "Decision on plan Dispatch desk: 2 changes: rejected.",
		TrenovaWritten: true,
		Asked:          "put a note on SEED-SHP-008 and SEED-SHP-010",
	}

	encoded, err := sonic.Marshal(NewRunContext(followUp, 0))
	require.NoError(t, err)
	var decoded RunContext
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))
	req := decoded.request()

	assert.True(t, req.TrenovaWritten)
	assert.False(t, req.AskedByPerson(),
		"the loop reads a decision follow-up as Trenova's note, not a person's question")
	assert.Equal(t, followUp.Asked, req.Ask(),
		"a proposal the follow-up makes answers what the person typed, not the note")

	asked := NewRunContext(&agentruntimeports.RunRequest{Input: "which loads are late?"}, 0)
	assert.True(t, asked.request().AskedByPerson())
}
