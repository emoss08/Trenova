package assignmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const hosStateMaxAgeSeconds = int64(12 * 3600)

func (s *service) runHOSComplianceChecks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dc *dispatchcontrol.DispatchControl,
	primaryWorkerID pulid.ID,
	secondaryWorkerID *pulid.ID,
	multiErr *errortypes.MultiError,
) error {
	if !dc.EnforceHOSCompliance || s.telematicsRepo == nil {
		return nil
	}

	workerIDs := []pulid.ID{primaryWorkerID}
	if secondaryWorkerID != nil && !secondaryWorkerID.IsNil() {
		workerIDs = append(workerIDs, *secondaryWorkerID)
	}

	states, err := s.telematicsRepo.ListWorkerHOSStates(
		ctx,
		&repositories.ListWorkerHOSStatesRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
		},
	)
	if err != nil {
		return err
	}

	statesByWorker := make(map[pulid.ID]*telematics.WorkerHOSState, len(states))
	for _, state := range states {
		statesByWorker[state.WorkerID] = state
	}

	checkWorkerHOSClocks(statesByWorker[primaryWorkerID], dc, "primaryWorker", multiErr)
	if secondaryWorkerID != nil && !secondaryWorkerID.IsNil() {
		checkWorkerHOSClocks(statesByWorker[*secondaryWorkerID], dc, "secondaryWorker", multiErr)
	}
	return nil
}

func checkWorkerHOSClocks(
	state *telematics.WorkerHOSState,
	dc *dispatchcontrol.DispatchControl,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if state == nil {
		return
	}
	if timeutils.NowUnix()-state.RecordedAt > hosStateMaxAgeSeconds {
		return
	}

	errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

	if state.ShiftDrivingViolationMs > 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			"Driver has an active shift driving HOS violation (49 CFR 395.3(a)(3))",
		)
	}
	if state.CycleViolationMs > 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			"Driver has an active cycle HOS violation (49 CFR 395.3(b))",
		)
	}
	if state.DriveRemainingMs <= 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			"Driver has no drive time remaining on the 11-hour clock (49 CFR 395.3(a)(3))",
		)
	}
	if state.ShiftRemainingMs <= 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			"Driver has no on-duty time remaining in the 14-hour window (49 CFR 395.3(a)(2))",
		)
	}
	if state.CycleRemainingMs <= 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			"Driver has no cycle time remaining (49 CFR 395.3(b))",
		)
	}
	if state.DutyStatus == telematics.DutyStatusDriving && state.BreakRemainingMs <= 0 {
		multiErr.WithPrefix(prefix).Add(
			"hos",
			errCode,
			fmt.Sprintf(
				"Driver requires a 30-minute rest break before driving (49 CFR 395.3(a)(3)(ii)); break clock at %d minutes",
				state.BreakRemainingMs/60000,
			),
		)
	}
}
