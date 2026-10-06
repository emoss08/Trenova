package quotaservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func helperRequest(meter platformcatalog.MeterKey, quantity int64) *services.QuotaRequest {
	return &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Meter:      meter,
		Quantity:   quantity,
	}
}

func TestOrUnlimitedFallsBackToTheUnlimitedGuard(t *testing.T) {
	t.Parallel()

	assert.IsType(t, &quotaservice.UnlimitedQuotaGuard{}, quotaservice.OrUnlimited(nil))

	guard := mocks.NewMockQuotaGuard(t)
	assert.Same(t, guard, quotaservice.OrUnlimited(guard))
}

func TestEnforcingIsFalseOnlyForNoGuardOrTheUnlimitedGuard(t *testing.T) {
	t.Parallel()

	assert.False(t, quotaservice.Enforcing(nil))
	assert.False(t, quotaservice.Enforcing(quotaservice.NewUnlimited()))
	assert.True(t, quotaservice.Enforcing(mocks.NewMockQuotaGuard(t)))
}

func TestDecisionErrorTurnsARefusalIntoQuotaExceeded(t *testing.T) {
	t.Parallel()

	require.NoError(t, quotaservice.DecisionError(nil))
	require.NoError(t, quotaservice.DecisionError(&services.QuotaDecision{Allowed: true}))

	err := quotaservice.DecisionError(&services.QuotaDecision{
		Meter:   platformcatalog.MeterShipmentsTotal,
		Plan:    platformplan.PlanKeyFreeDemo,
		Allowed: false,
		Limit:   12,
		Used:    12,
	})
	require.True(t, errortypes.IsQuotaExceededError(err))

	params := err.(*errortypes.QuotaExceededError).Params()
	assert.Equal(t, string(platformcatalog.MeterShipmentsTotal), params["meter"])
	assert.Equal(t, "12", params["limit"])
	assert.Equal(t, "12", params["used"])
	assert.Equal(t, string(platformplan.PlanKeyFreeDemo), params["plan"])
}

func TestPreflightSkipsTheUnlimitedGuard(t *testing.T) {
	t.Parallel()

	require.NoError(t, quotaservice.Preflight(
		t.Context(),
		quotaservice.NewUnlimited(),
		helperRequest(platformcatalog.MeterShipmentsTotal, 1),
	))
	require.NoError(t, quotaservice.Preflight(
		t.Context(),
		nil,
		helperRequest(platformcatalog.MeterShipmentsTotal, 1),
	))
}

func TestPreflightRefusesWhenTheCheckDisallows(t *testing.T) {
	t.Parallel()

	guard := mocks.NewMockQuotaGuard(t)
	req := helperRequest(platformcatalog.MeterCustomersTotal, 1)
	guard.EXPECT().Check(mock.Anything, req).Return(&services.QuotaDecision{
		Meter:   req.Meter,
		Plan:    platformplan.PlanKeyFreeDemo,
		Allowed: false,
		Limit:   8,
		Used:    8,
	}, nil)

	err := quotaservice.Preflight(t.Context(), guard, req)
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestPreflightPassesAnAllowedCheckAndReturnsCheckErrors(t *testing.T) {
	t.Parallel()

	guard := mocks.NewMockQuotaGuard(t)
	allowed := helperRequest(platformcatalog.MeterLocationsTotal, 1)
	failing := helperRequest(platformcatalog.MeterWorkersTotal, 1)
	boom := errors.New("count failed")
	guard.EXPECT().Check(mock.Anything, allowed).Return(&services.QuotaDecision{Allowed: true}, nil)
	guard.EXPECT().Check(mock.Anything, failing).Return(nil, boom)

	require.NoError(t, quotaservice.Preflight(t.Context(), guard, allowed))
	require.ErrorIs(t, quotaservice.Preflight(t.Context(), guard, failing), boom)
}

func TestEnforceAllStopsAtTheFirstRefusal(t *testing.T) {
	t.Parallel()

	guard := mocks.NewMockQuotaGuard(t)
	first := *helperRequest(platformcatalog.MeterDocumentFileBytes, 10)
	second := *helperRequest(platformcatalog.MeterDocumentUploads, 1)
	third := *helperRequest(platformcatalog.MeterDocumentStorageBytes, 10)
	refusal := errortypes.NewQuotaExceededError(string(second.Meter), 25, 25, "free_demo")

	guard.EXPECT().Enforce(mock.Anything, &first).Return(nil).Once()
	guard.EXPECT().Enforce(mock.Anything, &second).Return(refusal).Once()

	err := quotaservice.EnforceAll(t.Context(), guard, first, second, third)
	require.ErrorIs(t, err, refusal)
}

func TestEnforceAllSkipsTheUnlimitedGuard(t *testing.T) {
	t.Parallel()

	require.NoError(t, quotaservice.EnforceAll(
		t.Context(),
		quotaservice.NewUnlimited(),
		*helperRequest(platformcatalog.MeterShipmentsTotal, 5),
	))
	require.NoError(t, quotaservice.EnforceAll(t.Context(), nil))
}
