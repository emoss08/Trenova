package agentjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/evalcasejobs"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}, &evalcasejobs.Activities{}},
		Workflows:  nil,
		Shapes: map[string]entitlementtest.Shape{
			"agent-definition-schedules": entitlementtest.InOneActivity,
			"agent-proposal-expiry":      entitlementtest.InOneActivity,
			"assistant-retention":        entitlementtest.InOneActivity,
			"agent-eval-case-capture":    entitlementtest.InOneActivity,
			"agent-eval-case-retention":  entitlementtest.InOneActivity,
		},
	})
}
