package conversationscheduleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestCreateRefusesScheduledRequestsThePlanRestricts(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}

	result, err := svc.Create(t.Context(), CreateRequest{
		ThreadID: pulid.MustNew("thr_"),
		Text:     "every weekday at 9am summarize late loads",
		Actor: serviceports.RequestActor{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			UserID:         tenant.UserID,
		},
	})

	require.Nil(t, result)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
