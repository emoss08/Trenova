package fiscaljobs

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
			"fiscal-auto-close-periods":    entitlementtest.FansOut,
			"fiscal-auto-create-next-year": entitlementtest.FansOut,
			"fiscal-year-close-readiness":  entitlementtest.FansOut,
		},
	})
}
