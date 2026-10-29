package completionrouter

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
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
	svc.Quota = spentGuard(t, tenant)

	_, err := svc.CompleteStructured(
		t.Context(),
		&llmports.StructuredCompletionRequest{TenantInfo: tenant},
	)
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.CompleteChat(t.Context(), &llmports.ChatCompletionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsQuotaExceededError(err))

	_, err = svc.SubmitBackground(
		t.Context(),
		&llmports.StructuredCompletionRequest{TenantInfo: tenant},
	)
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestAssertWithinSpendSkipsCallsWithoutATenant(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.Quota = platformplanportsmock.NewMockQuotaGuard(t)

	require.NoError(t, svc.AssertWithinSpend(t.Context(), pagination.TenantInfo{}))
}
