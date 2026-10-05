package planbilling

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLocalPlanUsageProvider_AllowsUnrelatedMeters(t *testing.T) {
	t.Parallel()

	provider := NewLocalPlanUsageProvider(LocalPlanUsageProviderParams{
		Quota: mocks.NewMockQuotaGuard(t),
	})

	result, err := provider.CheckLimit(t.Context(), &services.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterAPIRequests,
		Quantity: 1,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, usageservice.ReasonNotPlanMetered, result.Reason)
}

func TestLocalPlanUsageProvider_MapsDocumentChecksToThePlan(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().
		Check(mock.Anything, &services.QuotaRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			Meter:      platformcatalog.MeterDocumentUploads,
			Quantity:   1,
		}).
		Return(&services.QuotaDecision{
			Meter:   platformcatalog.MeterDocumentUploads,
			Plan:    platformplan.PlanKeyFreeDemo,
			Allowed: false,
			Limit:   25,
			Used:    25,
		}, nil).
		Once()

	provider := NewLocalPlanUsageProvider(LocalPlanUsageProviderParams{Quota: quota})

	err := usageservice.CheckDocumentUploadLimit(t.Context(), provider, usageservice.DocumentUploadUsageParams{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	})

	var exceeded *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &exceeded)
	assert.Equal(t, "documents.uploads", exceeded.Meter)
	assert.Equal(t, int64(25), exceeded.Limit)
	assert.Equal(t, int64(25), exceeded.Used)
	assert.Equal(t, "free_demo", exceeded.Plan)
}

func TestLocalPlanUsageProvider_ReportsWithinAndUnlimited(t *testing.T) {
	t.Parallel()

	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *services.QuotaRequest) bool {
			return req.Meter == platformcatalog.MeterDocumentStorageBytes
		})).
		Return(&services.QuotaDecision{Allowed: true, Limit: 100, Used: 10, Remaining: 90, Plan: platformplan.PlanKeyFreeDemo}, nil).
		Once()
	quota.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *services.QuotaRequest) bool {
			return req.Meter == platformcatalog.MeterDocumentFileBytes
		})).
		Return(&services.QuotaDecision{Allowed: true, Unlimited: true, Plan: platformplan.PlanKeyUnlimited}, nil).
		Once()

	provider := NewLocalPlanUsageProvider(LocalPlanUsageProviderParams{Quota: quota})

	within, err := provider.CheckLimit(t.Context(), &services.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentStorageBytes,
		Quantity: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, usageservice.ReasonWithinPlan, within.Reason)
	assert.Equal(t, int64(90), within.Remaining)
	assert.Equal(t, "free_demo", within.Plan)

	unlimited, err := provider.CheckLimit(t.Context(), &services.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentFileBytes,
		Quantity: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, usageservice.ReasonUnlimitedPlan, unlimited.Reason)

	recorded, err := provider.RecordUsage(t.Context(), &services.UsageRecordRequest{
		MeterKey: platformcatalog.MeterDocumentUploads,
		Quantity: 1,
	})
	require.NoError(t, err)
	assert.False(t, recorded.Recorded, "plan usage is counted from the records themselves")
}

func TestLocalPlanUsageProvider_PropagatesGuardErrors(t *testing.T) {
	t.Parallel()

	boom := errortypes.NewPlanRestrictionError("", errortypes.PlanRestrictionReasonReadOnly, "free_demo")
	quota := mocks.NewMockQuotaGuard(t)
	quota.EXPECT().Check(mock.Anything, mock.Anything).Return(nil, boom).Once()

	provider := NewLocalPlanUsageProvider(LocalPlanUsageProviderParams{Quota: quota})
	_, err := provider.CheckLimit(t.Context(), &services.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentUploads,
		Quantity: 1,
	})
	require.True(t, errors.Is(err, boom))
}
