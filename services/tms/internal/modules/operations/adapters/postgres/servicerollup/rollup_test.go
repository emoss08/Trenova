package servicerollup_test

import (
	"testing"

	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicerollup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrationsDir = "../../../../../../schema/migrations"

func TestEveryTableHoldsTheColumnsItsReadsSum(t *testing.T) {
	t.Parallel()

	for _, table := range []*rollupread.Table{
		servicerollup.DailyTable, servicerollup.CustomerDailyTable,
	} {
		for _, name := range []string{
			rollupread.Bucket, servicerollup.Delivered, servicerollup.OnTime, servicerollup.Late,
		} {
			assert.True(t, table.Has(name), "%s has no %s", table.Ref.Table.Name, name)
		}
	}
	assert.True(t, servicerollup.CustomerDailyTable.Has(servicerollup.Customer))
	assert.False(t, servicerollup.DailyTable.Has(servicerollup.Customer))
}

func TestTheRollupsAreBucketedInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	for _, r := range servicerollup.All() {
		assert.True(t, r.TenantZoned(), r.Name())
	}
}

func TestEveryRollupHasItsCurrentMigration(t *testing.T) {
	t.Parallel()

	require.NoError(t, rollup.CheckMigrations(migrationsDir, servicerollup.All()...))
}
