package storagequotaservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/storagequotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gate struct {
	state *managedusage.State
}

func (g gate) Decide(
	_ context.Context,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	return managedusage.Decide(g.state, meter, requested), nil
}

func (g gate) State(context.Context) (*managedusage.State, error) {
	return g.state, nil
}

func statusAt(t *testing.T, used int64, overage bool) *storagequotaservice.Status {
	t.Helper()
	svc := storagequotaservice.New(storagequotaservice.Params{
		Quota: quotaservice.NewManaged(gate{state: &managedusage.State{
			WindowStart: 1,
			WindowEnd:   2,
			Overage:     overage,
			Meters: []managedusage.Meter{{
				Meter:     platformcatalog.MeterDocumentStorageBytes,
				Unit:      managedusage.UnitBytes,
				Quantity:  used,
				Allowance: 1000,
			}},
		}}, func() int64 { return 0 }),
	})
	status, err := svc.Status(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	return status
}

func TestTheStorageStatusWarnsAtFourFifths(t *testing.T) {
	t.Parallel()

	assert.False(t, statusAt(t, 799, false).Warning)
	status := statusAt(t, 800, false)
	assert.True(t, status.Managed)
	assert.True(t, status.Warning)
	assert.False(t, status.Stopped)
	assert.Equal(t, int64(800), status.Used)
	assert.Equal(t, int64(1000), status.Included)
}

func TestTheStorageStatusIsStoppedAtTheLimitOnlyWithOverageOff(t *testing.T) {
	t.Parallel()

	assert.True(t, statusAt(t, 1000, false).Stopped)
	status := statusAt(t, 1000, true)
	assert.False(t, status.Stopped)
	assert.True(t, status.Overage)
}

func TestASelfHostedInstanceHasNoStorageStatus(t *testing.T) {
	t.Parallel()

	svc := storagequotaservice.New(storagequotaservice.Params{})
	status, err := svc.Status(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.False(t, status.Managed)
}
