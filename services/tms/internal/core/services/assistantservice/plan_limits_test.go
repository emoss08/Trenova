package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssertWithinPlanRefusesOnceTheMonthsSpendIsGone(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := mocks.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, &serviceports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}).Return(&serviceports.QuotaDecision{Allowed: true}, nil)
	guard.EXPECT().Check(mock.Anything, &serviceports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	}).Return(&serviceports.QuotaDecision{
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
	guard := mocks.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, mock.Anything).Return(&serviceports.QuotaDecision{
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
