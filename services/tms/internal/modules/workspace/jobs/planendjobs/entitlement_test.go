package planendjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  (&ScheduleProvider{managed: true}).GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  workflows(),
		Shapes: map[string]entitlementtest.Shape{
			"managed-plan-end-reminder": entitlementtest.NoTenant,
		},
	})
}
