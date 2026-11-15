package turnlimits

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssertWithinPlanAsksOnlyTheMessageMeter(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}).Return(&platformplanports.QuotaDecision{Allowed: true}, nil).Once()

	require.NoError(t, (&Limits{Quota: guard}).AssertWithinPlan(t.Context(), tenant))
}

func TestAssertMayAskAsksOnlyTheMessageMeter(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   1,
	}).Return(&platformplanports.QuotaDecision{Allowed: true}, nil).Once()

	require.NoError(t, (&Limits{Quota: guard}).AssertMayAsk(t.Context(), pulid.Nil, tenant))
}

func TestAssertWithinPlanRefusesATurnPastTheMessageLimit(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, mock.Anything).Return(&platformplanports.QuotaDecision{
		Meter:   platformcatalog.MeterAIAssistantMessages,
		Plan:    platformplan.PlanKeyFreeDemo,
		Limit:   25,
		Used:    26,
		Allowed: false,
	}, nil).Once()

	err := (&Limits{Quota: guard}).AssertWithinPlan(t.Context(), tenant)
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestAssertWithinPlanPassesWithoutAGuard(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Limits{}).AssertWithinPlan(t.Context(), plantest.Tenant()))
}
