package usageservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
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
	_ managedscope.Scope,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	if f.err != nil {
		return managedusage.Decision{}, f.err
	}
	return managedusage.Decide(f.state, meter, requested), nil
}

func (f *fakeGate) State(context.Context, managedscope.Scope) (*managedusage.State, error) {
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
	p := usageservice.NewManagedUsageProvider(gate, managedscope.NewResolver(false))

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
	require.ErrorAs(t, usageservice.LimitError(res, 101), &quota)

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

	p := usageservice.NewManagedUsageProvider(
		&fakeGate{err: errors.New("down")},
		managedscope.NewResolver(false),
	)
	_, err := p.CheckLimit(t.Context(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterDocumentStorageBytes,
		Quantity: 1,
	})
	require.Error(t, err)
}

func storageGate(overage bool) *fakeGate {
	return &fakeGate{state: &managedusage.State{
		WindowStart: 1,
		WindowEnd:   2,
		Overage:     overage,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterDocumentStorageBytes,
			Unit:      managedusage.UnitBytes,
			Quantity:  1000,
			Allowance: 1000,
		}},
	}}
}

func TestAnUploadAtTheStorageLimitIsRefusedWithTheStorageWordingWhenOverageIsOff(t *testing.T) {
	t.Parallel()

	err := usageservice.CheckDocumentBytesLimit(
		t.Context(),
		usageservice.NewManagedUsageProvider(storageGate(false), managedscope.NewResolver(false)),
		usageservice.DocumentBytesUsageParams{Bytes: 1},
	)

	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota)
	assert.Equal(t, string(platformcatalog.MeterDocumentStorageBytes), quota.Meter)
	assert.Contains(t, quota.Message, "Turn on overage from your Trenova account")
	assert.Contains(t, quota.Message, "delete documents")
}

func TestAnUploadAtTheStorageLimitIsAllowedWhenOverageIsOn(t *testing.T) {
	t.Parallel()

	require.NoError(t, usageservice.CheckDocumentBytesLimit(
		t.Context(),
		usageservice.NewManagedUsageProvider(storageGate(true), managedscope.NewResolver(false)),
		usageservice.DocumentBytesUsageParams{Bytes: 1},
	))
}
