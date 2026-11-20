package edijobs

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
			"edi-inbound-poll":        entitlementtest.FansOut,
			"edi-flow-timers":         entitlementtest.InOneActivity,
			"edi-raw-retention-purge": entitlementtest.FansOut,
		},
	})
}
