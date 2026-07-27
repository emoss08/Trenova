package dispatchconsoleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNow = int64(1_700_000_000)

func TestUrgencyFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		windowStart int64
		expect      UrgencyBucket
	}{
		{"already open and uncovered", testNow - 3600, UrgencyLate},
		{"within four hours", testNow + 2*secondsPerHour, UrgencyNow},
		{"later today", testNow + 12*secondsPerHour, UrgencyToday},
		{"tomorrow", testNow + 36*secondsPerHour, UrgencyTomorrow},
		{"further out", testNow + 5*secondsPerDay, UrgencyPlanned},
		{"no appointment", 0, UrgencyPlanned},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := urgencyFor(&repositories.BoardMove{OriginWindowStart: tc.windowStart}, testNow)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestDecorateMove_MarksCoverage(t *testing.T) {
	t.Parallel()

	uncovered := decorateMove(&repositories.BoardMove{
		OriginWindowStart: testNow + secondsPerHour,
	}, testNow)
	assert.False(t, uncovered.IsCovered)
	assert.Equal(t, int64(60), uncovered.MinutesToPickup)

	covered := decorateMove(&repositories.BoardMove{
		AssignmentID:      pulid.MustNew("asn_"),
		OriginWindowStart: testNow + secondsPerHour,
	}, testNow)
	assert.True(t, covered.IsCovered)
}

func snapshotWith(
	driver *repositories.BoardDriver,
	w *worker.Worker,
	state *telematics.WorkerHOSState,
) *dispatchcandidateservice.FleetSnapshot {
	return &dispatchcandidateservice.FleetSnapshot{
		Now:     testNow,
		Drivers: []*repositories.BoardDriver{driver},
		WorkersByID: map[pulid.ID]*worker.Worker{
			driver.WorkerID: w,
		},
		HOSByWorker: map[pulid.ID]*telematics.WorkerHOSState{
			driver.WorkerID: state,
		},
		PosByTractor:        map[pulid.ID]*telematics.VehiclePosition{},
		CommitmentsByWorker: map[pulid.ID][]*repositories.WorkerCommitment{},
		TimeOffByWorker:     map[pulid.ID][]*repositories.WorkerTimeOff{},
		WorkloadByWorker:    map[pulid.ID]*repositories.WorkerWorkload{},
	}
}

func availableWorker(id pulid.ID) *worker.Worker {
	return &worker.Worker{
		ID:                   id,
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
		Profile:              &worker.WorkerProfile{DOB: timeutils.YearsAgoUnix(40)},
	}
}

func control() *dispatchcontrol.DispatchControl {
	return &dispatchcontrol.DispatchControl{
		ComplianceEnforcementLevel: dispatchcontrol.ComplianceEnforcementLevelWarning,
	}
}

func TestDecorateDrivers_IdleDriverReadsAsOpen(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID, FirstName: "Ada", LastName: "Reed"}
	snapshot := snapshotWith(driver, availableWorker(workerID), &telematics.WorkerHOSState{
		DutyStatus:       telematics.DutyStatusOffDuty,
		DriveRemainingMs: 11 * 3600 * 1000,
		RecordedAt:       testNow,
	})

	rows := decorateDrivers(snapshot, control(), testNow)

	require.Len(t, rows, 1)
	assert.Equal(t, AvailabilityOpen, rows[0].Availability)
	assert.Equal(t, testNow, rows[0].ProjectedTimeAvailable)
	assert.False(t, rows[0].HOSIsStale)
	assert.Equal(t, int64(11*3600*1000), rows[0].DriveRemainingMs)
}

func TestDecorateDrivers_PTAComesFromTheLastCommitment(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID}
	snapshot := snapshotWith(driver, availableWorker(workerID), nil)
	snapshot.CommitmentsByWorker[workerID] = []*repositories.WorkerCommitment{
		{WindowEnd: testNow + 10*secondsPerHour},
		{WindowEnd: testNow + 30*secondsPerHour},
	}

	rows := decorateDrivers(snapshot, control(), testNow)

	require.Len(t, rows, 1)
	assert.Equal(t, testNow+30*secondsPerHour, rows[0].ProjectedTimeAvailable)
	assert.Equal(t, AvailabilityWorking, rows[0].Availability)
}

func TestDecorateDrivers_DriverWrappingUpReadsAsFinishing(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID}
	snapshot := snapshotWith(driver, availableWorker(workerID), nil)
	snapshot.CommitmentsByWorker[workerID] = []*repositories.WorkerCommitment{
		{WindowEnd: testNow + secondsPerHour},
	}

	rows := decorateDrivers(snapshot, control(), testNow)

	assert.Equal(t, AvailabilityFinishing, rows[0].Availability)
}

// A blocked driver must read as blocked whatever their calendar says: the reason they
// cannot work outranks whether they happen to be free.
func TestDecorateDrivers_BlockOutranksTimeOffAndCommitments(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID}
	w := availableWorker(workerID)
	w.AvailableForDispatch = false

	snapshot := snapshotWith(driver, w, nil)
	snapshot.TimeOffByWorker[workerID] = []*repositories.WorkerTimeOff{
		{StartDate: testNow - secondsPerDay, EndDate: testNow + secondsPerDay},
	}

	rows := decorateDrivers(snapshot, control(), testNow)

	assert.Equal(t, AvailabilityBlocked, rows[0].Availability)
	assert.NotEmpty(t, rows[0].Findings)
}

func TestDecorateDrivers_ActiveTimeOffReadsAsTimeOff(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID}
	snapshot := snapshotWith(driver, availableWorker(workerID), nil)
	snapshot.TimeOffByWorker[workerID] = []*repositories.WorkerTimeOff{
		{StartDate: testNow - secondsPerDay, EndDate: testNow + secondsPerDay},
	}

	rows := decorateDrivers(snapshot, control(), testNow)

	assert.Equal(t, AvailabilityTimeOff, rows[0].Availability)
}

func TestDecorateDrivers_StaleHOSIsFlaggedNotBlocked(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	driver := &repositories.BoardDriver{WorkerID: workerID}
	dc := control()
	dc.EnforceHOSCompliance = true

	snapshot := snapshotWith(driver, availableWorker(workerID), &telematics.WorkerHOSState{
		RecordedAt: testNow - dispatcheligibility.HOSStateMaxAgeSeconds - 1,
	})

	rows := decorateDrivers(snapshot, dc, testNow)

	assert.True(t, rows[0].HOSIsStale)
	assert.Equal(t, AvailabilityOpen, rows[0].Availability,
		"a stale feed is missing information, not a reason to refuse the driver")
}

func TestResolveWindow_FallsBackToTheConfiguredHorizon(t *testing.T) {
	t.Parallel()

	dc := control()
	dc.AutoAssignPlanningHorizonHours = 24

	start, end := resolveWindow(0, 0, dc, testNow)
	assert.Equal(t, testNow, start)
	assert.Equal(t, testNow+24*secondsPerHour, end)

	explicitStart, explicitEnd := resolveWindow(500, 900, dc, testNow)
	assert.Equal(t, int64(500), explicitStart)
	assert.Equal(t, int64(900), explicitEnd)
}

func TestCustomerIDsOf_DeduplicatesAndSkipsEmpty(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("cus_")
	second := pulid.MustNew("cus_")

	ids := customerIDsOf([]*repositories.BoardMove{
		{CustomerID: first},
		{CustomerID: second},
		{CustomerID: first},
		{},
	})

	assert.ElementsMatch(t, []pulid.ID{first, second}, ids)
}

func TestSpanOf_CoversEveryMove(t *testing.T) {
	t.Parallel()

	end := int64(5000)
	start, finish := spanOf([]*repositories.BoardMove{
		{OriginWindowStart: 2000, DestinationWindowStart: 3000},
		{OriginWindowStart: 1000, DestinationWindowStart: 4000, DestinationWindowEnd: &end},
	})

	assert.Equal(t, int64(1000), start)
	assert.Equal(t, int64(5000), finish)
}
