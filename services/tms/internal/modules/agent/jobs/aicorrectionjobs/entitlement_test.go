package aicorrectionjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  (&ScheduleProvider{extraction: true}).GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  Workflows,
		Shapes: map[string]entitlementtest.Shape{
			"ai-correction-retention":   entitlementtest.FansOut,
			"extraction-accuracy-drift": entitlementtest.FansOut,
		},
	})
}
