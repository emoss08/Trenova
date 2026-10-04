package temporaltype

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestToPlanRefusalMakesQuotaAndPlanErrorsNonRetryable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		err      error
		expected ErrorType
	}{
		{errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo"), ErrorTypeQuotaExceeded},
		{errortypes.NewPlanRestrictionError("sms", "", "free_demo"), ErrorTypePlanRestricted},
	} {
		converted := ToPlanRefusal(tc.err)

		var appErr *temporal.ApplicationError
		require.ErrorAs(t, converted, &appErr)
		assert.True(t, appErr.NonRetryable())
		assert.Equal(t, tc.expected.String(), appErr.Type())
		assert.False(t, IsRetryable(converted))
	}
}

func TestToPlanRefusalLeavesOtherErrorsAlone(t *testing.T) {
	t.Parallel()

	boom := errors.New("database unavailable")
	assert.Same(t, boom, ToPlanRefusal(boom))

	_, ok := NewPlanRefusalError(boom)
	assert.False(t, ok)
}

func TestNewPlanRefusalErrorCarriesTheParams(t *testing.T) {
	t.Parallel()

	refusal, ok := NewPlanRefusalError(errortypes.NewQuotaExceededError("customers.total", 8, 8, "free_demo"))
	require.True(t, ok)
	assert.Equal(t, "customers.total", refusal.Details["meter"])
	assert.Equal(t, "8", refusal.Details["limit"])
	assert.False(t, refusal.Retryable)
}
