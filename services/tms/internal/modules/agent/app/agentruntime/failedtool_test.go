package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuardedDispatch_AFailedCallsTextReachesTheModelInsideTheFence(t *testing.T) {
	t.Parallel()

	ledger := &stubLedger{
		Verdict: agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh},
	}
	result, _ := guardedRun(t, ledger, errors.New("driver is out of hours"))

	require.True(t, result.Messages[2].ToolFailed)
	message, fenced := toolresult.UnfenceFailure(result.Messages[2].Content)
	require.True(t, fenced)
	assert.Equal(t, "driver is out of hours", message)
}
