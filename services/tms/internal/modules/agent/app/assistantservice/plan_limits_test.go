package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssertWithinPlanRefusesOnceTheMonthsSpendIsGone(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}).Return(&platformplanports.QuotaDecision{Allowed: true}, nil)
	guard.EXPECT().Check(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	}).Return(&platformplanports.QuotaDecision{
		Meter:   platformcatalog.MeterAISpendCents,
		Plan:    platformplan.PlanKeyFreeDemo,
		Limit:   150,
		Used:    150,
		Allowed: false,
	}, nil)

	err := (&Service{quota: guard}).assertWithinPlan(t.Context(), tenant)
	require.True(t, errortypes.IsQuotaExceededError(err))
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

	err := (&Service{quota: guard}).assertWithinPlan(t.Context(), tenant)
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestAssertWithinPlanPassesWithoutAGuard(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Service{}).assertWithinPlan(t.Context(), plantest.Tenant()))
}
