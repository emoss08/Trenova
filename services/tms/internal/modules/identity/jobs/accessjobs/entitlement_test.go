package accessjobs

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
			"orphaned-organization-access-sweep": entitlementtest.InOneActivity,
			"support-access-expiry-sweep":        entitlementtest.InOneActivity,
			"support-log-request-timeout-sweep":  entitlementtest.InOneActivity,
		},
	})
}
