package smsjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
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

	_, err := env.ExecuteActivity(activities.SendSMSActivity, &SendSMSPayload{
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
