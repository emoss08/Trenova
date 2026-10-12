package assistantjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestFinish_UsageLimitCarriesItsFigures(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	a, _ := finishActivities(turn, newLedger())
	in := finishInput(turn)
	in.Rejection = "You've used your AI allowance for this month (13 of 13 questions)."
	in.RejectionParams = map[string]string{
		"code": "person_allowance", "used": "13", "limit": "13", "resetsAt": "1793577600",
	}

	ending := finish(t, a, in)

	data, ok := ending.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, in.Rejection, data["message"])
	assert.Equal(t, errorCodeUsageLimit, data["code"])
	assert.Equal(t, map[string]any{
		"kind": "person_allowance", "used": "13", "limit": "13", "resetsAt": float64(1793577600),
	}, data["limit"])
}

func TestFinish_OtherRefusalsStayASentence(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	a, _ := finishActivities(turn, newLedger())

	ending := finish(t, a, finishInput(turn))

	data, ok := ending.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, data, "code")
}

func TestUsageLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{
		"kind": "monthly_budget", "used": "48.20", "limit": "50.00", "resetsAt": int64(0),
	}, usageLimit(map[string]string{
		"code": "agent_budget", "cap": "monthly_budget", "spent": "48.20", "limit": "50.00",
	}))
	assert.Nil(t, usageLimit(map[string]string{"code": "agent_budget", "cap": "tool_daily_limit"}))
	assert.Nil(t, usageLimit(nil))
}

func TestRejectionDetailsRoundTrip(t *testing.T) {
	t.Parallel()

	refused := errortypes.NewBusinessError("spent")
	refused.Params = map[string]string{"code": "person_allowance", "used": "3"}
	err := temporal.NewNonRetryableApplicationError(
		refused.Error(), errTypeRejected, refused, rejectionParams(refused),
	)

	assert.Equal(t, refused.Params, rejectionDetails(err))
	assert.Nil(t, rejectionDetails(errors.New("plain")))
}
