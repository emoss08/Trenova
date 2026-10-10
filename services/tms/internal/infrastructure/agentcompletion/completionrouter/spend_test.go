package completionrouter

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func spentGuard(t *testing.T, tenant pagination.TenantInfo) *platformplanportsmock.MockQuotaGuard {
	t.Helper()

	guard := platformplanportsmock.NewMockQuotaGuard(t)
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

	return guard
}

func TestEveryModelEntryRefusesOnceTheMonthsSpendIsGone(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := newTestService(t)
	svc.quota = spentGuard(t, tenant)

	_, err := svc.CompleteStructured(t.Context(), &agentports.StructuredCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.CompleteChat(t.Context(), &agentports.ChatCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.SubmitBackground(t.Context(), &agentports.StructuredCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestAssertWithinSpendSkipsCallsWithoutATenant(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.quota = platformplanportsmock.NewMockQuotaGuard(t)

	require.NoError(t, svc.assertWithinSpend(t.Context(), pagination.TenantInfo{}))
}
