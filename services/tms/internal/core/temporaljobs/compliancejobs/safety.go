package compliancejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// safetyLapseWindowDays is how far back the sweep looks for points that rolled
// off. A day would be enough if the sweep never missed a night; two gives a
// missed run a chance to catch up without a full rebuild.
const safetyLapseWindowDays = 2

// SafetyRollupSweepResult reports what the sweep repaired.
type SafetyRollupSweepResult struct {
	WorkersChecked int `json:"workersChecked"`
	Failed         int `json:"failed"`
}

// SafetyRollupSweepActivity repairs the roster's safety cache for workers
// whose standing moved without anyone writing anything: points roll off two
// years after an event, and disciplinary actions lapse a year after they are
// issued. Both improve a worker's rating on a date, and nothing else would
// notice.
//
// It deliberately does not walk every worker. Only those with something that
// lapsed inside the window can have changed, which keeps a nightly job over a
// few thousand drivers down to a handful of recomputes.
func (a *Activities) SafetyRollupSweepActivity(
	ctx context.Context,
) (*SafetyRollupSweepResult, error) {
	result := new(SafetyRollupSweepResult)
	now := timeutils.NowUnix()
	since := now - safetyLapseWindowDays*secondsPerDay

	refs, err := a.safetyRepo.ListWorkersWithLapsedPoints(
		ctx,
		&repositories.ListWorkersWithLapsedPointsRequest{
			Since: since,
			Until: now,
			Limit: sweepPageSize,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, ref := range refs {
		result.WorkersChecked++
		tenantInfo := pagination.TenantInfo{
			OrgID: ref.OrganizationID,
			BuID:  ref.BusinessUnitID,
		}
		if _, rErr := a.safety.RefreshRollup(ctx, tenantInfo, ref.WorkerID); rErr != nil {
			result.Failed++
			a.logger.Warn("failed to refresh safety rollup after sweep",
				zap.String("workerId", ref.WorkerID.String()),
				zap.Error(rErr))
		}
		activity.RecordHeartbeat(ctx, result.WorkersChecked)
	}

	return result, nil
}
