package telematicsjobs

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
			"samsara-telematics-poll":    entitlementtest.FansOut,
			"telematics-retention-sweep": entitlementtest.InOneActivity,
			"samsara-telematics-sweep":   entitlementtest.FansOut,
		},
	})
}
