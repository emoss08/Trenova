package usageservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckDocumentBytesLimit(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	require.NoError(t, CheckDocumentBytesLimit(t.Context(), nil, DocumentBytesUsageParams{Bytes: 10}))
	require.NoError(t, CheckDocumentBytesLimit(t.Context(), mocks.NewMockUsageProvider(t), DocumentBytesUsageParams{}))

	provider := mocks.NewMockUsageProvider(t)
	provider.EXPECT().
		CheckLimit(mock.Anything, mock.MatchedBy(func(req *services.UsageLimitCheckRequest) bool {
			return req.MeterKey == platformcatalog.MeterDocumentFileBytes && req.Quantity == 2048
		})).
		Return(&services.UsageLimitCheckResult{Allowed: true}, nil).
		Once()
	provider.EXPECT().
		CheckLimit(mock.Anything, mock.MatchedBy(func(req *services.UsageLimitCheckRequest) bool {
			return req.MeterKey == platformcatalog.MeterDocumentStorageBytes && req.Quantity == 2048
		})).
		Return(&services.UsageLimitCheckResult{
			MeterKey: platformcatalog.MeterDocumentStorageBytes,
			Allowed:  false,
			Reason:   ReasonQuotaExceeded,
			Limit:    4096,
			Used:     3000,
			Plan:     "free_demo",
		}, nil).
		Once()

	err := CheckDocumentBytesLimit(t.Context(), provider, DocumentBytesUsageParams{
		TenantInfo: tenant,
		Bytes:      2048,
	})
	var exceeded *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &exceeded)
	assert.Equal(t, "documents.storage_bytes", exceeded.Meter)
	assert.Equal(t, int64(3000), exceeded.Used)
}

func TestLimitError_FallsBackToAuthorization(t *testing.T) {
	t.Parallel()

	err := LimitError(&services.UsageLimitCheckResult{Reason: "plan_inactive"})
	assert.True(t, errortypes.IsAuthorizationError(err))
}
