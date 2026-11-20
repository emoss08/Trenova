package accountingsyncjobs

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
			"accounting-connection-check":      entitlementtest.InOneActivity,
			"accounting-webhook-subscriptions": entitlementtest.InOneActivity,
			"accounting-reference-refresh":     entitlementtest.FansOut,
			"accounting-changes-read":          entitlementtest.InOneActivity,
			"accounting-drift-reconcile":       entitlementtest.InOneActivity,
			"accounting-drift-weekly":          entitlementtest.InOneActivity,
			"accounting-sync-dispatch":         entitlementtest.InOneActivity,
			"accounting-sync-safety-net":       entitlementtest.InOneActivity,
			"accounting-sync-retention":        entitlementtest.InOneActivity,
		},
	})
}
