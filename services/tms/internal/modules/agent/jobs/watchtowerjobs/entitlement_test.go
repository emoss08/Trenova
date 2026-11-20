package watchtowerjobs

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
			"watchtower-reconcile":       entitlementtest.InOneActivity,
			"no-check-call-sweep":        entitlementtest.InOneActivity,
			"exception-attention":        entitlementtest.InOneActivity,
			"shipment-exception-sweep":   entitlementtest.InOneActivity,
			"factoring-watchtower-sweep": entitlementtest.InOneActivity,
			"decision-sweep":             entitlementtest.InOneActivity,
			"watchtower-retention":       entitlementtest.InOneActivity,
		},
	})
}
