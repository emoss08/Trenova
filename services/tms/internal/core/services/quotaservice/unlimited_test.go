package quotaservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnlimitedQuotaGuard(t *testing.T) {
	t.Parallel()

	guard := quotaservice.NewUnlimited()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	req := &services.QuotaRequest{TenantInfo: tenant, Meter: platformcatalog.MeterShipmentsTotal, Quantity: 9_999}

	require.NoError(t, guard.Enforce(t.Context(), req))
	require.ErrorIs(t, guard.Enforce(t.Context(), nil), quotaservice.ErrRequestRequired)

	decision, err := guard.Check(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.True(t, decision.Unlimited)

	summary, err := guard.Usage(t.Context(), tenant)
	require.NoError(t, err)
	assert.True(t, summary.Unlimited)
	assert.Empty(t, summary.Meters)
	assert.Equal(t, platformplan.PlanKeyUnlimited, summary.Plan)
}

func TestNewIsUnlimited(t *testing.T) {
	t.Parallel()

	assert.IsType(t, &quotaservice.UnlimitedQuotaGuard{}, quotaservice.New())
}
