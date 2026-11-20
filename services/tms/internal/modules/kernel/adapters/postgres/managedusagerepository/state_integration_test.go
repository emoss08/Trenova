//go:build integration

package managedusagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func shipments(quantity, ceiling int64) *managedusage.State {
	return &managedusage.State{
		WindowStart: 1_000,
		WindowEnd:   2_000,
		Overage:     true,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Quantity:  quantity,
			Allowance: 100,
			Ceiling:   ceiling,
		}},
		ReceivedAt: 1_500,
	}
}

func TestEachScopeKeepsItsOwnUsageState(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	org := managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.NoError(t, repo.Save(ctx, managedscope.Instance(), shipments(10, 0)))
	require.NoError(t, repo.Save(ctx, org, shipments(1_400, 1_500)))
	require.NoError(
		t,
		repo.Save(ctx, org, shipments(1_450, 1_500)),
		"a second save replaces the row",
	)

	instance, err := repo.Get(ctx, managedscope.Instance())
	require.NoError(t, err)
	assert.Equal(t, managedscope.InstanceKey, instance.ScopeKey)
	meter, ok := instance.Meter(platformcatalog.MeterShipmentsTotal)
	require.True(t, ok)
	assert.Equal(t, int64(10), meter.Quantity)
	assert.Zero(t, meter.Ceiling)

	scoped, err := repo.Get(ctx, org)
	require.NoError(t, err)
	assert.Equal(t, org.Key(), scoped.ScopeKey)
	meter, ok = scoped.Meter(platformcatalog.MeterShipmentsTotal)
	require.True(t, ok)
	assert.Equal(t, int64(1_450), meter.Quantity)
	assert.Equal(t, int64(1_500), meter.Ceiling, "the ceiling is kept with the meter")

	_, err = repo.Get(ctx, managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}))
	require.ErrorIs(t, err, managedusage.ErrNoState)
}
