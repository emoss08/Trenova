package fuelrollupjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  workflows(),
		Shapes: map[string]entitlementtest.Shape{
			RecentScheduleID:   entitlementtest.NoTenant,
			TrailingScheduleID: entitlementtest.NoTenant,
		},
	})
}
