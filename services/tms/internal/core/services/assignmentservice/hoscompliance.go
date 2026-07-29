package assignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) runHOSComplianceChecks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dc *dispatchcontrol.DispatchControl,
	primaryWorkerID pulid.ID,
	secondaryWorkerID *pulid.ID,
	multiErr *errortypes.MultiError,
) error {
	if !dc.EnforceHOSCompliance || s.telematicsRepo == nil || s.integrationRepo == nil {
		return nil
	}

	integrations, err := s.integrationRepo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if !integration.HasEnabledTelematics(integrations) {
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

	now := timeutils.NowUnix()

	checkWorkerHOSClocks(statesByWorker[primaryWorkerID], dc, now, "primaryWorker", multiErr)
	if secondaryWorkerID != nil && !secondaryWorkerID.IsNil() {
		checkWorkerHOSClocks(
			statesByWorker[*secondaryWorkerID],
			dc,
			now,
			"secondaryWorker",
			multiErr,
		)
	}
	return nil
}

func checkWorkerHOSClocks(
	state *telematics.WorkerHOSState,
	dc *dispatchcontrol.DispatchControl,
	now int64,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   state,
		Control: dc,
		Now:     now,
	}).AppendToMultiError(multiErr, prefix)
}
