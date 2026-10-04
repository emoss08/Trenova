package completionrouter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func spentGuard(t *testing.T, tenant pagination.TenantInfo) *mocks.MockQuotaGuard {
	t.Helper()

	guard := mocks.NewMockQuotaGuard(t)
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

	return guard
}

func TestEveryModelEntryRefusesOnceTheMonthsSpendIsGone(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := newTestService(t)
	svc.quota = spentGuard(t, tenant)

	_, err := svc.CompleteStructured(t.Context(), &serviceports.StructuredCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.CompleteChat(t.Context(), &serviceports.ChatCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.SubmitBackground(t.Context(), &serviceports.StructuredCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestAssertWithinSpendSkipsCallsWithoutATenant(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.quota = mocks.NewMockQuotaGuard(t)

	require.NoError(t, svc.assertWithinSpend(t.Context(), pagination.TenantInfo{}))
}
