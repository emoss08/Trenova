package billingqueuejobs

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
			"billing-queue-recheck-sweep":         entitlementtest.InOneActivity,
			"billing-queue-nova-generation-sweep": entitlementtest.InOneActivity,
		},
	})
}
