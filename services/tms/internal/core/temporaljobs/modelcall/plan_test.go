package modelcall

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestClassifyMakesAPlanLimitNonRetryableAndKeepsItWhole(t *testing.T) {
	t.Parallel()

	quota := errortypes.NewQuotaExceededError("ai.spend_cents", 150, 152, "free_demo")
	classified := Classify(quota)

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, classified, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeModelRejected, appErr.Type())
	assert.False(t, Transient(quota))

	failure := FailureOf(classified)
	require.NotNil(t, failure.Plan)
	assert.Equal(t, "ai.spend_cents", failure.Plan.Meter)

	restored, ok := errors.AsType[*errortypes.QuotaExceededError](failure.Err())
	require.True(t, ok)
	assert.Equal(t, int64(150), restored.Limit)
	assert.Equal(t, int64(152), restored.Used)
	assert.Equal(t, "free_demo", restored.Plan)
}

func TestClassifyCarriesAPlanRestrictionThrough(t *testing.T) {
	t.Parallel()

	restriction := errortypes.NewPlanRestrictionError("document_intelligence", "plan_restricted", "free_demo")
	failure := FailureOf(Classify(restriction))

	require.NotNil(t, failure.Plan)
	restored, ok := errors.AsType[*errortypes.PlanRestrictionError](failure.Err())
	require.True(t, ok)
	assert.Equal(t, "document_intelligence", restored.Capability)
}
