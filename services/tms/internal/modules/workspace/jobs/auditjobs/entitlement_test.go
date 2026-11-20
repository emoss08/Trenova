package auditjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  Workflows,
		Shapes: map[string]entitlementtest.Shape{
			"audit-partition-check": entitlementtest.NoTenant,
			"audit-retention-purge": entitlementtest.NoTenant,
			"audit-seal":            entitlementtest.NoTenant,
			"audit-verify":          entitlementtest.NoTenant,
			"audit-export-cleanup":  entitlementtest.NoTenant,
		},
	})
}
