package paymentreminderjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
)

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  nil,
		Shapes: map[string]entitlementtest.Shape{
			"payment-reminder-sweep": entitlementtest.InOneActivity,
		},
	})
}
