package reportjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportcatalogjobs"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}, &reportcatalogjobs.Activities{}},
		Workflows:  Workflows,
		Shapes: map[string]entitlementtest.Shape{
			"report-schedule-dispatch":     entitlementtest.InOneActivity,
			"report-artifact-cleanup":      entitlementtest.InOneActivity,
			"report-zombie-reconciliation": entitlementtest.InOneActivity,
			"report-catalog-check":         entitlementtest.InOneActivity,
		},
	})
}
