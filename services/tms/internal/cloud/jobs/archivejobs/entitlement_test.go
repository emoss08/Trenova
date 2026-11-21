package archivejobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/stretchr/testify/assert"
)

func sharedServer() *managedconfig.Config {
	return &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared}
}

func TestTheNightlyArchiveServesOnlyActiveTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider(sharedServer()).GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  workflows(),
		Shapes: map[string]entitlementtest.Shape{
			NightlyScheduleID: entitlementtest.FansOut,
		},
	})
}

func TestTheNightlyArchiveIsScheduledOnlyOnASharedServer(t *testing.T) {
	t.Parallel()

	assert.Empty(t, NewScheduleProvider(&managedconfig.Config{Enabled: true}).GetSchedules())
	schedules := NewScheduleProvider(sharedServer()).GetSchedules()
	if assert.Len(t, schedules, 1) {
		assert.Equal(
			t,
			"0 7 * * *",
			schedules[0].Spec.Cron,
			"07:00 UTC is off-peak for US customers",
		)
	}
}
