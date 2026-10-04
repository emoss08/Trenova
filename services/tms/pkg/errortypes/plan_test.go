package errortypes_test

import (
	"fmt"
	"net/http"
	"testing"

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
	assert.Equal(t, http.StatusPaymentRequired, errortypes.HTTPStatus(wrapped))
	assert.Equal(t, http.StatusPaymentRequired, errortypes.HTTPStatusWithCode(errortypes.ErrQuotaExceeded))

	fields := err.LogFields()
	assert.Equal(t, "shipments.total", fields["quota_meter"])
	assert.Equal(t, int64(12), fields["quota_limit"])

	var errorable errortypes.Errorable = err
	assert.Equal(t, errortypes.ErrQuotaExceeded, errorable.GetCode())
}

func TestPlanRestrictionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason      string
		wantReason  string
		wantMessage string
	}{
		{
			reason:      "",
			wantReason:  errortypes.PlanRestrictionReasonPlan,
			wantMessage: "This feature is not available on your organization's plan",
		},
		{
			reason:      errortypes.PlanRestrictionReasonReadOnly,
			wantReason:  errortypes.PlanRestrictionReasonReadOnly,
			wantMessage: "This organization's trial has ended and it is now read-only",
		},
		{
			reason:      errortypes.PlanRestrictionReasonExpired,
			wantReason:  errortypes.PlanRestrictionReasonExpired,
			wantMessage: "This organization's trial has expired",
		},
		{
			reason:      errortypes.PlanRestrictionReasonSignupsPaused,
			wantReason:  errortypes.PlanRestrictionReasonSignupsPaused,
			wantMessage: "Signups are paused right now. You have been added to the wait list.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantReason, func(t *testing.T) {
			t.Parallel()

			err := errortypes.NewPlanRestrictionError("api_keys", tt.reason, "free_demo")
			assert.Equal(t, errortypes.ErrPlanRestricted, err.GetCode())
			assert.Equal(t, tt.wantReason, err.Reason)
			assert.Equal(t, tt.wantMessage, err.Error())
			assert.Equal(t, map[string]string{
				"capability": "api_keys",
				"reason":     tt.wantReason,
				"plan":       "free_demo",
			}, err.Params())
		})
	}

	err := fmt.Errorf("wrap: %w", errortypes.NewPlanRestrictionError("sso", "", "free_demo"))
	require.True(t, errortypes.IsPlanRestrictionError(err))
	assert.False(t, errortypes.IsQuotaExceededError(err))
	assert.False(t, errortypes.IsAuthorizationError(err))
	assert.Equal(t, http.StatusForbidden, errortypes.HTTPStatus(err))
	assert.Equal(t, http.StatusForbidden, errortypes.HTTPStatusWithCode(errortypes.ErrPlanRestricted))
}
