package schedulingrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestAssignedDaysQuery_CountsLiveAssignmentsPerRosterDay(t *testing.T) {
	t.Parallel()

	repo := &repository{assignments: crossref.RotaAssignments()}
	keeltest.Snapshot(t, repo.assignedDaysQuery(&workerports.RotaQuery{
		WeekStart:   1_700_000_000,
		WeekEnd:     1_700_604_800,
		FleetCodeID: pulid.ID("fc_1"),
		ManagerIDs:  []pulid.ID{"usr_1"},
	}))
}
