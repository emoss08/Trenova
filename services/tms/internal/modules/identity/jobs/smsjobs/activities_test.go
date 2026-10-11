package smsjobs

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/jobs/smsjobs/smscontract"

	"github.com/emoss08/trenova/internal/infrastructure/sms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestSendSMSActivityRefusesWithoutCallingTwilioWhenThePlanRestrictsSMS(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	activities := &Activities{
		logger: zap.NewNop(),
		plans:  plantest.Restricting(t, platformplan.CapabilitySMS),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	_, err := env.ExecuteActivity(activities.SendSMSActivity, &smscontract.SendSMSPayload{
		PhoneNumber:    "+15555550100",
		Message:        "Your PTO was approved",
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, temporaltype.ErrorTypePlanRestricted.String(), appErr.Type())
}

func TestSendFailureClassifiesTwilioErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		nonRetryable bool
		delay        time.Duration
	}{
		{name: "not configured", err: sms.ErrNotConfigured, nonRetryable: true},
		{
			name:         "invalid number",
			err:          &sms.APIError{StatusCode: http.StatusBadRequest, Code: 21211},
			nonRetryable: true,
		},
		{
			name:         "bad credentials",
			err:          &sms.APIError{StatusCode: http.StatusUnauthorized, Code: 20003},
			nonRetryable: true,
		},
		{
			name:  "rate limited",
			err:   &sms.APIError{StatusCode: http.StatusTooManyRequests, RetryAfter: 20},
			delay: 20 * time.Second,
		},
		{name: "server error", err: &sms.APIError{StatusCode: http.StatusBadGateway}},
		{name: "network failure", err: errors.New("connection reset")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, sendFailure(tt.err), &appErr)
			require.Equal(t, tt.nonRetryable, appErr.NonRetryable())
			require.Equal(t, tt.delay, appErr.NextRetryDelay())
		})
	}
}
