package aiauditjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  (&ScheduleProvider{projectorInterval: time.Minute}).GetSchedules(),
		Activities: []any{&Activities{}, &ExportActivities{}},
		Workflows:  convertWorkflows(RegisterWorkflows()),
		Shapes: map[string]entitlementtest.Shape{
			"ai-audit-projector":       entitlementtest.InOneActivity,
			"ai-audit-verify":          entitlementtest.FansOut,
			"ai-audit-retention-purge": entitlementtest.InOneActivity,
			"ai-audit-export-cleanup":  entitlementtest.InOneActivity,
		},
	})
}
