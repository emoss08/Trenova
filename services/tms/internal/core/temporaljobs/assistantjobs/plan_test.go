package assistantjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPlanRefusalTurnsAQuestionAwayAsAUsageLimit(t *testing.T) {
	t.Parallel()

	quota := errortypes.NewQuotaExceededError("ai.assistant_messages", 25, 25, "free_demo")
	require.True(t, rejected(quota))
	require.True(t, rejected(errortypes.NewPlanRestrictionError("", "subscription_read_only", "free_demo")))

	params := rejectionParams(quota)
	assert.Equal(t, planLimitCode, params["code"])
	assert.Equal(t, "ai.assistant_messages", params["meter"])

	limit := usageLimit(params)
	require.NotNil(t, limit)
	assert.Equal(t, planLimitKind, limit["kind"])
	assert.Equal(t, "25", limit["limit"])
	assert.Equal(t, "free_demo", limit["plan"])

	restricted := rejectionParams(errortypes.NewPlanRestrictionError("", "subscription_read_only", "free_demo"))
	assert.Equal(t, planRestrictedCode, restricted["code"])
	assert.Nil(t, usageLimit(restricted))
}

func TestAFailedTurnSaysWhichPlanLimitStoppedIt(t *testing.T) {
	t.Parallel()

	ending := failedEndingFor(conversation.AssistantTurnStatusFailed, failedMessage, &modelcall.Failure{
		Message: "limit",
		Plan: &modelcall.PlanRefusal{
			Code:    string(errortypes.ErrQuotaExceeded),
			Message: "This organization has reached the limit of its plan",
			Meter:   "ai.spend_cents",
			Limit:   150,
			Used:    151,
			Plan:    "free_demo",
		},
	})

	data, ok := ending.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, errorCodeUsageLimit, data["code"])
	assert.Equal(t, "This organization has reached the limit of its plan", data["message"])
	assert.Equal(t, "This organization has reached the limit of its plan", ending.Result.Message)
	limit, ok := data["limit"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "151", limit["used"])
	assert.Equal(t, "ai.spend_cents", limit["meter"])
}
