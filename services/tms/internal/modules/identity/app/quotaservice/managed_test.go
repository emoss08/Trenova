package quotaservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGate struct {
	state *managedusage.State
}

func (f *fakeGate) Decide(
	_ context.Context,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	return managedusage.Decide(f.state, meter, requested), nil
}

func (f *fakeGate) State(context.Context) (*managedusage.State, error) {
	return f.state, nil
}

func managedGuard(state *managedusage.State) platformplanports.QuotaGuard {
	return quotaservice.New(quotaservice.Params{
		Config: &managedconfig.Config{Enabled: true},
		Gate:   &fakeGate{state: state},
		Clock:  func() int64 { return 42 },
	})
}

func shipmentState(quantity int64, stopped bool) *managedusage.State {
	return &managedusage.State{
		WindowStart: 10,
		WindowEnd:   20,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Quantity:  quantity,
			Allowance: 100,
			Stopped:   stopped,
		}},
	}
}

func shipments(n int64) *platformplanports.QuotaRequest {
	return &platformplanports.QuotaRequest{Meter: platformcatalog.MeterShipmentsTotal, Quantity: n}
}

func TestNewIsManagedWhenEnabled(t *testing.T) {
	t.Parallel()

	guard := managedGuard(nil)
	assert.IsType(t, &quotaservice.ManagedQuotaGuard{}, guard)
	assert.True(t, quotaservice.Enforcing(guard))
}

func TestManagedGuardEnforcesTheAllowance(t *testing.T) {
	t.Parallel()

	guard := managedGuard(shipmentState(95, false))
	require.NoError(t, guard.Enforce(t.Context(), shipments(5)))

	err := guard.Enforce(t.Context(), shipments(6))
	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota)

	require.NoError(t, quotaservice.Preflight(t.Context(), guard, shipments(1)))
}

func TestManagedGuardStopsWhenTheControlPlaneSays(t *testing.T) {
	t.Parallel()

	guard := managedGuard(shipmentState(1, true))
	decision, err := guard.Check(t.Context(), shipments(1))
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, platformplan.PlanKeyManaged, decision.Plan)
	assert.Equal(t, int64(10), decision.WindowStart)
	require.Error(t, quotaservice.EnforceAll(t.Context(), guard, *shipments(1)))
}

func TestManagedGuardLeavesOtherMetersAlone(t *testing.T) {
	t.Parallel()

	guard := managedGuard(shipmentState(1, true))
	decision, err := guard.Check(t.Context(), &platformplanports.QuotaRequest{
		Meter:    platformcatalog.MeterCustomersTotal,
		Quantity: 1,
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.True(t, decision.Unlimited)
}

func TestManagedGuardRefusesABadRequest(t *testing.T) {
	t.Parallel()

	guard := managedGuard(nil)
	require.ErrorIs(t, guard.Enforce(t.Context(), nil), quotaservice.ErrRequestRequired)
	require.ErrorIs(t, guard.Enforce(t.Context(), shipments(-1)), quotaservice.ErrInvalidQuantity)
}

func TestManagedGuardUsage(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{}
	summary, err := managedGuard(nil).Usage(t.Context(), tenant)
	require.NoError(t, err)
	assert.True(t, summary.Unlimited)
	assert.Equal(t, platformplan.OriginManaged, summary.Origin)

	summary, err = managedGuard(shipmentState(30, false)).Usage(t.Context(), tenant)
	require.NoError(t, err)
	assert.False(t, summary.Unlimited)
	require.Len(t, summary.Meters, 1)
	assert.Equal(t, int64(70), summary.Meters[0].Remaining)
	assert.Equal(t, int64(42), summary.CheckedAt)
}
