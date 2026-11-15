package errortypes_test

import (
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuotaExceededError(t *testing.T) {
	t.Parallel()

	err := errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo")

	assert.Equal(t, errortypes.ErrQuotaExceeded, err.GetCode())
	assert.Equal(t, "This organization has reached the limit of its plan", err.Error())
	assert.Equal(t, map[string]string{
		"meter": "shipments.total",
		"limit": "12",
		"used":  "12",
		"plan":  "free_demo",
	}, err.Params())

	wrapped := fmt.Errorf("create shipment: %w", err)
	assert.True(t, errortypes.IsQuotaExceededError(wrapped))
	assert.False(t, errortypes.IsPlanRestrictionError(wrapped))

	fields := err.LogFields()
	assert.Equal(t, "shipments.total", fields["quota_meter"])
	assert.Equal(t, int64(12), fields["quota_limit"])

	var errorable errortypes.Errorable = err
	assert.Equal(t, errortypes.ErrQuotaExceeded, errorable.GetCode())
}

func TestPlanRestrictionErrorForEachReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason      string
		wantMessage string
	}{
		{
			reason:      errortypes.PlanRestrictionReasonReadOnly,
			wantMessage: "This organization's trial has ended and it is now read-only",
		},
		{
			reason:      errortypes.PlanRestrictionReasonExpired,
			wantMessage: "This organization's trial has expired",
		},
		{
			reason:      errortypes.PlanRestrictionReasonSignupsPaused,
			wantMessage: "Signups are paused right now. You have been added to the wait list.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			t.Parallel()

			err := errortypes.NewPlanRestrictionError(tt.reason, "free_demo")
			assert.Equal(t, errortypes.ErrPlanRestricted, err.GetCode())
			assert.Equal(t, tt.reason, err.Reason)
			assert.Empty(t, err.Feature)
			assert.Empty(t, err.AddOn)
			assert.Equal(t, tt.wantMessage, err.Error())

			message, args := err.LocalizedMessage()
			assert.Equal(t, tt.wantMessage, message)
			assert.Empty(t, args)

			assert.Equal(t, map[string]string{
				"feature": "",
				"addOn":   "",
				"reason":  tt.reason,
				"plan":    "free_demo",
			}, err.Params())

			body, marshalErr := sonic.Marshal(err)
			require.NoError(t, marshalErr)
			assert.JSONEq(t, `{
				"code": "PLAN_RESTRICTED",
				"message": "`+tt.wantMessage+`",
				"feature": "",
				"addOn": "",
				"reason": "`+tt.reason+`",
				"plan": "free_demo"
			}`, string(body))

			fields := err.LogFields()
			assert.Equal(t, "", fields["plan_feature"])
			assert.Equal(t, "", fields["plan_add_on"])
			assert.Equal(t, tt.reason, fields["plan_reason"])
			assert.Equal(t, "free_demo", fields["plan_key"])
		})
	}
}

func TestFeatureRestrictionErrorNamesTheAddOnThatSellsIt(t *testing.T) {
	t.Parallel()

	err := errortypes.NewFeatureRestrictionError("ai.desk", "Trenova AI")

	assert.Equal(t, errortypes.ErrPlanRestricted, err.GetCode())
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, err.Reason)
	assert.Equal(t, "ai.desk", err.Feature)
	assert.Equal(t, "Trenova AI", err.AddOn)
	assert.Empty(t, err.Plan)
	assert.Equal(
		t,
		"This feature is part of the Trenova AI add-on, which is not on your plan",
		err.Error(),
	)

	message, args := err.LocalizedMessage()
	assert.Equal(t, "This feature is part of the {0} add-on, which is not on your plan", message)
	assert.Equal(t, []any{"Trenova AI"}, args)

	assert.Equal(t, map[string]string{
		"feature": "ai.desk",
		"addOn":   "Trenova AI",
		"reason":  errortypes.PlanRestrictionReasonPlan,
		"plan":    "",
	}, err.Params())

	body, marshalErr := sonic.Marshal(err)
	require.NoError(t, marshalErr)
	assert.JSONEq(t, `{
		"code": "PLAN_RESTRICTED",
		"message": "This feature is part of the {0} add-on, which is not on your plan",
		"feature": "ai.desk",
		"addOn": "Trenova AI",
		"reason": "plan_restricted",
		"plan": ""
	}`, string(body))

	fields := err.LogFields()
	assert.Equal(t, "ai.desk", fields["plan_feature"])
	assert.Equal(t, "Trenova AI", fields["plan_add_on"])
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, fields["plan_reason"])
}

func TestFeatureRestrictionErrorForABaseFeatureNamesNoAddOn(t *testing.T) {
	t.Parallel()

	err := errortypes.NewFeatureRestrictionError("platform.api_keys", "")

	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, err.Reason)
	assert.Equal(t, "platform.api_keys", err.Feature)
	assert.Empty(t, err.AddOn)
	assert.Equal(t, "This feature is not available on your organization's plan", err.Error())

	message, args := err.LocalizedMessage()
	assert.Equal(t, "This feature is not available on your organization's plan", message)
	assert.Empty(t, args)

	assert.Equal(t, map[string]string{
		"feature": "platform.api_keys",
		"addOn":   "",
		"reason":  errortypes.PlanRestrictionReasonPlan,
		"plan":    "",
	}, err.Params())

	body, marshalErr := sonic.Marshal(err)
	require.NoError(t, marshalErr)
	assert.JSONEq(t, `{
		"code": "PLAN_RESTRICTED",
		"message": "This feature is not available on your organization's plan",
		"feature": "platform.api_keys",
		"addOn": "",
		"reason": "plan_restricted",
		"plan": ""
	}`, string(body))
}

func TestPlanRestrictionErrorIsRecognisedThroughWrapping(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf(
		"wrap: %w",
		errortypes.NewFeatureRestrictionError("platform.identity", "Security & Identity"),
	)
	require.True(t, errortypes.IsPlanRestrictionError(err))
	assert.False(t, errortypes.IsQuotaExceededError(err))
	assert.False(t, errortypes.IsAuthorizationError(err))
}
