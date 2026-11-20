package managedusageservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func organization() managedscope.Scope {
	return managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func shipmentState(quantity, allowance, ceiling int64, overage bool) *managedusage.State {
	return &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Overage:     overage,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Quantity:  quantity,
			Allowance: allowance,
			Ceiling:   ceiling,
		}},
	}
}

func TestEachOrganizationIsDecidedOnItsOwnAllowance(t *testing.T) {
	t.Parallel()

	full, room := organization(), organization()
	svc := newServiceIn(&fakeRepo{}, true, nil, nil)
	_, err := svc.SetState(t.Context(), full, shipmentState(100, 100, 0, false))
	require.NoError(t, err)
	_, err = svc.SetState(t.Context(), room, shipmentState(10, 100, 0, false))
	require.NoError(t, err)

	d, err := svc.Decide(t.Context(), full, platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonAllowance, d.Reason)

	d, err = svc.Decide(t.Context(), room, platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed)

	d, err = svc.Decide(t.Context(), organization(), platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.False(t, d.Managed, "an organization with no usage state yet is not limited")
}

func TestASetStateDropsOnlyItsOwnScopeFromTheCache(t *testing.T) {
	t.Parallel()

	changed, untouched := organization(), organization()
	repo := &fakeRepo{}
	svc := newServiceIn(repo, true, nil, nil)
	for _, scope := range []managedscope.Scope{changed, untouched} {
		_, err := svc.State(t.Context(), scope)
		require.NoError(t, err)
	}

	_, err := svc.SetState(t.Context(), changed, shipmentState(1, 100, 0, false))
	require.NoError(t, err)
	for _, scope := range []managedscope.Scope{changed, untouched} {
		_, err = svc.State(t.Context(), scope)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, repo.getsBy[changed.Key()], "read, read by SetState, read again")
	assert.Equal(t, 1, repo.getsBy[untouched.Key()], "still cached")
}

func TestPastTheCeilingAnActionIsRefusedEvenWithOverage(t *testing.T) {
	t.Parallel()

	org := organization()
	svc := newServiceIn(&fakeRepo{}, true, nil, nil)
	_, err := svc.SetState(t.Context(), org, shipmentState(1_499, 100, 1_500, true))
	require.NoError(t, err)

	d, err := svc.Decide(t.Context(), org, platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed, "overage carries it up to the ceiling")

	d, err = svc.Decide(t.Context(), org, platformcatalog.MeterShipmentsTotal, 2)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonCeiling, d.Reason)
	assert.Equal(t, int64(1_500), d.Ceiling)
}

func TestALevelMeterIsReadForTheScopeAsked(t *testing.T) {
	t.Parallel()

	org := organization()
	storage := &fakeLevel{
		meter:  platformcatalog.MeterDocumentStorageBytes,
		levels: map[string]int64{org.Key(): 900},
	}
	repo := &fakeRepo{
		scoped: map[string]*managedusage.State{org.Key(): storageState(0, 1_000, false)},
	}
	svc := newServiceIn(repo, true, nil, []managedusageports.LevelReader{storage})

	d, err := svc.Decide(t.Context(), org, platformcatalog.MeterDocumentStorageBytes, 200)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Equal(t, int64(900), d.Used)
	require.Len(t, storage.asked, 1)
	assert.Equal(t, org, storage.asked[0])
}

func TestTheDedicatedReadingsAreForTheInstance(t *testing.T) {
	t.Parallel()

	current := now - now%hour
	shipments := &fakeCounter{meter: platformcatalog.MeterShipmentsTotal}
	storage := &fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes}
	svc := newService(
		&fakeRepo{},
		[]managedusageports.HourlyCounter{shipments},
		[]managedusageports.LevelReader{storage},
	)

	_, err := svc.Readings(t.Context(), instance, current)
	require.NoError(t, err)
	assert.True(t, shipments.req.Scope.IsInstance())
	require.Len(t, storage.asked, 1)
	assert.True(t, storage.asked[0].IsInstance())
}

func TestAScopeFromTheOtherTenancyIsRefused(t *testing.T) {
	t.Parallel()

	dedicated := newService(&fakeRepo{}, nil, nil)
	_, err := dedicated.Decide(t.Context(), organization(), platformcatalog.MeterShipmentsTotal, 1)
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)

	shared := newServiceIn(&fakeRepo{}, true, nil, nil)
	_, err = shared.SetState(t.Context(), instance, shipmentState(0, 1, 0, false))
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)
}
