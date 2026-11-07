package usageservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGate struct {
	state *managedusage.State
	err   error
}

func (f *fakeGate) Decide(
	_ context.Context,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	if f.err != nil {
		return managedusage.Decision{}, f.err
	}
	return managedusage.Decide(f.state, meter, requested), nil
}

func (f *fakeGate) State(context.Context) (*managedusage.State, error) {
	return f.state, f.err
}

func TestManagedUsageProviderStopsAtTheAllowance(t *testing.T) {
	t.Parallel()

	gate := &fakeGate{state: &managedusage.State{
		WindowStart: 1,
		WindowEnd:   2,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterDocumentStorageBytes,
			Unit:      managedusage.UnitBytes,
			Quantity:  900,
			Allowance: 1000,
			HardCap:   true,
		}},
	}}
	p := usageservice.NewManagedUsageProvider(gate)

	res, err := p.CheckLimit(t.Context(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentStorageBytes,
		Quantity: 100,
	})
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, usageservice.ReasonWithinPlan, res.Reason)
	assert.Equal(t, int64(100), res.Remaining)

	res, err = p.CheckLimit(t.Context(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentStorageBytes,
		Quantity: 101,
	})
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, usageservice.LimitError(res), &quota)

	res, err = p.CheckLimit(t.Context(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentUploads,
		Quantity: 1,
	})
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, usageservice.ReasonNotPlanMetered, res.Reason)
}

func TestManagedUsageProviderReturnsAGateFailure(t *testing.T) {
	t.Parallel()

	p := usageservice.NewManagedUsageProvider(&fakeGate{err: errors.New("down")})
	_, err := p.CheckLimit(t.Context(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentStorageBytes,
		Quantity: 1,
	})
	require.Error(t, err)
}
