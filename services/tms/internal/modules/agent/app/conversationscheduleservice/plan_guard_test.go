package conversationscheduleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestCreateRefusesScheduledRequestsThePlanRestricts(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := &Service{features: plantest.Without(platformcatalog.FeatureAgentAutomation)}

	result, err := svc.Create(t.Context(), CreateRequest{
		ThreadID: pulid.MustNew("thr_"),
		Text:     "every weekday at 9am summarize late loads",
		Actor: permissionports.RequestActor{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			UserID:         tenant.UserID,
		},
	})

	require.Nil(t, result)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
