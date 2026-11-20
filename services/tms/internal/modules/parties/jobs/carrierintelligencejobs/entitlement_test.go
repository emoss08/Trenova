package carrierintelligencejobs

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
			"carrier-intelligence-sweep":       entitlementtest.FansOut,
			"carrier-intelligence-reconcile":   entitlementtest.FansOut,
			"carrier-intelligence-digest":      entitlementtest.FansOut,
			"carrier-intelligence-maintenance": entitlementtest.FansOut,
		},
	})
}
