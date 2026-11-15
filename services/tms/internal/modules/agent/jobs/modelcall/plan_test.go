package modelcall

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestClassifyMakesAPlanLimitNonRetryableAndKeepsItWhole(t *testing.T) {
	t.Parallel()

	quota := errortypes.NewQuotaExceededError("ai.assistant_messages", 150, 152, "free_demo")
	classified := Classify(quota)

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, classified, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeModelRejected, appErr.Type())
	assert.False(t, Transient(quota))

	failure := modelcallcontract.FailureOf(classified)
	require.NotNil(t, failure.Plan)
	assert.Equal(t, "ai.assistant_messages", failure.Plan.Meter)

	restored, ok := errors.AsType[*errortypes.QuotaExceededError](failure.Err())
	require.True(t, ok)
	assert.Equal(t, int64(150), restored.Limit)
	assert.Equal(t, int64(152), restored.Used)
	assert.Equal(t, "free_demo", restored.Plan)
}

func TestClassifyCarriesAPlanRestrictionThrough(t *testing.T) {
	t.Parallel()

	restriction := errortypes.NewFeatureRestrictionError(
		"document_intelligence.ai",
		"Document Intelligence",
	)
	failure := modelcallcontract.FailureOf(Classify(restriction))

	require.NotNil(t, failure.Plan)
	assert.Equal(t, "document_intelligence.ai", failure.Plan.Feature)
	assert.Equal(t, "Document Intelligence", failure.Plan.AddOn)
	assert.Equal(
		t,
		"This feature is part of the Document Intelligence add-on, which is not on your plan",
		failure.Plan.Message,
	)
	restored, ok := errors.AsType[*errortypes.PlanRestrictionError](failure.Err())
	require.True(t, ok)
	assert.Equal(t, "document_intelligence.ai", restored.Feature)
	assert.Equal(t, "Document Intelligence", restored.AddOn)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restored.Reason)
	assert.Equal(t, restriction.Error(), restored.Error())
}

func TestClassifyCarriesASubscriptionRestrictionThrough(t *testing.T) {
	t.Parallel()

	restriction := errortypes.NewPlanRestrictionError(
		errortypes.PlanRestrictionReasonReadOnly,
		"free_demo",
	)
	failure := modelcallcontract.FailureOf(Classify(restriction))

	require.NotNil(t, failure.Plan)
	restored, ok := errors.AsType[*errortypes.PlanRestrictionError](failure.Err())
	require.True(t, ok)
	assert.Empty(t, restored.Feature)
	assert.Equal(t, errortypes.PlanRestrictionReasonReadOnly, restored.Reason)
	assert.Equal(t, "free_demo", restored.Plan)
	assert.Equal(t, "This organization's trial has ended and it is now read-only", restored.Error())
}
